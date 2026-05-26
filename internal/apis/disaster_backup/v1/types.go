package backup

import (
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DisasterBackupDTO struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Namespace         string                  `json:"namespace"`
	CreationTimestamp common.LocalTime        `json:"creationTimestamp"`
	Labels            map[string]string       `json:"labels,omitempty"`
	Annotations       map[string]string       `json:"annotations,omitempty"`
	Spec              DisasterBackupSpecDTO   `json:"spec"`
	Status            DisasterBackupStatusDTO `json:"status"`
}

type DisasterBackupStatusDTO struct {
	Conditions []common.LocalCondition        `json:"conditions,omitempty"`
	Phase      dapisv1.PhaseType              `json:"phase,omitempty"`
	Resources  map[string][]dapisv1.Resources `json:"resources,omitempty"`
	Workload   map[string][]dapisv1.Resources `json:"workload,omitempty"`
	UpdateTime common.LocalTime               `json:"updateTime"`
}

type DisasterBackupSpecDTO struct {
	LabelSelector  *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Namespace      string                `json:"namespace,omitempty"`
	DisasterConfig string                `json:"disasterConfig"`
	NewNamespace   string                `json:"newNamespace,omitempty"`
	NewStorageName string                `json:"newStorageName,omitempty"`
}

func ConvertToDisasterBackupDTO(item *dapisv1.DisasterBackup) DisasterBackupDTO {
	return DisasterBackupDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Labels:            item.Labels,
		Annotations:       item.Annotations,
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertStatusToDTO(status dapisv1.DisasterBackupStatus) DisasterBackupStatusDTO {
	return DisasterBackupStatusDTO{
		Conditions: common.NewLocalConditions(status.Conditions),
		Phase:      status.Phase,
		Resources:  status.Resources,
		Workload:   status.Workload,
		UpdateTime: common.NewLocalTime(status.UpdateTime),
	}
}

func ConvertSpecToDTO(spec dapisv1.DisasterBackupSpec) DisasterBackupSpecDTO {
	return DisasterBackupSpecDTO{
		LabelSelector:  spec.LabelSelector,
		Namespace:      spec.Namespace,
		DisasterConfig: spec.DisasterConfig,
		NewNamespace:   spec.NewNamespace,
		NewStorageName: spec.NewStorageName,
	}
}

// CreateDisasterBackupRequest defines the request body for creating a DisasterBackup
type CreateDisasterBackupRequest struct {
	Name           string                `json:"name" binding:"required"`
	LabelSelector  *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Namespace      string                `json:"namespace,omitempty"`
	DisasterConfig string                `json:"disasterConfig" binding:"required"`
	NewNamespace   string                `json:"newNamespace,omitempty"`
	NewStorageName string                `json:"newStorageName,omitempty"`
}

// ToCRD converts the request DTO to the Operator's DisasterBackupSpec
func (r *CreateDisasterBackupRequest) ToCRD() dapisv1.DisasterBackupSpec {
	return dapisv1.DisasterBackupSpec{
		LabelSelector:  r.LabelSelector,
		Namespace:      r.Namespace,
		DisasterConfig: r.DisasterConfig,
		NewNamespace:   r.NewNamespace,
		NewStorageName: r.NewStorageName,
	}
}

// UpdateDisasterBackupRequest defines the request body for updating a DisasterBackup
type UpdateDisasterBackupRequest struct {
	Name           string                `json:"name" binding:"required"`
	LabelSelector  *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Namespace      string                `json:"namespace,omitempty"`
	DisasterConfig string                `json:"disasterConfig,omitempty"`
	NewNamespace   string                `json:"newNamespace,omitempty"`
	NewStorageName string                `json:"newStorageName,omitempty"`
}

// MergeToCRD updates the existing DisasterBackupSpec with fields from the request
func (r *UpdateDisasterBackupRequest) MergeToCRD(spec *dapisv1.DisasterBackupSpec) {
	if r.LabelSelector != nil {
		spec.LabelSelector = r.LabelSelector
	}
	if r.Namespace != "" {
		spec.Namespace = r.Namespace
	}
	if r.DisasterConfig != "" {
		spec.DisasterConfig = r.DisasterConfig
	}
	if r.NewNamespace != "" {
		spec.NewNamespace = r.NewNamespace
	}
	if r.NewStorageName != "" {
		spec.NewStorageName = r.NewStorageName
	}
}
