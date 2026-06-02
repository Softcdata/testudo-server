package cluster

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-operator/pkg/tools"
	"github.com/softcdata/testudo-server/configs"
	platformlicenseapi "github.com/softcdata/testudo-server/internal/apis/platform_license/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	watchutils "github.com/softcdata/testudo-server/internal/utils"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterHandler struct {
	*kube.KubeClient
	Rg                   *route.RouterGroup
	Mw                   []app.HandlerFunc
	ClusterLister        listers.ClusterLister
	DisasterConfigLister listers.DisasterConfigLister
	InstanceLister       listers.DisasterInstanceLister

	GetClusterClient   func(ctx context.Context, clusterName string) (ctrclient.Client, error)
	LicenseService     clusterLicenseService
	LicenseGateEnabled bool
}

const managedVeleroRegistrySecretNamePrefix = "cluster-velero-regcred"

type clusterLicenseService interface {
	CheckClusterCreate(ctx context.Context) (platformlicenseapi.ClusterCreateCheck, error)
}

func NewClusterHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *ClusterHandler {
	var licenseService clusterLicenseService
	if kc != nil && kc.ClusterClient != nil {
		namespace := ""
		caPath := ""
		if configs.Cfg != nil {
			namespace = configs.Cfg.License.Namespace
			caPath = configs.Cfg.License.CAPath
		}
		service := platformlicenseapi.NewService(kc.RuntimeClient(), namespace, caPath)
		service.RuntimeReader = kc.RuntimeReader()
		if bundle := kc.LicenseCABundle(); len(bundle) > 0 {
			service.CABundle = bundle
		}
		licenseService = service
	}
	return &ClusterHandler{
		KubeClient:           kc,
		Rg:                   rg,
		Mw:                   mw,
		ClusterLister:        kc.InformerFactory.Disaster().V1().Clusters().Lister(),
		DisasterConfigLister: kc.InformerFactory.Disaster().V1().DisasterConfigs().Lister(),
		InstanceLister:       kc.InformerFactory.Disaster().V1().DisasterInstances().Lister(),
		LicenseService:       licenseService,
		LicenseGateEnabled:   licenseGateEnabled(),
	}
}

func licenseGateEnabled() bool {
	if configs.Cfg == nil {
		return true
	}
	return configs.Cfg.License.Enabled
}

func managedVeleroRegistrySecretName(clusterName string) string {
	return fmt.Sprintf("%s-%s", managedVeleroRegistrySecretNamePrefix, clusterName)
}

func isManagedVeleroRegistrySecretRef(clusterName string, ref *corev1.LocalObjectReference) bool {
	return ref != nil && ref.Name == managedVeleroRegistrySecretName(clusterName)
}

func normalizeVeleroImageRegistry(registry string) string {
	return strings.Trim(strings.TrimSpace(registry), "/")
}

func validateVeleroInstallWriteRequest(imageRegistry, username, password string, removeCredential bool) error {
	imageRegistry = normalizeVeleroImageRegistry(imageRegistry)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if strings.Contains(imageRegistry, "://") {
		return fmt.Errorf("veleroInstall.imageRegistry must not include a URL scheme")
	}
	if removeCredential && (username != "" || password != "") {
		return fmt.Errorf("veleroInstall.removeCredential and username/password are mutually exclusive")
	}
	if (username == "") != (password == "") {
		return fmt.Errorf("veleroInstall.username and password must be provided together")
	}
	if (username != "" || password != "") && imageRegistry == "" {
		return fmt.Errorf("veleroInstall.imageRegistry is required when username/password are provided")
	}
	return nil
}

func registryAuthServer(imageRegistry string) string {
	imageRegistry = normalizeVeleroImageRegistry(imageRegistry)
	if imageRegistry == "" {
		return ""
	}
	parts := strings.Split(imageRegistry, "/")
	return parts[0]
}

func buildDockerConfigJSON(imageRegistry, username, password string) ([]byte, error) {
	authServer := registryAuthServer(imageRegistry)
	if authServer == "" {
		return nil, fmt.Errorf("veleroInstall.imageRegistry is required")
	}
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	payload := map[string]map[string]map[string]string{
		"auths": {
			authServer: {
				"username": username,
				"password": password,
				"auth":     auth,
			},
		},
	}
	return json.Marshal(payload)
}

