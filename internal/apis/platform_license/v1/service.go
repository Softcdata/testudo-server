package platformlicenseapi

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	platformlicense "github.com/softcdata/testudo-operator/pkg/license"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	StatusSourceConfigMap      = "statusConfigMap"
	StatusSourceLiveEvaluation = "liveEvaluation"
)

type Service struct {
	RuntimeClient client.Client
	RuntimeReader client.Reader
	Namespace     string
	CAPath        string
	CABundle      []byte
	Verifier      *platformlicense.Verifier
	Now           func() time.Time
}

func NewService(runtimeClient client.Client, namespace, caPath string) *Service {
	return &Service{
		RuntimeClient: runtimeClient,
		Namespace:     strings.TrimSpace(namespace),
		CAPath:        strings.TrimSpace(caPath),
		Verifier:      platformlicense.NewDefaultVerifier(),
	}
}

func (s *Service) Status(ctx context.Context) (StatusDTO, error) {
	if s == nil || s.RuntimeClient == nil {
		return StatusDTO{}, fmt.Errorf("license runtime client is not initialized")
	}
	clusterCount, err := s.CountClusters(ctx)
	if err != nil {
		return StatusDTO{}, err
	}

	configMap := &corev1.ConfigMap{}
	err = s.reader().Get(ctx, types.NamespacedName{
		Namespace: s.effectiveNamespace(),
		Name:      platformlicense.StatusConfigMapName,
	}, configMap)
	if err == nil {
		status := statusFromConfigMap(configMap)
		status.ClusterCount = clusterCount
		status.Source = StatusSourceConfigMap
		return s.withCurrentFingerprint(ctx, status), nil
	}
	if !apierrors.IsNotFound(err) {
		return StatusDTO{}, err
	}
	return s.liveStatus(ctx, clusterCount)
}

func (s *Service) Install(ctx context.Context, raw []byte) (StatusDTO, error) {
	if s == nil || s.RuntimeClient == nil {
		return StatusDTO{}, fmt.Errorf("license runtime client is not initialized")
	}
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return StatusDTO{}, fmt.Errorf("license is required")
	}

	namespace := s.effectiveNamespace()
	desired := platformlicense.BuildLicenseSecret(namespace, raw)
	existing := &corev1.Secret{}
	key := types.NamespacedName{Namespace: namespace, Name: platformlicense.LicenseSecretName}
	err := s.RuntimeClient.Get(ctx, key, existing)
	if apierrors.IsNotFound(err) {
		if err := s.RuntimeClient.Create(ctx, desired); err != nil {
			return StatusDTO{}, err
		}
	} else if err != nil {
		return StatusDTO{}, err
	} else {
		existing.Type = desired.Type
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		for k, v := range desired.Labels {
			existing.Labels[k] = v
		}
		if existing.Data == nil {
			existing.Data = map[string][]byte{}
		}
		existing.Data[platformlicense.LicenseSecretDataKey] = raw
		if err := s.RuntimeClient.Update(ctx, existing); err != nil {
			return StatusDTO{}, err
		}
	}

	clusterCount, err := s.CountClusters(ctx)
	if err != nil {
		return StatusDTO{}, err
	}
	return s.liveStatus(ctx, clusterCount)
}

func (s *Service) CheckClusterCreate(ctx context.Context) (ClusterCreateCheck, error) {
	if s == nil || s.RuntimeClient == nil {
		return ClusterCreateCheck{}, fmt.Errorf("license runtime client is not initialized")
	}
	currentCount, err := s.CountClusters(ctx)
	if err != nil {
		return ClusterCreateCheck{}, err
	}
	entitlement := s.store().Evaluate(ctx, s.effectiveVerifier())
	status := statusFromEntitlement(entitlement, currentCount, StatusSourceLiveEvaluation, s.now())
	if entitlement.CanCreateCluster(currentCount) {
		return ClusterCreateCheck{
			Allowed: true,
			Status:  status,
		}, nil
	}

	licenseReason := entitlement.StableReason()
	if licenseReason == "" {
		licenseReason = entitlement.Reason
	}
	meta := LicenseErrorMeta{
		Reason:          platformlicense.ReasonLicenseLimitExceeded,
		LicenseReason:   licenseReason,
		State:           string(entitlement.State),
		MaxClusters:     entitlement.ClusterLimit(),
		CurrentClusters: currentCount,
	}
	return ClusterCreateCheck{
		Allowed: false,
		Status:  status,
		Meta:    meta,
		Message: fmt.Sprintf("cluster quota exceeded: maxClusters=%d currentClusters=%d licenseState=%s", entitlement.ClusterLimit(), currentCount, entitlement.State),
	}, nil
}

func (s *Service) CountClusters(ctx context.Context) (int, error) {
	if s == nil || s.RuntimeClient == nil {
		return 0, fmt.Errorf("license runtime client is not initialized")
	}
	clusters := &dapisv1.ClusterList{}
	if err := s.reader().List(ctx, clusters); err != nil {
		return 0, err
	}
	count := 0
	for i := range clusters.Items {
		if clusters.Items[i].DeletionTimestamp.IsZero() {
			count++
		}
	}
	return count, nil
}

func (s *Service) liveStatus(ctx context.Context, clusterCount int) (StatusDTO, error) {
	entitlement := s.store().Evaluate(ctx, s.effectiveVerifier())
	status := statusFromEntitlement(entitlement, clusterCount, StatusSourceLiveEvaluation, s.now())
	return s.withCurrentFingerprint(ctx, status), nil
}

