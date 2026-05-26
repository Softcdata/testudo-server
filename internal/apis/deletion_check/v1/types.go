package deletioncheck

type DeletionCheckRequest struct {
	ResourceKind string `json:"resource_kind"`
	Name         string `json:"name"`
	Namespace    string `json:"namespace,omitempty"`
}

type ResourceRef struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	UID       string `json:"uid,omitempty"`
}

type DependencyRef struct {
	Kind         string `json:"kind,omitempty"`
	Name         string `json:"name,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	UID          string `json:"uid,omitempty"`
	RelationCode string `json:"relation_code,omitempty"`
	Token        string `json:"token,omitempty"`
	Unresolved   bool   `json:"unresolved,omitempty"`
}

type CleanupRef struct {
	Kind             string            `json:"kind"`
	Name             string            `json:"name,omitempty"`
	Namespace        string            `json:"namespace,omitempty"`
	UID              string            `json:"uid,omitempty"`
	Cluster          string            `json:"cluster,omitempty"`
	RelationCode     string            `json:"relation_code,omitempty"`
	Strategy         string            `json:"strategy,omitempty"`
	Resolved         bool              `json:"resolved"`
	Selector         map[string]string `json:"selector,omitempty"`
	UnresolvedReason string            `json:"unresolved_reason,omitempty"`
}

type CleanupPlan struct {
	HasCleanup       bool         `json:"has_cleanup"`
	FinalizerCleanup []CleanupRef `json:"finalizer_cleanup"`
	CascadeCleanup   []CleanupRef `json:"cascade_cleanup"`
}

type DeletionCheckResponse struct {
	Target      ResourceRef     `json:"target"`
	Upstream    []DependencyRef `json:"upstream"`
	Downstream  []DependencyRef `json:"downstream"`
	CleanupPlan CleanupPlan     `json:"cleanup_plan"`
	CanDelete   bool            `json:"can_delete"`
	Message     string          `json:"message"`
}