type dockerConfigJSONPayload struct {
	Auths map[string]dockerAuthConfig `json:"auths"`
}

type dockerAuthConfig struct {
	Username string `json:"username"`
	Auth     string `json:"auth"`
}

func usernameFromDockerAuthConfig(auth dockerAuthConfig) string {
	if username := strings.TrimSpace(auth.Username); username != "" {
		return username
	}
	rawAuth := strings.TrimSpace(auth.Auth)
	if rawAuth == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(rawAuth)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func usernameFromDockerConfigJSON(dockerConfigJSON []byte, imageRegistry string) string {
	if len(dockerConfigJSON) == 0 {
		return ""
	}
	var payload dockerConfigJSONPayload
	if err := json.Unmarshal(dockerConfigJSON, &payload); err != nil {
		return ""
	}
	if len(payload.Auths) == 0 {
		return ""
	}
	authServer := registryAuthServer(imageRegistry)
	if authServer != "" {
		if auth, ok := payload.Auths[authServer]; ok {
			return usernameFromDockerAuthConfig(auth)
		}
	}
	if len(payload.Auths) == 1 {
		for _, auth := range payload.Auths {
			return usernameFromDockerAuthConfig(auth)
		}
	}
	return ""
}

func (cluster *ClusterHandler) veleroRegistryUsername(c context.Context, item *dapisv1.Cluster) string {
	if cluster == nil || cluster.K8sClient == nil || item == nil || item.Spec.VeleroInstall == nil {
		return ""
	}
	ref := item.Spec.VeleroInstall.RegistryCredentialSecretRef
	if !isManagedVeleroRegistrySecretRef(item.Name, ref) {
		return ""
	}
	secret, err := cluster.K8sClient.CoreV1().Secrets(common.DisasterSystemNamespace).Get(c, ref.Name, matev1.GetOptions{})
	if err != nil || secret.Type != corev1.SecretTypeDockerConfigJson {
		return ""
	}
	return usernameFromDockerConfigJSON(secret.Data[corev1.DockerConfigJsonKey], item.Spec.VeleroInstall.ImageRegistry)
}

func (cluster *ClusterHandler) convertToDisasterClusterDTO(c context.Context, item *dapisv1.Cluster) DisasterClusterDTO {
	dto := ConvertToDisasterClusterDTO(item)
	if dto.Spec.VeleroInstall != nil {
		dto.Spec.VeleroInstall.Username = cluster.veleroRegistryUsername(c, item)
	}
	return dto
}

func (cluster *ClusterHandler) upsertManagedVeleroRegistrySecret(ctx context.Context, namespace, clusterName, imageRegistry, username, password string) (*corev1.LocalObjectReference, bool, error) {
	secretName := managedVeleroRegistrySecretName(clusterName)
	dockerConfigJSON, err := buildDockerConfigJSON(imageRegistry, username, password)
	if err != nil {
		return nil, false, err
	}

	secretsClient := cluster.K8sClient.CoreV1().Secrets(namespace)
	existing, err := secretsClient.Get(ctx, secretName, matev1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, false, err
	}

	created := false
	if errors.IsNotFound(err) {
		existing = &corev1.Secret{
			ObjectMeta: matev1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{},
		}
		created = true
	}

	existing.Type = corev1.SecretTypeDockerConfigJson
	if existing.Data == nil {
		existing.Data = make(map[string][]byte)
	}
	existing.Data[corev1.DockerConfigJsonKey] = dockerConfigJSON

	if created {
		if _, err := secretsClient.Create(ctx, existing, matev1.CreateOptions{}); err != nil {
			return nil, false, err
		}
	} else {
		if _, err := secretsClient.Update(ctx, existing, matev1.UpdateOptions{}); err != nil {
			return nil, false, err
		}
	}

	return &corev1.LocalObjectReference{Name: secretName}, created, nil
}

func (cluster *ClusterHandler) deleteManagedVeleroRegistrySecret(ctx context.Context, namespace, clusterName string) error {
	err := cluster.K8sClient.CoreV1().Secrets(namespace).Delete(ctx, managedVeleroRegistrySecretName(clusterName), matev1.DeleteOptions{})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func (cluster *ClusterHandler) clusters(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// Map simple 'tag' query param to the actual label key
	if tag, ok := qParams.Filters["tag"]; ok {
		delete(qParams.Filters, "tag")
		qParams.Filters[ClusterTagLabel] = tag
	}

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取初步后的数据 (从本地缓存)
	items, err := cluster.ClusterLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 3.0 全量内存模糊过滤 (实现“一律模糊搜索”)
	filteredItems := make([]*dapisv1.Cluster, 0)
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

	// 3.1 内存关键字过滤
	if qParams.Keyword != "" {
		var matched []*dapisv1.Cluster
		keyword := qParams.Keyword
		for _, item := range filteredItems {
			if strings.Contains(item.Name, keyword) {
				matched = append(matched, item)
				continue
			}
			if tag, ok := item.Labels[ClusterTagLabel]; ok && strings.Contains(tag, keyword) {
				matched = append(matched, item)
				continue
			}
		}
		filteredItems = matched
	}

	// 4. 内存排序逻辑
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.Cluster, field string) int {
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
	dtos := make([]DisasterClusterDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = cluster.convertToDisasterClusterDTO(c, item)
	}

	// 6. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterCluster",
		dtos,
		qParams,
		total,
		nil, // 暂无额外链接
		func(item DisasterClusterDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (cluster *ClusterHandler) clusterNames(c context.Context, ctx *app.RequestContext) {
	// Parse options but ignore limit/page as we want all names
	qParams := transport.ParseOptions(c, ctx)

	// Still allow filtering by tag/labels if needed
	if tag, ok := qParams.Filters["tag"]; ok {
		delete(qParams.Filters, "tag")
		qParams.Filters[ClusterTagLabel] = tag
	}

	selector := transport.BuildLabelSelector(qParams)
	items, err := cluster.ClusterLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 全量内存模糊过滤
	matchedItems := make([]*dapisv1.Cluster, 0)
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

	if qParams.Keyword != "" {
		var matched []*dapisv1.Cluster
		keyword := qParams.Keyword
		for _, item := range items {
			if strings.Contains(item.Name, keyword) {
				matched = append(matched, item)
				continue
			}
			if tag, ok := item.Labels[ClusterTagLabel]; ok && strings.Contains(tag, keyword) {
				matched = append(matched, item)
				continue
			}
		}
		items = matched
	}

	dtos := make([]DisasterClusterNameDTO, len(items))
	for i, item := range items {
		dtos[i] = DisasterClusterNameDTO{
			Name:                   item.Name,
			ID:                     string(item.UID),
			NamespaceCount:         item.Status.NamespaceCount,
			ResourceTotalCount:     item.Status.ResourceTotalCount,
			WorkloadNamespaceCount: item.Status.WorkloadNamespaceCount,
			WorkloadTotalCount:     item.Status.WorkloadTotalCount,
			Tag:                    item.Labels[ClusterTagLabel],
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, dtos, nil)
}

func (cluster *ClusterHandler) cluster(c context.Context, ctx *app.RequestContext) {
	item, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, ctx.Param("name"), matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := cluster.convertToDisasterClusterDTO(c, item)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)

}

func (cluster *ClusterHandler) listRestoreClasses(c context.Context, ctx *app.RequestContext) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	targetCli, err := cluster.getClusterClient(c, name)
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	storageClassList := &storagev1.StorageClassList{}
	if err := targetCli.List(c, storageClassList); err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("failed to list storageclasses: %v", err), nil)
		return
	}

	ingressClassList := &networkingv1.IngressClassList{}
	if err := targetCli.List(c, ingressClassList); err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("failed to list ingressclasses: %v", err), nil)
		return
	}

	storageClasses := make([]RestoreClassItemDTO, 0, len(storageClassList.Items))
	for _, item := range storageClassList.Items {
		storageClasses = append(storageClasses, RestoreClassItemDTO{
			Name:      item.Name,
			IsDefault: isStorageClassDefault(item.Annotations),
		})
	}
	sort.SliceStable(storageClasses, func(i, j int) bool {
		return storageClasses[i].Name < storageClasses[j].Name
	})

	ingressClasses := make([]RestoreClassItemDTO, 0, len(ingressClassList.Items))
	for _, item := range ingressClassList.Items {
		ingressClasses = append(ingressClasses, RestoreClassItemDTO{
			Name:      item.Name,
			IsDefault: isIngressClassDefault(item.Annotations),
		})
	}
	sort.SliceStable(ingressClasses, func(i, j int) bool {
		return ingressClasses[i].Name < ingressClasses[j].Name
	})

	transport.WriteSuccess(ctx, consts.StatusOK, RestoreClassListDTO{
		TargetCluster:  name,
		StorageClasses: storageClasses,
		IngressClasses: ingressClasses,
	}, nil)
}

