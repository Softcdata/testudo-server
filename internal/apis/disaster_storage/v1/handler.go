package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/service/verifier"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/util/retry"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	corev1 "k8s.io/api/core/v1"
)

type StorageHandler struct {
	*kube.KubeClient
	Rg                      *route.RouterGroup
	Mw                      []app.HandlerFunc
	StorageRepositoryLister listers.StorageRepositoryLister
	BSLVerifier             verifier.BSLVerifier
}

func NewStorageHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *StorageHandler {
	return &StorageHandler{
		KubeClient:              kc,
		Rg:                      rg,
		Mw:                      mw,
		StorageRepositoryLister: kc.InformerFactory.Disaster().V1().StorageRepositories().Lister(),
		BSLVerifier:             verifier.NewBSLVerifier(),
	}
}

type Storage struct {
	Name          string           `json:"name" yaml:"name"`
	Provisioner   string           `json:"provisioner" yaml:"provisioner"`
	Endpoint      string           `json:"endpoint" yaml:"endpoint"`
	Region        string           `json:"region" yaml:"region"`
	Bucket        string           `json:"bucket" yaml:"bucket"`
	AccessKey     string           `json:"accessKey" yaml:"accessKey"`
	SecretKey     string           `json:"secretKey" yaml:"secretKey"`
	LastCheckTime common.LocalTime `json:"lastCheckTime" yaml:"lastCheckTime"`
}

const managedCASecretNamePrefix = "storage-ca"

func managedCASecretName(storageName string) string {
	return fmt.Sprintf("%s-%s", managedCASecretNamePrefix, storageName)
}

func isManagedCASecretRef(storageName string, ref *corev1.LocalObjectReference) bool {
	return ref != nil && ref.Name == managedCASecretName(storageName)
}

func validateAddressingStyle(style dapisv1.StorageRepositoryAddressingStyle) error {
	if style == "" ||
		style == dapisv1.StorageRepositoryAddressingStylePathStyle ||
		style == dapisv1.StorageRepositoryAddressingStyleVirtualHostedStyle {
		return nil
	}
	return fmt.Errorf("addressingStyle must be PathStyle or VirtualHostedStyle")
}

func validateCAWriteRequest(caBundle string, caSecretRef *corev1.LocalObjectReference, clearCA bool) error {
	modes := 0
	if caBundle != "" {
		modes++
	}
	if caSecretRef != nil {
		if caSecretRef.Name == "" {
			return fmt.Errorf("caSecretRef.name is required")
		}
		modes++
	}
	if clearCA {
		modes++
	}
	if modes > 1 {
		return fmt.Errorf("caBundle, caSecretRef and clearCa are mutually exclusive")
	}
	return nil
}

