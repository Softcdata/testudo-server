package appbackup

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hlog "github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/transport"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type VeleroBackupIncludesDTO struct {
	IncludedNamespaces []string `json:"includedNamespaces"`
	IncludedResources  []string `json:"includedResources"`
}

type backupResourceListFetcher func(ctx context.Context, remote ctrlclient.Reader, backup *velerov1.Backup, httpClient *http.Client) (map[string][]string, error)

const (
	downloadRequestPollInterval = 100 * time.Millisecond
	downloadRequestTimeout      = 10 * time.Second
	downloadBodyTimeout         = 5 * time.Second
	resourceListURLExpiry       = 2 * time.Minute
)

// getVeleroBackupIncludes returns includedNamespaces/includedResources from Velero Backup spec.
//
// Route: GET /apis/appbackups.testudo.softcdata.com/v1/velero/backups/:backupName/includes?cluster=<clusterName>
func (h *AppBackupHandler) getVeleroBackupIncludes(c context.Context, ctx *app.RequestContext) {
	backupName := strings.TrimSpace(ctx.Param("backupName"))
	if backupName == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationBackupNameRequired, nil, nil)
		return
	}

	clusterName := strings.TrimSpace(string(ctx.Query("cluster")))
	if clusterName == "" {
		clusterName = strings.TrimSpace(string(ctx.Query("clusterName")))
	}
	if clusterName == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationClusterRequired, nil, nil)
		return
	}

	if h.getRemoteClient == nil {
		transport.WriteErrorKey(ctx, transport.CodeInternalServerError, i18n.KeyRemoteClientNotReady, nil, nil)
		return
	}
	remote, err := h.getRemoteClient(c, clusterName)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	backup := &velerov1.Backup{}
	if err := remote.Get(c, ctrlclient.ObjectKey{Name: backupName, Namespace: common.VeleroNamespace}, backup); err != nil {
		if apierrors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, fmt.Sprintf("velero backup %s not found in cluster %s", backupName, clusterName), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	if cached, ok := h.getCachedBackupIncludes(clusterName, backup); ok {
		transport.WriteSuccess(ctx, consts.StatusOK, cached, nil)
		return
	}

	resp, err := h.computeVeleroBackupIncludes(c, clusterName, remote, backup)
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}
	transport.WriteSuccess(ctx, consts.StatusOK, resp, nil)
}

func (h *AppBackupHandler) computeVeleroBackupIncludes(ctx context.Context, clusterName string, remote ctrlclient.Reader, backup *velerov1.Backup) (VeleroBackupIncludesDTO, error) {
	if h == nil || backup == nil {
		return VeleroBackupIncludesDTO{}, fmt.Errorf("backup handler is not initialized")
	}
	if h.includesCache == nil {
		dto, _, err := h.computeVeleroBackupIncludesUncached(ctx, clusterName, remote, backup)
		return dto, err
	}

	if cached, ok := h.getCachedBackupIncludes(clusterName, backup); ok {
		return cached, nil
	}

	dto, cacheable, err := h.computeVeleroBackupIncludesUncached(ctx, clusterName, remote, backup)
	if err != nil {
		return VeleroBackupIncludesDTO{}, err
	}
	if cacheable {
		h.setCachedBackupIncludes(clusterName, backup, dto)
	}
	return dto, nil
}

func (h *AppBackupHandler) computeVeleroBackupIncludesUncached(ctx context.Context, clusterName string, remote ctrlclient.Reader, backup *velerov1.Backup) (VeleroBackupIncludesDTO, bool, error) {
	includedNamespaces := backup.Spec.IncludedNamespaces
	if includedNamespaces == nil {
		includedNamespaces = make([]string, 0)
	}
	includedResources := backup.Spec.IncludedResources
	if includedResources == nil {
		includedResources = make([]string, 0)
	}

	if resourceList, err := h.loadBackupResourceListUncached(ctx, clusterName, remote, backup); err == nil {
		actualNamespaces, actualResources := buildIncludesFromResourceList(resourceList)
		return VeleroBackupIncludesDTO{
			IncludedNamespaces: actualNamespaces,
			IncludedResources:  actualResources,
		}, true, nil
	} else {
		hlog.CtxWarnf(
			ctx,
			"failed to fetch velero backup resource list, fallback to backup spec: cluster=%s backup=%s storageLocation=%s err=%v",
			clusterName,
			backup.Name,
			backup.Spec.StorageLocation,
			err,
		)
	}

	return VeleroBackupIncludesDTO{
		IncludedNamespaces: includedNamespaces,
		IncludedResources:  includedResources,
	}, false, nil
}