func (cluster *ClusterHandler) protectedNamespaces(c context.Context, ctx *app.RequestContext) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	if _, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, name, matev1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	index, err := common.BuildProtectedNamespaceIndex(cluster.DisasterConfigLister, cluster.InstanceLister)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	qParams := transport.ParseOptions(c, ctx)
	records := index.Records(name)
	if qParams.Keyword != "" {
		keyword := qParams.Keyword
		filtered := make([]common.ProtectedNamespaceRecord, 0, len(records))
		for _, record := range records {
			if strings.Contains(record.Namespace, keyword) {
				filtered = append(filtered, record)
				continue
			}
			for _, owner := range record.Owners {
				if strings.Contains(owner.InstanceName, keyword) ||
					strings.Contains(owner.InstanceNamespace, keyword) ||
					strings.Contains(owner.ConfigName, keyword) {
					filtered = append(filtered, record)
					break
				}
			}
		}
		records = filtered
	}

	items := make([]ClusterProtectedNamespaceDTO, 0, len(records))
	for _, record := range records {
		owners := make([]ClusterProtectedNamespaceOwnerDTO, 0, len(record.Owners))
		for _, owner := range record.Owners {
			owners = append(owners, ClusterProtectedNamespaceOwnerDTO{
				InstanceName:      owner.InstanceName,
				InstanceNamespace: owner.InstanceNamespace,
				ConfigName:        owner.ConfigName,
			})
		}
		items = append(items, ClusterProtectedNamespaceDTO{
			Namespace: record.Namespace,
			Cluster:   record.Cluster,
			Owners:    owners,
		})
	}

	if qParams.Sort == "" {
		qParams.Sort = "namespace"
		qParams.Order = "asc"
	}
	sortedItems := transport.Sort(items, qParams, func(a, b ClusterProtectedNamespaceDTO, field string) int {
		switch field {
		case "cluster":
			if clusterCmp := strings.Compare(a.Cluster, b.Cluster); clusterCmp != 0 {
				return clusterCmp
			}
			return strings.Compare(a.Namespace, b.Namespace)
		case "namespace":
			fallthrough
		default:
			if namespaceCmp := strings.Compare(a.Namespace, b.Namespace); namespaceCmp != 0 {
				return namespaceCmp
			}
			return strings.Compare(a.Cluster, b.Cluster)
		}
	})

	if len(ctx.Query("limit")) == 0 {
		qParams.Limit = len(sortedItems)
		if qParams.Limit == 0 {
			qParams.Limit = 1
		}
	}
	pagedItems, total := transport.Paginate(sortedItems, qParams)
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"clusterProtectedNamespace",
		pagedItems,
		qParams,
		total,
		nil,
		nil,
	)
	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (cluster *ClusterHandler) getClusterClient(ctx context.Context, clusterName string) (ctrclient.Client, error) {
	if cluster.GetClusterClient != nil {
		return cluster.GetClusterClient(ctx, clusterName)
	}
	if cluster.KubeClient == nil || cluster.KubeClient.ClusterClient == nil {
		return nil, fmt.Errorf("cluster client is not initialized")
	}
	return cluster.KubeClient.GetKubeClient(ctx, cluster.KubeClient.RuntimeClient(), cluster.KubeClient.Scheme(), clusterName)
}