func (s *StorageHandler) storages(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步后的数据
	items, err := s.StorageRepositoryLister.StorageRepositories(common.DisasterSystemNamespace).List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤
	filteredItems := make([]*dapisv1.StorageRepository, 0)
	for _, item := range items {
		match := true
		for k, v := range qParams.Filters {
			actual := item.Labels[k]
			if !transport.MatchFuzzy(actual, v) {
				match = false
				break
			}
		}
		if match {
			filteredItems = append(filteredItems, item)
		}
	}

	// 4. 内存排序逻辑
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.StorageRepository, field string) int {
		switch field {
		case "name":
			return strings.Compare(a.Name, b.Name)
		case "creationTimestamp":
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			return 0
		default:
			return 0
		}
	})

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// Convert to DTOs
	dtos := make([]DisasterStorageDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToDisasterStorageDTO(item)
	}

	// 6. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"storageRepository",
		dtos,
		qParams,
		total,
		nil,
		func(item DisasterStorageDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (s *StorageHandler) storageNames(c context.Context, ctx *app.RequestContext) {
	qParams := transport.ParseOptions(c, ctx)
	selector := transport.BuildLabelSelector(qParams)
	items, err := s.StorageRepositoryLister.StorageRepositories(common.DisasterSystemNamespace).List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 内存模糊过滤
	matchedItems := make([]*dapisv1.StorageRepository, 0)
	for _, item := range items {
		match := true
		for k, v := range qParams.Filters {
			actual := item.Labels[k]
			if !transport.MatchFuzzy(actual, v) {
				match = false
				break
			}
		}
		if match {
			matchedItems = append(matchedItems, item)
		}
	}
	items = matchedItems

	dtos := make([]DisasterStorageNameDTO, len(items))
	for i, item := range items {
		dtos[i] = DisasterStorageNameDTO{
			Name:   item.Name,
			ID:     string(item.UID),
			Status: item.Status.Status,
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, dtos, nil)
}

func (s *StorageHandler) storage(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	item, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterStorageDTO(item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (s *StorageHandler) createStorage(c context.Context, ctx *app.RequestContext) {
	var req CreateDisasterStorageRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateAddressingStyle(req.AddressingStyle); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateCAWriteRequest(req.CABundle, req.CASecretRef, false); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Validate endpoint format: must start with http:// or https://
	if !strings.HasPrefix(req.Endpoint, "http://") && !strings.HasPrefix(req.Endpoint, "https://") {
		transport.WriteError(ctx, transport.CodeBadRequest, "endpoint must start with http:// or https://", nil)
		return
	}

	body := dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: common.DisasterSystemNamespace,
		},
		Spec: req.ToCRD(),
	}
	managedSecretCreated := false
	if req.CASecretRef != nil {
		if err := s.ensureReferencedCASecretExists(c, common.DisasterSystemNamespace, req.CASecretRef); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
	}
	if req.CABundle != "" {
		caSecretRef, created, err := s.upsertManagedCASecret(c, common.DisasterSystemNamespace, req.Name, req.CABundle)
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		body.Spec.CASecretRef = caSecretRef
		managedSecretCreated = created
	}

	// Inject trace_id annotation for operator correlation
	// Inject trace_id annotation for operator correlation
	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if body.Annotations == nil {
			body.Annotations = make(map[string]string)
		}
		body.Annotations["testudo.softcdata.com/user"] = user.(string)
	}

	rc, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Create(c, &body, metav1.CreateOptions{})
	if err != nil {
		if managedSecretCreated {
			_ = s.deleteManagedCASecret(c, common.DisasterSystemNamespace, req.Name)
		}
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterStorageDTO(rc)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (s *StorageHandler) updateStorage(c context.Context, ctx *app.RequestContext) {
	var req UpdateDisasterStorageRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if req.AddressingStyle != nil {
		if err := validateAddressingStyle(*req.AddressingStyle); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
	}
	if err := validateCAWriteRequest(valueOrEmpty(req.CABundle), req.CASecretRef, boolOrFalse(req.ClearCA)); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Validate endpoint format if provided: must start with http:// or https://
	if req.Endpoint != "" && !strings.HasPrefix(req.Endpoint, "http://") && !strings.HasPrefix(req.Endpoint, "https://") {
		transport.WriteError(ctx, transport.CodeBadRequest, "endpoint must start with http:// or https://", nil)
		return
	}

	var result *dapisv1.StorageRepository
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		existing, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, req.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if req.CASecretRef != nil {
			if err := s.ensureReferencedCASecretExists(c, common.DisasterSystemNamespace, req.CASecretRef); err != nil {
				return err
			}
		}
		oldManagedSecret := isManagedCASecretRef(existing.Name, existing.Spec.CASecretRef)

		// Update Spec
		req.MergeToCRD(&existing.Spec)
		if req.CABundle != nil {
			caSecretRef, _, err := s.upsertManagedCASecret(c, common.DisasterSystemNamespace, existing.Name, *req.CABundle)
			if err != nil {
				return err
			}
			existing.Spec.CASecretRef = caSecretRef
		}
		if req.CASecretRef != nil {
			if oldManagedSecret && !isManagedCASecretRef(existing.Name, req.CASecretRef) {
				if err := s.deleteManagedCASecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
					return err
				}
			}
			existing.Spec.CASecretRef = req.CASecretRef
		}
		if req.ClearCA != nil && *req.ClearCA {
			if oldManagedSecret {
				if err := s.deleteManagedCASecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
					return err
				}
			}
			existing.Spec.CASecretRef = nil
		}

		if existing.Labels == nil {
			existing.Labels = make(map[string]string)
		}
		if existing.Labels[common.DisasterSystemLabel] == "" {
			existing.Labels[common.DisasterSystemLabel] = common.DisasterSystemLabelValue
		}

		// Update trace-id annotation for this write operation
		// Update trace-id annotation for this write operation
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}

		result, err = s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Update(c, existing, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		if errors.IsConflict(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := ConvertToDisasterStorageDTO(result)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (s *StorageHandler) deleteStorage(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")

	// Best-effort annotate before delete for correlation
	existing, _ := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, name, metav1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}
		_, _ = s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Update(c, existing, metav1.UpdateOptions{})
		if isManagedCASecretRef(existing.Name, existing.Spec.CASecretRef) {
			_ = s.deleteManagedCASecret(c, common.DisasterSystemNamespace, existing.Name)
		}
	}
	err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Delete(c, name, metav1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
}

func (s *StorageHandler) patchStorage(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	var req PatchStorageRepositoryRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if req.AddressingStyle != nil {
		if err := validateAddressingStyle(*req.AddressingStyle); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
	}
	if err := validateCAWriteRequest(valueOrEmpty(req.CABundle), req.CASecretRef, boolOrFalse(req.ClearCA)); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 1. Get existing resource
	existing, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	updated := false
	oldManagedSecret := isManagedCASecretRef(existing.Name, existing.Spec.CASecretRef)
	// 2. Update allowed fields only
	if req.AccessKey != nil {
		existing.Spec.AccessKey = *req.AccessKey
		updated = true
	}
	if req.SecretKey != nil {
		existing.Spec.SecretKey = *req.SecretKey
		updated = true
	}
	if req.Bucket != nil {
		existing.Spec.Bucket = *req.Bucket
		updated = true
	}
	if req.Region != nil {
		existing.Spec.Region = *req.Region
		updated = true
	}
	if req.AddressingStyle != nil {
		existing.Spec.AddressingStyle = normalizeAddressingStyle(*req.AddressingStyle)
		updated = true
	}
	if req.CABundle != nil {
		caSecretRef, _, err := s.upsertManagedCASecret(c, common.DisasterSystemNamespace, existing.Name, *req.CABundle)
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		existing.Spec.CASecretRef = caSecretRef
		updated = true
	}
	if req.CASecretRef != nil {
		if err := s.ensureReferencedCASecretExists(c, common.DisasterSystemNamespace, req.CASecretRef); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		if oldManagedSecret && !isManagedCASecretRef(existing.Name, req.CASecretRef) {
			if err := s.deleteManagedCASecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
				transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
				return
			}
		}
		existing.Spec.CASecretRef = req.CASecretRef
		updated = true
	}
	if req.ClearCA != nil && *req.ClearCA {
		if oldManagedSecret {
			if err := s.deleteManagedCASecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
				transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
				return
			}
		}
		existing.Spec.CASecretRef = nil
		updated = true
	}

	if !updated {
		dto := ConvertToDisasterStorageDTO(existing)
		transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
		return
	}

	// 3. Save updates
	// 3. Save updates
	transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string)
		}
		existing.Annotations["testudo.softcdata.com/user"] = user.(string)
	}
	rc, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Update(c, existing, metav1.UpdateOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := ConvertToDisasterStorageDTO(rc)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (s *StorageHandler) watchStorages(c context.Context, ctx *app.RequestContext) {

	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Watch(ctx, metav1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.StorageRepository); ok {
			return ConvertToDisasterStorageDTO(item)
		}
		return nil
	})
}

