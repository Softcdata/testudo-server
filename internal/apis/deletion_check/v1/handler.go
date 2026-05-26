package deletioncheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/i18n"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/transport"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeletionCheckHandler implements the unified dependency check endpoint:
// POST /api/v1/deletion/check (and /apis/v1/deletion/check)
//
// This endpoint is a "read-only decision helper": it never blocks deletes server-side.
// can_delete is derived only from upstream emptiness, per openspec.
type DeletionCheckHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc

	getRemoteClient remoteClientGetter
}

func NewDeletionCheckHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *DeletionCheckHandler {
	h := &DeletionCheckHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
	}

	// Default implementation uses Cluster CR to build a client for the target cluster.
	// Tests can override this getter to return a fake remote client.
	h.getRemoteClient = func(ctx context.Context, clusterName string) (remoteLister, error) {
		if kc == nil || kc.ClusterClient == nil {
			return nil, fmt.Errorf("cluster client is not initialized")
		}
		clusterName = strings.TrimSpace(clusterName)
		if clusterName == "" {
			return nil, fmt.Errorf("cluster name is empty")
		}
		remote, err := kc.GetKubeClient(ctx, kc.RuntimeClient(), kc.Scheme(), clusterName)
		if err != nil {
			return nil, err
		}
		return remote, nil
	}

	return h
}

func (h *DeletionCheckHandler) check(c context.Context, ctx *app.RequestContext) {
	var req DeletionCheckRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	kind := normalizeKind(req.ResourceKind)
	if kind == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationUnsupportedResourceKind, map[string]any{"kind": req.ResourceKind, "supported": supportedKindsHint()}, nil)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		transport.WriteErrorKey(ctx, transport.CodeBadRequest, i18n.KeyValidationNameRequired, nil, nil)
		return
	}

	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		namespace = common.DisasterSystemNamespace
	}

	targetObj, err := h.getObjectByKind(c, kind, namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
			return
		}
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	targetRef := ResourceRef{
		Kind:      kind,
		Name:      targetObj.GetName(),
		Namespace: targetObj.GetNamespace(),
		UID:       string(targetObj.GetUID()),
	}

	targetToken := h.getDependencyToken(targetObj)
	if targetToken == "" {
		transport.WriteError(ctx, transport.CodeInternalServerError, "failed to derive dependency-token for target", nil)
		return
	}

	cleanupPlan := h.buildCleanupPlan(c, kind, namespace, targetObj)

	upstreamKey := dependencyToLabelKey(targetToken)
	upstream := h.queryUpstreamByLabelKey(c, upstreamKey, namespace, targetRef.UID, func(upstreamKind string, upstreamObj metav1.Object) bool {
		// Exclude sub-resources that will be deleted together with the target.
		// They must appear only in cleanup_plan, not in upstream (blocking list).
		if upstreamObj != nil && targetRef.UID != "" && hasOwnerUID(upstreamObj.GetOwnerReferences(), targetRef.UID) {
			return false
		}
		switch kind {
		case kindDisasterInstance:
			if upstreamKind == kindDataSync || upstreamKind == kindResourceSync || upstreamKind == kindDisasterOperation {
				return false
			}
		case kindDisasterGroup, kindDisasterDrill:
			if upstreamKind == kindDisasterOperation {
				return false
			}
		}
		return true
	})
	upstream = append(upstream, ownerUpstreamRefs(targetObj)...)
	upstream = dedupDependencyRefs(upstream)

	downstream := h.queryDownstreamByLabels(c, targetObj.GetLabels(), namespace)

	if upstream == nil {
		upstream = make([]DependencyRef, 0)
	}
	if downstream == nil {
		downstream = make([]DependencyRef, 0)
	}

	canDelete := canDeleteFromUpstreamCount(len(upstream))

	resp := DeletionCheckResponse{
		Target:      targetRef,
		Upstream:    upstream,
		Downstream:  downstream,
		CleanupPlan: cleanupPlan,
		CanDelete:   canDelete,
		Message:     "OK",
	}
	transport.WriteSuccess(ctx, consts.StatusOK, resp, nil)
}

func (h *DeletionCheckHandler) getDependencyToken(obj metav1.Object) string {
	if obj == nil {
		return ""
	}
	labels := obj.GetLabels()
	if len(labels) > 0 {
		if token := normalizeHexToken(labels[labelDependencyToken]); token != "" {
			return token
		}
	}
	// Backward-compat / backfill window: if token label is missing, derive from UID.
	return buildDependencyToken(string(obj.GetUID()))
}