func (cluster *ClusterHandler) createCluster(c context.Context, ctx *app.RequestContext) {
	var req CreateDisasterClusterRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if req.VeleroInstall != nil {
		if err := validateVeleroInstallWriteRequest(req.VeleroInstall.ImageRegistry, req.VeleroInstall.Username, req.VeleroInstall.Password, req.VeleroInstall.RemoveCredential); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		req.VeleroInstall.ImageRegistry = normalizeVeleroImageRegistry(req.VeleroInstall.ImageRegistry)
	}
	normalizedImageSources, err := normalizeClusterImageSources(req.ImageSources)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	requestEndpoint, err := resolveCreateClusterEffectiveEndpoint(req)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	items, err := cluster.ClusterLister.List(labels.Everything())
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	if err := findClusterEndpointConflict(items, req.Name, requestEndpoint); err != nil {
		if conflictErr, ok := err.(*clusterEndpointConflictError); ok {
			transport.WriteError(ctx, transport.CodeConflict, conflictErr.Error(), conflictErr.meta)
			return
		}
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	if !cluster.checkClusterCreateLicense(c, ctx) {
		return
	}

	labels := make(map[string]string)
	if req.Tag != "" {
		labels[ClusterTagLabel] = req.Tag
	}

	annotations := make(map[string]string)
	if req.Description != "" {
		annotations[ClusterDescriptionAnnotation] = req.Description
	}

	body := dapisv1.Cluster{
		ObjectMeta: matev1.ObjectMeta{
			Name:        req.Name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: req.ToCRD(),
	}
	body.Spec.ImageSources = normalizedImageSources

	managedSecretCreated := false
	if req.VeleroInstall != nil && body.Spec.VeleroInstall == nil && req.VeleroInstall.ImageRegistry != "" {
		body.Spec.VeleroInstall = &dapisv1.VeleroInstallSpec{ImageRegistry: req.VeleroInstall.ImageRegistry}
	}
	if req.VeleroInstall != nil && strings.TrimSpace(req.VeleroInstall.Username) != "" && strings.TrimSpace(req.VeleroInstall.Password) != "" {
		secretRef, created, err := cluster.upsertManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, req.Name, req.VeleroInstall.ImageRegistry, req.VeleroInstall.Username, req.VeleroInstall.Password)
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		if body.Spec.VeleroInstall == nil {
			body.Spec.VeleroInstall = &dapisv1.VeleroInstallSpec{}
		}
		body.Spec.VeleroInstall.ImageRegistry = req.VeleroInstall.ImageRegistry
		body.Spec.VeleroInstall.RegistryCredentialSecretRef = secretRef
		managedSecretCreated = created
	}
	if body.Spec.VeleroInstall != nil && body.Spec.VeleroInstall.ImageRegistry == "" && body.Spec.VeleroInstall.RegistryCredentialSecretRef == nil {
		body.Spec.VeleroInstall = nil
	}

	// Inject trace_id annotation for operator correlation
	transport.SetTraceAnnotation(&body.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if body.Annotations == nil {
			body.Annotations = make(map[string]string)
		}
		body.Annotations["testudo.softcdata.com/user"] = user.(string)
	}

	rc, err := cluster.DisasterClient.DisasterV1().Clusters().Create(c, &body, matev1.CreateOptions{})
	if err != nil {
		if managedSecretCreated {
			_ = cluster.deleteManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, req.Name)
		}
		if errors.IsAlreadyExists(err) {
			transport.WriteError(ctx, transport.CodeConflict, err.Error(), nil)
			return
		}
		if errors.IsInvalid(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	dto := cluster.convertToDisasterClusterDTO(c, rc)
	transport.WriteSuccess(ctx, consts.StatusCreated, dto, nil)
}

func (cluster *ClusterHandler) checkClusterCreateLicense(c context.Context, ctx *app.RequestContext) bool {
	if !cluster.LicenseGateEnabled {
		return true
	}
	if cluster.LicenseService == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyLicenseServiceNotReady, nil, nil)
		return false
	}
	check, err := cluster.LicenseService.CheckClusterCreate(c)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return false
	}
	if check.Allowed {
		return true
	}
	message := check.Message
	if strings.TrimSpace(message) == "" {
		message = "cluster quota exceeded"
	}
	transport.WriteError(ctx, transport.CodeForbidden, message, check.Meta)
	return false
}