func (h *AppBackupHandler) loadBackupResourceListUncached(ctx context.Context, clusterName string, remote ctrlclient.Reader, backup *velerov1.Backup) (map[string][]string, error) {
	if resourceList, err := h.fetchBackupResourceListFromStorage(ctx, clusterName, remote, backup); err == nil {
		return resourceList, nil
	}

	if h.fetchBackupResourceList == nil {
		return nil, fmt.Errorf("backup resource list fetcher is not configured")
	}

	httpClient := h.buildBackupResourceListHTTPClient(ctx, clusterName, backup)
	return h.fetchBackupResourceList(ctx, remote, backup, httpClient)
}

func (h *AppBackupHandler) fetchBackupResourceListFromStorage(ctx context.Context, clusterName string, remote ctrlclient.Reader, backup *velerov1.Backup) (map[string][]string, error) {
	if h == nil || h.Storage == nil {
		return nil, fmt.Errorf("storage service is not initialized")
	}

	repo, err := h.resolveStorageRepositoryForBackup(ctx, clusterName, backup)
	if err != nil {
		return nil, err
	}

	caBundle, err := h.loadStorageCABundle(ctx, repo)
	if err != nil {
		return nil, err
	}

	objectKey, err := h.resolveBackupResourceListObjectKey(ctx, remote, clusterName, backup)
	if err != nil {
		return nil, err
	}

	downloadURL, err := h.Storage.GetDownloadURL(
		ctx,
		repo.Spec.Endpoint,
		repo.Spec.AccessKey,
		repo.Spec.SecretKey,
		repo.Spec.Bucket,
		repo.Spec.Region,
		repo.Spec.GetAddressingStyle(),
		caBundle,
		objectKey,
		resourceListURLExpiry,
	)
	if err != nil {
		return nil, err
	}

	httpClient, err := newBackupResourceHTTPClient(caBundle)
	if err != nil {
		return nil, err
	}

	return downloadAndDecodeBackupResourceList(ctx, downloadURL, httpClient)
}

func (h *AppBackupHandler) buildBackupResourceListHTTPClient(ctx context.Context, clusterName string, backup *velerov1.Backup) *http.Client {
	httpClient, err := newBackupResourceHTTPClient(nil)
	if err != nil {
		hlog.CtxWarnf(ctx, "failed to create default backup resource http client: %v", err)
		return http.DefaultClient
	}

	repo, err := h.resolveStorageRepositoryForBackup(ctx, clusterName, backup)
	if err != nil {
		hlog.CtxWarnf(
			ctx,
			"failed to resolve storage repository for velero backup includes, using default CA roots: cluster=%s backup=%s storageLocation=%s err=%v",
			clusterName,
			backup.Name,
			backup.Spec.StorageLocation,
			err,
		)
		return httpClient
	}

	caBundle, err := h.loadStorageCABundle(ctx, repo)
	if err != nil {
		hlog.CtxWarnf(
			ctx,
			"failed to load storage CA bundle for velero backup includes, using default CA roots: cluster=%s backup=%s storageRepository=%s err=%v",
			clusterName,
			backup.Name,
			repo.Name,
			err,
		)
		return httpClient
	}
	if len(caBundle) == 0 {
		return httpClient
	}

	customHTTPClient, err := newBackupResourceHTTPClient(caBundle)
	if err != nil {
		hlog.CtxWarnf(
			ctx,
			"failed to build CA-aware backup resource http client, using default CA roots: cluster=%s backup=%s storageRepository=%s err=%v",
			clusterName,
			backup.Name,
			repo.Name,
			err,
		)
		return httpClient
	}
	return customHTTPClient
}