func (s *StorageHandler) validateS3Connection(c context.Context, ctx *app.RequestContext) {
	var req ValidateS3ConnectionRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateAddressingStyle(req.AddressingStyle); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if err := validateCAWriteRequest(req.CABundle, req.CASecretRef, false); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Configure AWS session
	region := req.Region
	if region == "" {
		region = "us-east-1"
	}
	caBundle, err := s.resolveRequestCABundle(c, req.CABundle, req.CASecretRef)
	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]string{"error": err.Error()})
		return
	}
	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(req.AccessKey, req.SecretKey, "")),
	}
	httpClient, err := buildValidationHTTPClient(caBundle)
	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]string{"error": err.Error()})
		return
	}
	if httpClient != nil {
		loadOptions = append(loadOptions, config.WithHTTPClient(httpClient))
	}

	cfg, err := config.LoadDefaultConfig(c, loadOptions...)

	if err != nil {
		// Session creation error
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]string{"error": fmt.Sprintf("Failed to load aws config: %v", err)})
		return
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(req.Endpoint)
		o.UsePathStyle = normalizeAddressingStyle(req.AddressingStyle) != dapisv1.StorageRepositoryAddressingStyleVirtualHostedStyle
	})

	// If bucket is provided, check existence and permissions
	if req.Bucket != "" {
		_, err = client.HeadBucket(c, &s3.HeadBucketInput{
			Bucket: aws.String(req.Bucket),
		})
	} else {
		// If no bucket provided, check credentials by listing buckets
		_, err = client.ListBuckets(c, &s3.ListBucketsInput{})
	}

	if err != nil {
		// Validation failed
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]string{"error": err.Error()})
		return
	}

	// Validation success
	transport.WriteSuccess(ctx, consts.StatusOK, true, nil)
}
func (s *StorageHandler) validateStorage(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	item, err := s.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(c, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	if item.Status.Status != dapisv1.StorageRepositoryStatusAvailable {
		transport.WriteSuccess(ctx, consts.StatusOK, false, nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, true, nil)
}

func (s *StorageHandler) validateBSLConnectivity(c context.Context, ctx *app.RequestContext) {
	var req ValidateConnectivityRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 1. Get Kube Client for target cluster
	cli, err := s.KubeClient.GetKubeClient(c, s.KubeClient.RuntimeClient(), s.KubeClient.Scheme(), req.ClusterName)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("Failed to get client for cluster %s: %v", req.ClusterName, err), nil)
		return
	}

	// 2. Verify BSL
	valid, message, err := s.BSLVerifier.VerifyBSL(c, cli, s.KubeClient.RuntimeClient(), req.StorageName, req.ClusterName)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	if !valid {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]string{"error": message})
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, true, map[string]string{"message": message})
}