func (cluster *ClusterHandler) deleteCluster(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	uninstallVeleroRaw := strings.TrimSpace(string(ctx.Query("uninstallVelero")))
	var uninstallVeleroOpt *bool
	if uninstallVeleroRaw != "" {
		uninstallVelero, err := strconv.ParseBool(uninstallVeleroRaw)
		if err != nil {
			transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationInvalidBoolQuery, map[string]any{"name": "uninstallVelero"}, nil)
			return
		}
		uninstallVeleroOpt = &uninstallVelero
	}

	// Best-effort annotation before delete
	existing, _ := cluster.DisasterClient.DisasterV1().Clusters().Get(c, name, matev1.GetOptions{})
	if existing != nil {
		transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
		if user, ok := ctx.Get("userName"); ok {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			existing.Annotations["testudo.softcdata.com/user"] = user.(string)
		}
		if uninstallVeleroOpt != nil {
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			if *uninstallVeleroOpt {
				existing.Annotations[metadata.AnnotationUninstallVelero] = "true"
			} else {
				delete(existing.Annotations, metadata.AnnotationUninstallVelero)
			}
		}
		if _, err := cluster.DisasterClient.DisasterV1().Clusters().Update(c, existing, matev1.UpdateOptions{}); err != nil {
			// uninstallVelero 是删除行为的关键开关，写入失败时必须终止删除，避免出现“集群已删但 Velero 未清理”的假成功。
			if uninstallVeleroOpt != nil {
				transport.WriteError(ctx, transport.CodeInternalServerError, fmt.Sprintf("failed to persist uninstallVelero before delete: %v", err), nil)
				return
			}
		}
	}
	err := cluster.DisasterClient.DisasterV1().Clusters().Delete(c, name, matev1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	if existing != nil && existing.Spec.VeleroInstall != nil && isManagedVeleroRegistrySecretRef(existing.Name, existing.Spec.VeleroInstall.RegistryCredentialSecretRef) {
		_ = cluster.deleteManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, existing.Name)
	}
	transport.WriteSuccess(ctx, consts.StatusOK, utils.H{"name": name}, nil)
}