func (h *AppBackupHandler) resolveBackupResourceListObjectKey(ctx context.Context, remote ctrlclient.Reader, clusterName string, backup *velerov1.Backup) (string, error) {
	prefix := strings.TrimSpace(clusterName)
	storageLocation := strings.TrimSpace(backup.Spec.StorageLocation)

	if storageLocation != "" {
		bsl := &velerov1.BackupStorageLocation{}
		err := remote.Get(ctx, ctrlclient.ObjectKey{Name: storageLocation, Namespace: backup.Namespace}, bsl)
		switch {
		case err == nil:
			if bslPrefix := strings.TrimSpace(bsl.Spec.ObjectStorage.Prefix); bslPrefix != "" {
				prefix = bslPrefix
			}
		case apierrors.IsNotFound(err):
			// Keep clusterName fallback.
		default:
			return "", err
		}
	}

	fileName := fmt.Sprintf("%s-resource-list.json.gz", backup.Name)
	if prefix == "" {
		return path.Join("backups", backup.Name, fileName), nil
	}
	return path.Join(prefix, "backups", backup.Name, fileName), nil
}

func (h *AppBackupHandler) resolveStorageRepositoryForBackup(ctx context.Context, clusterName string, backup *velerov1.Backup) (*dapisv1.StorageRepository, error) {
	if h == nil || h.DisasterClient == nil {
		return nil, fmt.Errorf("disaster client is not initialized")
	}

	storageLocation := strings.TrimSpace(backup.Spec.StorageLocation)
	if storageLocation == "" {
		return nil, fmt.Errorf("velero backup storageLocation is empty")
	}

	var lastErr error
	for _, name := range candidateStorageRepositoryNames(storageLocation, clusterName) {
		repo, err := h.DisasterClient.DisasterV1().StorageRepositories(common.DisasterSystemNamespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil {
			return repo, nil
		}
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("storage repository %s not found", storageLocation)
	}
	return nil, fmt.Errorf("storage repository %s not found", storageLocation)
}

func candidateStorageRepositoryNames(storageLocation, clusterName string) []string {
	storageLocation = strings.TrimSpace(storageLocation)
	clusterName = strings.TrimSpace(clusterName)
	if storageLocation == "" {
		return nil
	}

	names := []string{storageLocation}
	if clusterName == "" {
		return names
	}

	suffix := "-" + clusterName
	if strings.HasSuffix(storageLocation, suffix) {
		trimmed := strings.TrimSuffix(storageLocation, suffix)
		if trimmed != "" && trimmed != storageLocation {
			names = append(names, trimmed)
		}
	}
	return names
}

func newBackupResourceHTTPClient(caBundle []byte) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default transport is not *http.Transport")
	}

	cloned := transport.Clone()
	if len(caBundle) > 0 {
		rootCAs, err := x509.SystemCertPool()
		if err != nil || rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
			return nil, fmt.Errorf("failed to append backup resource CA bundle")
		}

		if cloned.TLSClientConfig == nil {
			cloned.TLSClientConfig = &tls.Config{}
		} else {
			cloned.TLSClientConfig = cloned.TLSClientConfig.Clone()
		}
		cloned.TLSClientConfig.RootCAs = rootCAs
	}

	return &http.Client{
		Transport: cloned,
		Timeout:   downloadBodyTimeout,
	}, nil
}

func backupIncludesCacheKey(clusterName string, backup *velerov1.Backup) string {
	return strings.Join([]string{
		strings.TrimSpace(clusterName),
		strings.TrimSpace(backup.Namespace),
		strings.TrimSpace(backup.Name),
	}, "|")
}

func (h *AppBackupHandler) getCachedBackupIncludes(clusterName string, backup *velerov1.Backup) (VeleroBackupIncludesDTO, bool) {
	if h == nil || h.includesCache == nil || backup == nil {
		return VeleroBackupIncludesDTO{}, false
	}

	key := backupIncludesCacheKey(clusterName, backup)
	now := time.Now()

	h.includesCache.mu.RLock()
	entry, ok := h.includesCache.entries[key]
	h.includesCache.mu.RUnlock()
	if !ok {
		return VeleroBackupIncludesDTO{}, false
	}

	if now.After(entry.expiresAt) || entry.resourceVersion != backup.ResourceVersion {
		h.includesCache.mu.Lock()
		delete(h.includesCache.entries, key)
		h.includesCache.mu.Unlock()
		return VeleroBackupIncludesDTO{}, false
	}
	return entry.dto, true
}

