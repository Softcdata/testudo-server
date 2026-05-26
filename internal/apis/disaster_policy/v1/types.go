package policy

import (
	"errors"
	"fmt"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ExternalPolicyTypeAutoBackup     = "AutoBackup"
	ExternalPolicyTypeSyncPolicy     = "SyncPolicy"
	legacyExternalPolicyTypeData     = "DataSync"
	legacyExternalPolicyTypeResource = "ResourceSync"
)

type policyValidationError struct {
	message string
}

func (e policyValidationError) Error() string {
	return e.message
}

func newPolicyValidationError(format string, args ...any) error {
	return policyValidationError{message: fmt.Sprintf(format, args...)}
}

func isPolicyValidationError(err error) bool {
	var target policyValidationError
	return errors.As(err, &target)
}

// DisasterPolicyDTO is the data transfer object for DisasterPolicy
type DisasterPolicyDTO struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	Namespace         string                  `json:"namespace"`
	Labels            map[string]string       `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime        `json:"creation_timestamp"`
	Spec              DisasterPolicySpecDTO   `json:"spec"`
	Status            DisasterPolicyStatusDTO `json:"status"`
}

type DisasterPolicySpecDTO struct {
	Type        string              `json:"type"`
	Schedule    string              `json:"schedule"`
	StartTime   *common.LocalTime   `json:"startTime,omitempty"`
	TTL         *metav1.Duration    `json:"ttl,omitempty"`
	Description string              `json:"description,omitempty"`
	State       dapisv1.PolicyState `json:"state"`
}

type DisasterPolicyNameDTO struct {
	Name     string           `json:"name"`
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Schedule string           `json:"schedule,omitempty"`
	TTL      *metav1.Duration `json:"ttl,omitempty"`
}

type DisasterPolicyStatusDTO struct {
	Phase             dapisv1.PolicyPhase `json:"phase,omitempty"`
	LastExecutionTime *common.LocalTime   `json:"lastExecutionTime,omitempty"`
	NextExecutionTime *common.LocalTime   `json:"nextExecutionTime,omitempty"`
	Reason            string              `json:"reason,omitempty"`
	Message           string              `json:"message,omitempty"`
}

func ConvertToDisasterPolicyDTO(item *dapisv1.DisasterPolicy) DisasterPolicyDTO {
	return DisasterPolicyDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Labels:            normalizedPolicyLabels(item.Labels, item.Spec.Type),
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.DisasterPolicySpec) DisasterPolicySpecDTO {
	return DisasterPolicySpecDTO{
		Type:        externalPolicyType(spec.Type),
		Schedule:    spec.Schedule,
		StartTime:   common.NewLocalTimePtr(spec.StartTime),
		TTL:         spec.TTL,
		Description: spec.Description,
		State:       spec.State,
	}
}

func ConvertStatusToDTO(status dapisv1.DisasterPolicyStatus) DisasterPolicyStatusDTO {
	return DisasterPolicyStatusDTO{
		Phase:             status.Phase,
		LastExecutionTime: common.NewLocalTimePtr(status.LastExecutionTime),
		NextExecutionTime: common.NewLocalTimePtr(status.NextExecutionTime),
		Reason:            status.Reason,
		Message:           status.Message,
	}
}

func normalizedPolicyLabels(labels map[string]string, policyType dapisv1.PolicyType) map[string]string {
	if labels == nil {
		return nil
	}

	out := make(map[string]string, len(labels))
	for k, v := range labels {
		out[k] = v
	}

	if externalPolicyType(policyType) == ExternalPolicyTypeSyncPolicy {
		out[metadata.LabelDisasterPolicyType] = ExternalPolicyTypeSyncPolicy
	}

	return out
}

// CreateDisasterPolicyRequest defines the request body for creating a DisasterPolicy
type CreateDisasterPolicyRequest struct {
	Name        string              `json:"name" binding:"required,max=50,min=1"`
	Type        string              `json:"type" binding:"required"`
	Schedule    string              `json:"schedule" binding:"required"`
	StartTime   *metav1.Time        `json:"startTime,omitempty"`
	TTL         *metav1.Duration    `json:"ttl,omitempty"`
	Description string              `json:"description,omitempty" binding:"max=200"`
	State       dapisv1.PolicyState `json:"state" binding:"required ,oneof=Enabled Disabled"`
}

// ToCRD converts the request DTO to the Operator's DisasterPolicySpec
func (r *CreateDisasterPolicyRequest) ToCRD() (dapisv1.DisasterPolicySpec, error) {
	policyType, err := policyTypeFromCreateRequest(r.Type)
	if err != nil {
		return dapisv1.DisasterPolicySpec{}, err
	}
	if err := validatePolicyTTL(policyType, r.TTL); err != nil {
		return dapisv1.DisasterPolicySpec{}, err
	}
	return dapisv1.DisasterPolicySpec{
		Type:        policyType,
		Schedule:    r.Schedule,
		StartTime:   r.StartTime,
		TTL:         r.TTL,
		Description: r.Description,
		State:       r.State,
	}, nil
}

// UpdateDisasterPolicyRequest defines the request body for updating a DisasterPolicy
type UpdateDisasterPolicyRequest struct {
	Name        string               `json:"name" binding:"required"`
	Type        string               `json:"type,omitempty"`
	Schedule    string               `json:"schedule,omitempty"`
	StartTime   *metav1.Time         `json:"startTime,omitempty"`
	TTL         *metav1.Duration     `json:"ttl,omitempty"`
	ClearTTL    *bool                `json:"clearTTL,omitempty"`
	Description string               `json:"description,omitempty"`
	State       *dapisv1.PolicyState `json:"state,omitempty"`
}

// MergeToCRD updates the existing DisasterPolicySpec with fields from the request
func (r *UpdateDisasterPolicyRequest) MergeToCRD(spec *dapisv1.DisasterPolicySpec) error {
	if strings.TrimSpace(r.Type) != "" {
		policyType, err := policyTypeFromUpdateRequest(r.Type, spec.Type)
		if err != nil {
			return err
		}
		spec.Type = policyType
	}
	if r.Schedule != "" {
		spec.Schedule = r.Schedule
	}
	if r.StartTime != nil {
		spec.StartTime = r.StartTime
	}
	if r.ClearTTL != nil && *r.ClearTTL {
		spec.TTL = nil
	} else if r.TTL != nil {
		spec.TTL = r.TTL
	}
	if r.Description != "" {
		spec.Description = r.Description
	}
	if r.State != nil {
		spec.State = *r.State
	}
	return validatePolicyTTL(spec.Type, spec.TTL)
}

func validatePolicyTTL(policyType dapisv1.PolicyType, ttl *metav1.Duration) error {
	if ttl == nil {
		return nil
	}
	if policyType != dapisv1.PolicyTypeAutoBackup {
		return newPolicyValidationError("ttl is only supported for %s policies", ExternalPolicyTypeAutoBackup)
	}
	if ttl.Duration <= 0 {
		return newPolicyValidationError("ttl must be greater than 0")
	}
	return nil
}

func externalPolicyType(policyType dapisv1.PolicyType) string {
	switch policyType {
	case dapisv1.PolicyTypeAutoBackup:
		return ExternalPolicyTypeAutoBackup
	case dapisv1.PolicyTypeDataSync, dapisv1.PolicyTypeResourceSync:
		return ExternalPolicyTypeSyncPolicy
	default:
		return string(policyType)
	}
}

func matchesExternalPolicyTypeFilter(actual dapisv1.PolicyType, expected string) bool {
	switch strings.TrimSpace(expected) {
	case "":
		return true
	case ExternalPolicyTypeSyncPolicy:
		return actual == dapisv1.PolicyTypeDataSync || actual == dapisv1.PolicyTypeResourceSync
	case ExternalPolicyTypeAutoBackup:
		return actual == dapisv1.PolicyTypeAutoBackup
	case legacyExternalPolicyTypeData:
		return actual == dapisv1.PolicyTypeDataSync
	case legacyExternalPolicyTypeResource:
		return actual == dapisv1.PolicyTypeResourceSync
	default:
		return false
	}
}

func policyTypeFromCreateRequest(raw string) (dapisv1.PolicyType, error) {
	switch strings.TrimSpace(raw) {
	case ExternalPolicyTypeAutoBackup:
		return dapisv1.PolicyTypeAutoBackup, nil
	case ExternalPolicyTypeSyncPolicy, legacyExternalPolicyTypeData, legacyExternalPolicyTypeResource:
		// SyncPolicy is the unified server-facing type. The operator CRD still
		// stores a concrete sync policy type, and the controller only consumes
		// schedule/state, so a newly created SyncPolicy defaults to DataSync in
		// the CRD for compatibility. Existing sync policies keep their concrete
		// type on update; see policyTypeFromUpdateRequest.
		return dapisv1.PolicyTypeDataSync, nil
	default:
		return "", newPolicyValidationError("type must be one of %s or %s", ExternalPolicyTypeAutoBackup, ExternalPolicyTypeSyncPolicy)
	}
}

func policyTypeFromUpdateRequest(raw string, current dapisv1.PolicyType) (dapisv1.PolicyType, error) {
	switch strings.TrimSpace(raw) {
	case ExternalPolicyTypeAutoBackup:
		return dapisv1.PolicyTypeAutoBackup, nil
	case ExternalPolicyTypeSyncPolicy:
		if current == dapisv1.PolicyTypeDataSync || current == dapisv1.PolicyTypeResourceSync {
			return current, nil
		}
		return dapisv1.PolicyTypeDataSync, nil
	case legacyExternalPolicyTypeData:
		return dapisv1.PolicyTypeDataSync, nil
	case legacyExternalPolicyTypeResource:
		return dapisv1.PolicyTypeResourceSync, nil
	default:
		return "", newPolicyValidationError("type must be one of %s or %s", ExternalPolicyTypeAutoBackup, ExternalPolicyTypeSyncPolicy)
	}
}