func (cluster *ClusterHandler) validateCluster(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	item, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, name, matev1.GetOptions{})
	if errors.IsNotFound(err) {
		transport.WriteSuccess(ctx, consts.StatusOK, false, nil)
		return
	}
	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, nil)
		return
	}
	if item.Status.Status != dapisv1.ClusterStatusReady {
		transport.WriteSuccess(ctx, consts.StatusOK, false, nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, true, nil)

}

// watchClusters 使用通用工具监听所有 Cluster 资源变化
func (cluster *ClusterHandler) watchClusters(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().Clusters().Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.Cluster); ok {
			return cluster.convertToDisasterClusterDTO(c, item)
		}
		return nil
	})
}

// watchCluster 监听指定的 Cluster 资源变化
func (cluster *ClusterHandler) watchCluster(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	// 使用 FieldSelector 只监听指定的资源
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().Clusters().Watch(ctx, matev1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc, func(obj interface{}) interface{} {
		if item, ok := obj.(*dapisv1.Cluster); ok {
			return cluster.convertToDisasterClusterDTO(c, item)
		}
		return nil
	})
}

func (cluster *ClusterHandler) validateKubeConfig(c context.Context, ctx *app.RequestContext) {
	//验证kubeconfig是否能连接上集群
	var req ValidateKubeConfigRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]interface{}{"error": err.Error()})
		return
	}

	var clientConfig *rest.Config
	var err error

	if len(req.KubeConfig) > 0 {
		clientConfig, err = tools.GetRestConfig(req.KubeConfig)
	} else if req.Token != "" && req.Endpoint != "" {
		// Decode token if it's Base64 encoded (JWT starts with "eyJ")
		token := req.Token
		if !strings.HasPrefix(token, "eyJ") {
			if decoded, decErr := base64.StdEncoding.DecodeString(token); decErr == nil {
				token = string(decoded)
			}
		}
		clientConfig, err = tools.GetRestConfigFromToken(req.Endpoint, token)
	} else {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]interface{}{"error": "neither kubeconfig nor token/endpoint provided"})
		return
	}

	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]interface{}{"error": err.Error()})
		return
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]interface{}{"error": err.Error()})
		return
	}
	_, err = clientset.ServerVersion()
	if err != nil {
		transport.WriteSuccess(ctx, consts.StatusOK, false, map[string]interface{}{"error": err.Error()})
		return
	}

	transport.WriteSuccess(ctx, consts.StatusOK, true, nil)
}