type upstreamIncludeFunc func(upstreamKind string, upstreamObj metav1.Object) bool

func (h *DeletionCheckHandler) queryUpstreamByLabelKey(c context.Context, dependencyToKey string, namespace string, targetUID string, include upstreamIncludeFunc) []DependencyRef {
	if strings.TrimSpace(dependencyToKey) == "" {
		return nil
	}

	seen := make(map[string]struct{}, 64)
	out := make([]DependencyRef, 0)

	for _, kind := range supportedKindsForScan() {
		objects, err := h.listObjectsByKind(c, kind, namespace, dependencyToKey)
		if err != nil {
			// Best-effort: don't fail the whole request due to one kind list error.
			continue
		}
		for _, obj := range objects {
			if obj == nil {
				continue
			}
			if include != nil && !include(kind, obj) {
				continue
			}
			if targetUID != "" && string(obj.GetUID()) == targetUID {
				continue
			}

			refKey := kind + "/" + obj.GetNamespace() + "/" + obj.GetName()
			if _, ok := seen[refKey]; ok {
				continue
			}
			seen[refKey] = struct{}{}

			labels := obj.GetLabels()
			relationCode := ""
			if len(labels) > 0 {
				relationCode = normalizeRelationCode(labels[dependencyToKey])
			}

			out = append(out, DependencyRef{
				Kind:         kind,
				Name:         obj.GetName(),
				Namespace:    obj.GetNamespace(),
				UID:          string(obj.GetUID()),
				RelationCode: relationCode,
				Token:        h.getDependencyToken(obj),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func ownerUpstreamRefs(target metav1.Object) []DependencyRef {
	if target == nil {
		return nil
	}
	refs := target.GetOwnerReferences()
	if len(refs) == 0 {
		return nil
	}

	ns := target.GetNamespace()
	out := make([]DependencyRef, 0, len(refs))
	for _, r := range refs {
		if strings.TrimSpace(r.Kind) == "" || strings.TrimSpace(r.Name) == "" {
			continue
		}
		out = append(out, DependencyRef{
			Kind:         r.Kind,
			Name:         r.Name,
			Namespace:    ns,
			UID:          string(r.UID),
			RelationCode: "ownerReference",
		})
	}
	return out
}

func dedupDependencyRefs(in []DependencyRef) []DependencyRef {
	if len(in) == 0 {
		return in
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]DependencyRef, 0, len(in))
	for _, ref := range in {
		key := ref.Kind + "/" + ref.Namespace + "/" + ref.Name + "/" + ref.UID + "/" + ref.RelationCode
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (h *DeletionCheckHandler) queryDownstreamByLabels(c context.Context, labels map[string]string, namespace string) []DependencyRef {
	edges := parseDownstreamFromLabels(labels)
	if len(edges) == 0 {
		return nil
	}

	out := make([]DependencyRef, 0, len(edges))
	for _, edge := range edges {
		ref := DependencyRef{
			RelationCode: edge.RelationCode,
			Token:        edge.TargetToken,
		}

		obj, kind := h.findObjectByToken(c, edge.TargetToken, namespace)
		if obj == nil {
			ref.Unresolved = true
			out = append(out, ref)
			continue
		}

		ref.Kind = kind
		ref.Name = obj.GetName()
		ref.Namespace = obj.GetNamespace()
		ref.UID = string(obj.GetUID())
		out = append(out, ref)
	}
	return out
}

func (h *DeletionCheckHandler) findObjectByToken(c context.Context, token string, namespace string) (metav1.Object, string) {
	token = normalizeHexToken(token)
	if token == "" {
		return nil, ""
	}

	selector := fmt.Sprintf("%s=%s", labelDependencyToken, token)
	for _, kind := range supportedKindsForScan() {
		objs, err := h.listObjectsByKind(c, kind, namespace, selector)
		if err != nil {
			continue
		}
		if len(objs) > 0 && objs[0] != nil {
			return objs[0], kind
		}
	}
	return nil, ""
}

func (h *DeletionCheckHandler) getObjectByKind(c context.Context, kind string, namespace string, name string) (metav1.Object, error) {
	switch kind {
	case kindCluster:
		return h.DisasterClient.DisasterV1().Clusters().Get(c, name, metav1.GetOptions{})
	case kindDisasterConfig:
		return h.DisasterClient.DisasterV1().DisasterConfigs().Get(c, name, metav1.GetOptions{})
	case kindStorageRepository:
		return h.DisasterClient.DisasterV1().StorageRepositories(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterPolicy:
		return h.DisasterClient.DisasterV1().DisasterPolicies(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterInstance:
		return h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterGroup:
		return h.DisasterClient.DisasterV1().DisasterGroups(namespace).Get(c, name, metav1.GetOptions{})
	case kindAppBackup:
		return h.DisasterClient.DisasterV1().AppBackups(namespace).Get(c, name, metav1.GetOptions{})
	case kindAppRestore:
		return h.DisasterClient.DisasterV1().AppRestores(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterDrill:
		return h.DisasterClient.DisasterV1().DisasterDrills(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterBackup:
		return h.DisasterClient.DisasterV1().DisasterBackups(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterOperation:
		return h.DisasterClient.DisasterV1().DisasterOperations(namespace).Get(c, name, metav1.GetOptions{})
	case kindDataSync:
		return h.DisasterClient.DisasterV1().DataSyncs(namespace).Get(c, name, metav1.GetOptions{})
	case kindResourceSync:
		return h.DisasterClient.DisasterV1().ResourceSyncs(namespace).Get(c, name, metav1.GetOptions{})
	case kindDisasterJob:
		return h.DisasterClient.DisasterV1().DisasterJobs(namespace).Get(c, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

func (h *DeletionCheckHandler) listObjectsByKind(c context.Context, kind string, namespace string, labelSelector string) ([]metav1.Object, error) {
	opts := metav1.ListOptions{LabelSelector: strings.TrimSpace(labelSelector)}
	switch kind {
	case kindCluster:
		list, err := h.DisasterClient.DisasterV1().Clusters().List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterConfig:
		list, err := h.DisasterClient.DisasterV1().DisasterConfigs().List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindStorageRepository:
		list, err := h.DisasterClient.DisasterV1().StorageRepositories(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterPolicy:
		list, err := h.DisasterClient.DisasterV1().DisasterPolicies(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterInstance:
		list, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterGroup:
		list, err := h.DisasterClient.DisasterV1().DisasterGroups(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindAppBackup:
		list, err := h.DisasterClient.DisasterV1().AppBackups(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindAppRestore:
		list, err := h.DisasterClient.DisasterV1().AppRestores(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterDrill:
		list, err := h.DisasterClient.DisasterV1().DisasterDrills(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterBackup:
		list, err := h.DisasterClient.DisasterV1().DisasterBackups(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterOperation:
		list, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDataSync:
		list, err := h.DisasterClient.DisasterV1().DataSyncs(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindResourceSync:
		list, err := h.DisasterClient.DisasterV1().ResourceSyncs(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	case kindDisasterJob:
		list, err := h.DisasterClient.DisasterV1().DisasterJobs(namespace).List(c, opts)
		if err != nil {
			return nil, err
		}
		out := make([]metav1.Object, 0, len(list.Items))
		for i := range list.Items {
			out = append(out, &list.Items[i])
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported kind: %s", kind)
	}
}

const (
	// Dependency labels (defined by openspec unify-deletion-protection).
	// Keep these strings stable: do not change existing system labels.
	labelDependencyToken    = "testudo.softcdata.com/dependency-token"
	labelDependencyToPrefix = "testudo.softcdata.com/dependency-to-"

	kindCluster           = "Cluster"
	kindStorageRepository = "StorageRepository"
	kindDisasterPolicy    = "DisasterPolicy"
	kindDisasterConfig    = "DisasterConfig"
	kindDisasterInstance  = "DisasterInstance"
	kindDisasterGroup     = "DisasterGroup"
	kindAppBackup         = "AppBackup"
	kindAppRestore        = "AppRestore"
	kindDisasterDrill     = "DisasterDrill"
	kindDisasterBackup    = "DisasterBackup"

	// Internal source modules for upstream completeness (not necessarily exposed as deletable resources).
	kindDisasterOperation = "DisasterOperation"
	kindDataSync          = "DataSync"
	kindResourceSync      = "ResourceSync"

	// Compatibility only (spec requirement mentions it only for DisasterPolicy).
	kindDisasterJob = "DisasterJob"
)

func supportedKindsForScan() []string {
	return []string{
		kindCluster,
		kindStorageRepository,
		kindDisasterPolicy,
		kindDisasterConfig,
		kindDisasterInstance,
		kindDisasterGroup,
		kindAppBackup,
		kindAppRestore,
		kindDisasterDrill,
		kindDisasterBackup,
		kindDisasterOperation,
		kindDataSync,
		kindResourceSync,
		kindDisasterJob,
	}
}

func supportedKindsHint() string {
	// Keep this stable and friendly for clients.
	return strings.Join([]string{
		"Cluster",
		"StorageRepository",
		"DisasterPolicy",
		"DisasterConfig",
		"DisasterInstance",
		"DisasterGroup",
		"AppBackup",
		"AppRestore",
		"DisasterDrill",
		"DisasterBackup",
		"DisasterOperation",
		"DataSync",
		"ResourceSync",
		"DisasterJob",
	}, ", ")
}

type dependencyEdge struct {
	TargetToken  string
	RelationCode string
}

func buildDependencyToken(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(uid))
	return hex.EncodeToString(sum[:])[:16]
}

func dependencyToLabelKey(token string) string {
	token = normalizeHexToken(token)
	if token == "" {
		return ""
	}
	return labelDependencyToPrefix + token
}

func normalizeRelationCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "unknown"
	}

	var b strings.Builder
	b.Grow(len(code))
	for _, ch := range code {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '-' || ch == '_' || ch == '.' {
			b.WriteRune(ch)
			continue
		}
		b.WriteByte('-')
	}

	normalized := strings.TrimFunc(b.String(), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if normalized == "" {
		return "unknown"
	}
	if len(normalized) > 63 {
		normalized = normalized[:63]
		normalized = strings.TrimFunc(normalized, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r)
		})
		if normalized == "" {
			return "unknown"
		}
	}
	return normalized
}

func parseDownstreamFromLabels(labels map[string]string) []dependencyEdge {
	if len(labels) == 0 {
		return nil
	}

	out := make([]dependencyEdge, 0)
	for k, v := range labels {
		if !strings.HasPrefix(k, labelDependencyToPrefix) {
			continue
		}
		token := normalizeHexToken(strings.TrimPrefix(k, labelDependencyToPrefix))
		if token == "" {
			continue
		}
		out = append(out, dependencyEdge{
			TargetToken:  token,
			RelationCode: normalizeRelationCode(v),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].TargetToken != out[j].TargetToken {
			return out[i].TargetToken < out[j].TargetToken
		}
		return out[i].RelationCode < out[j].RelationCode
	})
	return out
}

func canDeleteFromUpstreamCount(upstreamCount int) bool {
	return upstreamCount == 0
}

func normalizeKind(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.Contains(s, "/") {
		parts := strings.Split(s, "/")
		s = parts[len(parts)-1]
	}

	// Normalize to alnum-only lowercase key.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	key := b.String()
	if key == "" {
		return ""
	}

	switch key {
	case "cluster", "clusters":
		return kindCluster
	case "storagerepository", "storagerepositories":
		return kindStorageRepository
	case "disasterpolicy", "disasterpolicies", "policy", "policies":
		return kindDisasterPolicy
	case "disasterconfig", "disasterconfigs", "dc", "dcs":
		return kindDisasterConfig
	case "disasterinstance", "disasterinstances", "instance", "instances":
		return kindDisasterInstance
	case "disastergroup", "disastergroups", "group", "groups":
		return kindDisasterGroup
	case "appbackup", "appbackups":
		return kindAppBackup
	case "apprestore", "apprestores":
		return kindAppRestore
	case "disasterdrill", "disasterdrills", "drill", "drills":
		return kindDisasterDrill
	case "disasterbackup", "disasterbackups":
		return kindDisasterBackup
	case "disasteroperation", "disasteroperations", "operation", "operations":
		return kindDisasterOperation
	case "datasync", "datasyncs":
		return kindDataSync
	case "resourcesync", "resourcesyncs":
		return kindResourceSync
	case "disasterjob", "disasterjobs", "job", "jobs":
		return kindDisasterJob
	default:
		return ""
	}
}

func normalizeHexToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 32 {
		out = out[:32]
	}
	return out
}