func (s *Service) withCurrentFingerprint(ctx context.Context, status StatusDTO) StatusDTO {
	now := s.now().Truncate(time.Second)
	status.FingerprintVersion = platformlicense.FingerprintVersionK8SV1

	fingerprint, err := s.store().Fingerprint(ctx)
	if err != nil {
		status.Fingerprint = ""
		status.FingerprintRequest = nil
		if shouldPreserveLicenseContentStatus(status) {
			return status
		}
		return StatusDTO{
			State:              string(platformlicense.StateUnknown),
			MaxClusters:        platformlicense.DefaultFreeMaxClusters,
			ClusterCount:       status.ClusterCount,
			Fingerprint:        "",
			FingerprintVersion: platformlicense.FingerprintVersionK8SV1,
			FingerprintMatched: false,
			Reason:             platformlicense.ReasonLicenseEnvironmentInvalid,
			Message:            fmt.Sprintf("compute deployment fingerprint: %v", err),
			LastCheckedAt:      now.Format(time.RFC3339),
			Source:             status.Source,
		}
	}

	status.Fingerprint = fingerprint
	status.FingerprintRequest = &FingerprintRequestDTO{
		Product:            platformlicense.ProductName,
		FingerprintVersion: platformlicense.FingerprintVersionK8SV1,
		Fingerprint:        fingerprint,
		Namespace:          s.effectiveNamespace(),
		GeneratedAt:        now.Format(time.RFC3339),
	}
	return status
}

func shouldPreserveLicenseContentStatus(status StatusDTO) bool {
	switch platformlicense.State(status.State) {
	case platformlicense.StateExpired,
		platformlicense.StateInvalidSignature,
		platformlicense.StateMalformed,
		platformlicense.StateUnsupportedVersion,
		platformlicense.StateProductMismatch,
		platformlicense.StateUnknownKey,
		platformlicense.StateNotYetValid:
		return true
	default:
		return false
	}
}

func (s *Service) store() platformlicense.KubernetesStore {
	return platformlicense.KubernetesStore{
		Client:    s.RuntimeClient,
		Reader:    s.reader(),
		Namespace: s.effectiveNamespace(),
		CAPath:    s.effectiveCAPath(),
		CABundle:  s.CABundle,
		Now:       s.Now,
	}
}

func (s *Service) reader() client.Reader {
	if s != nil && s.RuntimeReader != nil {
		return s.RuntimeReader
	}
	if s != nil {
		return s.RuntimeClient
	}
	return nil
}

func (s *Service) effectiveNamespace() string {
	if s != nil && strings.TrimSpace(s.Namespace) != "" {
		return strings.TrimSpace(s.Namespace)
	}
	return platformlicense.DefaultLicenseNamespace
}

func (s *Service) effectiveCAPath() string {
	if s != nil && strings.TrimSpace(s.CAPath) != "" {
		return strings.TrimSpace(s.CAPath)
	}
	return platformlicense.DefaultServiceAccountCAPath
}

func (s *Service) effectiveVerifier() *platformlicense.Verifier {
	if s != nil && s.Verifier != nil {
		return s.Verifier
	}
	return platformlicense.NewDefaultVerifier()
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func statusFromEntitlement(entitlement platformlicense.Entitlement, clusterCount int, source string, now time.Time) StatusDTO {
	status := StatusDTO{
		State:              string(entitlement.State),
		Edition:            entitlement.Edition,
		LicenseID:          entitlement.LicenseID,
		Customer:           entitlement.CustomerName(),
		MaxClusters:        entitlement.ClusterLimit(),
		ClusterCount:       clusterCount,
		FingerprintMatched: entitlement.FingerprintMatched,
		Reason:             entitlement.Reason,
		Message:            entitlement.Message,
		LastCheckedAt:      now.Truncate(time.Second).Format(time.RFC3339),
		Source:             source,
		Features:           featuresFromEntitlement(entitlement),
	}
	if entitlement.ExpiresAt != nil {
		status.ExpiresAt = entitlement.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return status
}

func statusFromConfigMap(configMap *corev1.ConfigMap) StatusDTO {
	data := map[string]string{}
	if configMap != nil {
		data = configMap.Data
	}
	maxClusters, _ := strconv.Atoi(strings.TrimSpace(data["maxClusters"]))
	clusterCount, _ := strconv.Atoi(strings.TrimSpace(data["clusterCount"]))
	fingerprintMatched, _ := strconv.ParseBool(strings.TrimSpace(data["fingerprintMatched"]))
	return StatusDTO{
		State:              data["state"],
		Edition:            data["edition"],
		LicenseID:          data["licenseId"],
		Customer:           data["customer"],
		ExpiresAt:          data["expiresAt"],
		MaxClusters:        maxClusters,
		ClusterCount:       clusterCount,
		FingerprintMatched: fingerprintMatched,
		Reason:             data["reason"],
		Message:            data["message"],
		LastCheckedAt:      data["lastCheckedAt"],
		Source:             StatusSourceConfigMap,
	}
}

func featuresFromEntitlement(entitlement platformlicense.Entitlement) []string {
	if len(entitlement.Features) == 0 {
		return nil
	}
	features := make([]string, 0, len(entitlement.Features))
	for feature, enabled := range entitlement.Features {
		if enabled {
			features = append(features, feature)
		}
	}
	sort.Strings(features)
	return features
}