func (s *StorageHandler) upsertManagedCASecret(ctx context.Context, namespace, storageName, caBundle string) (*corev1.LocalObjectReference, bool, error) {
	secretsClient := s.K8sClient.CoreV1().Secrets(namespace)
	secretName := managedCASecretName(storageName)
	secret, err := secretsClient.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
					Labels: map[string]string{
						"testudo.softcdata.com/storage-repository": storageName,
						"testudo.softcdata.com/managed-by":         "disaster-server",
					},
				},
				Data: map[string][]byte{
					dapisv1.StorageRepositoryCASecretKey: []byte(caBundle),
				},
			}
			if _, err := secretsClient.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
				return nil, false, err
			}
			return &corev1.LocalObjectReference{Name: secretName}, true, nil
		}
		return nil, false, err
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[dapisv1.StorageRepositoryCASecretKey] = []byte(caBundle)
	if secret.Labels == nil {
		secret.Labels = make(map[string]string)
	}
	secret.Labels["testudo.softcdata.com/storage-repository"] = storageName
	secret.Labels["testudo.softcdata.com/managed-by"] = "disaster-server"
	if _, err := secretsClient.Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return nil, false, err
	}
	return &corev1.LocalObjectReference{Name: secretName}, false, nil
}

func (s *StorageHandler) deleteManagedCASecret(ctx context.Context, namespace, storageName string) error {
	err := s.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, managedCASecretName(storageName), metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *StorageHandler) ensureReferencedCASecretExists(ctx context.Context, namespace string, ref *corev1.LocalObjectReference) error {
	if ref == nil || ref.Name == "" {
		return fmt.Errorf("caSecretRef.name is required")
	}
	_, err := s.K8sClient.CoreV1().Secrets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("CA secret %s/%s not found", namespace, ref.Name)
		}
		return err
	}
	return nil
}

func (s *StorageHandler) resolveRequestCABundle(ctx context.Context, inline string, ref *corev1.LocalObjectReference) ([]byte, error) {
	if inline != "" {
		return []byte(inline), nil
	}
	if ref == nil || ref.Name == "" {
		return nil, nil
	}
	secret, err := s.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	caBundle, ok := secret.Data[dapisv1.StorageRepositoryCASecretKey]
	if !ok || len(caBundle) == 0 {
		return nil, fmt.Errorf("secret %s does not contain %s", ref.Name, dapisv1.StorageRepositoryCASecretKey)
	}
	return caBundle, nil
}

func buildValidationHTTPClient(caBundle []byte) (*http.Client, error) {
	if len(caBundle) == 0 {
		return nil, nil
	}

	rootCAs, err := x509.SystemCertPool()
	if err != nil || rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}
	if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
		return nil, fmt.Errorf("failed to append custom CA bundle")
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.RootCAs = rootCAs

	return &http.Client{Transport: transport}, nil
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func boolOrFalse(v *bool) bool {
	return v != nil && *v
}