func (h *AppBackupHandler) setCachedBackupIncludes(clusterName string, backup *velerov1.Backup, dto VeleroBackupIncludesDTO) {
	if h == nil || h.includesCache == nil || backup == nil {
		return
	}

	h.includesCache.mu.Lock()
	h.includesCache.entries[backupIncludesCacheKey(clusterName, backup)] = cachedVeleroBackupIncludes{
		dto:             dto,
		resourceVersion: backup.ResourceVersion,
		expiresAt:       time.Now().Add(backupIncludesCacheTTL),
	}
	h.includesCache.mu.Unlock()
}

func fetchBackupResourceListFromDownloadRequest(ctx context.Context, remote ctrlclient.Reader, backup *velerov1.Backup, httpClient *http.Client) (map[string][]string, error) {
	remoteClient, ok := remote.(ctrlclient.Client)
	if !ok {
		return nil, fmt.Errorf("remote client does not support create/get operations")
	}

	req := &velerov1.DownloadRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%d", backup.Name, time.Now().UnixNano()),
			Namespace: backup.Namespace,
		},
		Spec: velerov1.DownloadRequestSpec{
			Target: velerov1.DownloadTarget{
				Kind: velerov1.DownloadTargetKindBackupResourceList,
				Name: backup.Name,
			},
		},
	}
	if err := remoteClient.Create(ctx, req); err != nil {
		return nil, err
	}
	defer func() {
		_ = remoteClient.Delete(context.Background(), req)
	}()

	downloadURL, err := waitDownloadRequestURL(ctx, remoteClient, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return downloadAndDecodeBackupResourceList(ctx, downloadURL, httpClient)
}

func waitDownloadRequestURL(ctx context.Context, remoteClient ctrlclient.Client, namespace, name string) (string, error) {
	pollCtx, cancel := context.WithTimeout(ctx, downloadRequestTimeout)
	defer cancel()

	ticker := time.NewTicker(downloadRequestPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pollCtx.Done():
			return "", fmt.Errorf("waiting download request timed out: %w", pollCtx.Err())
		case <-ticker.C:
			req := &velerov1.DownloadRequest{}
			if err := remoteClient.Get(pollCtx, ctrlclient.ObjectKey{Name: name, Namespace: namespace}, req); err != nil {
				return "", err
			}
			if strings.TrimSpace(req.Status.DownloadURL) != "" {
				return req.Status.DownloadURL, nil
			}
		}
	}
}

func downloadAndDecodeBackupResourceList(ctx context.Context, downloadURL string, httpClient *http.Client) (map[string][]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	if httpClient == nil {
		httpClient, err = newBackupResourceHTTPClient(nil)
		if err != nil {
			return nil, err
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download resource list failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("download resource list is empty")
	}

	decoded, err := tryGunzip(body)
	if err != nil {
		return nil, err
	}

	resourceList := make(map[string][]string)
	if err := json.Unmarshal(decoded, &resourceList); err != nil {
		return nil, err
	}
	return resourceList, nil
}

func tryGunzip(payload []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		// BackupResourceList is expected to be gzipped, but keep plain JSON fallback for robustness.
		return payload, nil
	}
	defer reader.Close()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func buildIncludesFromResourceList(resourceList map[string][]string) ([]string, []string) {
	resources := make([]string, 0, len(resourceList))
	namespacesSet := make(map[string]struct{})

	for resource, items := range resourceList {
		resources = append(resources, resource)
		for _, item := range items {
			if idx := strings.Index(item, "/"); idx > 0 {
				namespacesSet[item[:idx]] = struct{}{}
			}
		}
	}

	sort.Strings(resources)

	namespaces := make([]string, 0, len(namespacesSet))
	for ns := range namespacesSet {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)

	return namespaces, resources
}