func (cluster *ClusterHandler) refreshNamespaces(c context.Context, ctx *app.RequestContext) {
	name := strings.TrimSpace(ctx.Param("name"))
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	var req RefreshNamespacesRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}
	req.Type = strings.TrimSpace(req.Type)
	if !metadata.IsValidClusterStatsRefreshType(req.Type) {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationUnsupportedRefreshType, map[string]any{"type": req.Type}, nil)
		return
	}

	var updated *dapisv1.Cluster
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, name, matev1.GetOptions{})
		if err != nil {
			return err
		}
		if latest.Annotations == nil {
			latest.Annotations = make(map[string]string)
		}
		latest.Annotations[metadata.AnnotationRefreshClusterStats] = req.Type

		updated, err = cluster.DisasterClient.DisasterV1().Clusters().Update(c, latest, matev1.UpdateOptions{})
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

	transport.WriteSuccess(ctx, consts.StatusAccepted, RefreshNamespacesAcceptedDTO{
		Cluster: cluster.convertToDisasterClusterDTO(c, updated),
		Type:    req.Type,
	}, nil)
}

func (cluster *ClusterHandler) patchCluster(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	var req PatchDisasterClusterRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Get existing cluster
	existing, err := cluster.DisasterClient.DisasterV1().Clusters().Get(c, name, matev1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	updated := false
	oldManagedSecret := existing.Spec.VeleroInstall != nil && isManagedVeleroRegistrySecretRef(existing.Name, existing.Spec.VeleroInstall.RegistryCredentialSecretRef)
	if req.Token != nil {
		existing.Spec.Token = *req.Token
		updated = true
	}
	if req.Tag != nil {
		if existing.Labels == nil {
			existing.Labels = make(map[string]string)
		}
		if *req.Tag == "" {
			delete(existing.Labels, ClusterTagLabel)
		} else {
			existing.Labels[ClusterTagLabel] = *req.Tag
		}
		updated = true
	}

	if req.Description != nil {
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string)
		}
		if *req.Description == "" {
			delete(existing.Annotations, ClusterDescriptionAnnotation)
		} else {
			existing.Annotations[ClusterDescriptionAnnotation] = *req.Description
		}
		updated = true
	}
	if req.ImageSources != nil {
		imageSources, err := normalizeClusterImageSources(*req.ImageSources)
		if err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		existing.Spec.ImageSources = imageSources
		updated = true
	}
	if req.VeleroInstall != nil {
		effectiveImageRegistry := ""
		if existing.Spec.VeleroInstall != nil {
			effectiveImageRegistry = existing.Spec.VeleroInstall.ImageRegistry
		}
		imageRegistryProvided := req.VeleroInstall.ImageRegistry != nil
		if req.VeleroInstall.ImageRegistry != nil {
			effectiveImageRegistry = normalizeVeleroImageRegistry(*req.VeleroInstall.ImageRegistry)
		}

		username := ""
		if req.VeleroInstall.Username != nil {
			username = *req.VeleroInstall.Username
		}
		password := ""
		if req.VeleroInstall.Password != nil {
			password = *req.VeleroInstall.Password
		}
		removeCredential := req.VeleroInstall.RemoveCredential != nil && *req.VeleroInstall.RemoveCredential
		if err := validateVeleroInstallWriteRequest(effectiveImageRegistry, username, password, removeCredential); err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}

		clearVeleroInstall := imageRegistryProvided && effectiveImageRegistry == ""
		clearCredential := removeCredential || (req.VeleroInstall.Username != nil && strings.TrimSpace(username) == "")

		if clearVeleroInstall {
			if oldManagedSecret {
				if err := cluster.deleteManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
					transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
					return
				}
			}
			existing.Spec.VeleroInstall = nil
			updated = true
		} else if imageRegistryProvided {
			if existing.Spec.VeleroInstall == nil && effectiveImageRegistry != "" {
				existing.Spec.VeleroInstall = &dapisv1.VeleroInstallSpec{}
			}
			if existing.Spec.VeleroInstall != nil {
				existing.Spec.VeleroInstall.ImageRegistry = effectiveImageRegistry
			}
			updated = true
		}

		if strings.TrimSpace(username) != "" || strings.TrimSpace(password) != "" {
			secretRef, _, err := cluster.upsertManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, existing.Name, effectiveImageRegistry, username, password)
			if err != nil {
				transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
				return
			}
			if existing.Spec.VeleroInstall == nil {
				existing.Spec.VeleroInstall = &dapisv1.VeleroInstallSpec{}
			}
			existing.Spec.VeleroInstall.ImageRegistry = effectiveImageRegistry
			existing.Spec.VeleroInstall.RegistryCredentialSecretRef = secretRef
			updated = true
		}

		if !clearVeleroInstall && clearCredential {
			if oldManagedSecret {
				if err := cluster.deleteManagedVeleroRegistrySecret(c, common.DisasterSystemNamespace, existing.Name); err != nil {
					transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
					return
				}
			}
			if existing.Spec.VeleroInstall == nil {
				existing.Spec.VeleroInstall = &dapisv1.VeleroInstallSpec{}
			}
			existing.Spec.VeleroInstall.RegistryCredentialSecretRef = nil
			updated = true
		}

		if existing.Spec.VeleroInstall != nil && existing.Spec.VeleroInstall.ImageRegistry == "" && existing.Spec.VeleroInstall.RegistryCredentialSecretRef == nil {
			existing.Spec.VeleroInstall = nil
		}
	}

	if !updated {
		// No changes
		dto := cluster.convertToDisasterClusterDTO(c, existing)
		transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
		return
	}

	// Update
	transport.SetTraceAnnotation(&existing.ObjectMeta, ctx, metadata.AnnotationTraceID)
	if user, ok := ctx.Get("userName"); ok {
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string)
		}
		existing.Annotations["testudo.softcdata.com/user"] = user.(string)
	}
	rc, err := cluster.DisasterClient.DisasterV1().Clusters().Update(c, existing, matev1.UpdateOptions{})
	if err != nil {
		if errors.IsInvalid(err) {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	dto := cluster.convertToDisasterClusterDTO(c, rc)
	transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func normalizeClusterImageSources(items []ImageSourceDTO) ([]dapisv1.ImageSource, error) {
	if len(items) == 0 {
		return nil, nil
	}
	result := make([]dapisv1.ImageSource, 0, len(items))
	seenNames := make(map[string]struct{}, len(items))
	for i := range items {
		name := strings.TrimSpace(items[i].Name)
		registry := strings.TrimSpace(items[i].Registry)
		if name == "" {
			return nil, fmt.Errorf("imageSources[%d].name is required", i)
		}
		if registry == "" {
			return nil, fmt.Errorf("imageSources[%d].registry is required", i)
		}
		if _, exists := seenNames[name]; exists {
			return nil, fmt.Errorf("imageSources[%d].name %q is duplicated", i, name)
		}
		seenNames[name] = struct{}{}
		result = append(result, dapisv1.ImageSource{
			Name:     name,
			Registry: registry,
		})
	}
	return result, nil
}
