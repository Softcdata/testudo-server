package instance

import (
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

type StepStatusDTO struct {
	Name           string            `json:"name"`
	State          string            `json:"state"`
	StartTime      *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime *common.LocalTime `json:"completionTime,omitempty"`
	Message        string            `json:"message,omitempty"`
}

type LevelStatusDTO struct {
	Index           int               `json:"index"`
	Instances       []string          `json:"instances"`
	State           string            `json:"state"`
	StartTime       *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime  *common.LocalTime `json:"completionTime,omitempty"`
	FailedInstances []string          `json:"failedInstances,omitempty"`
}

type GroupStatusDTO struct {
	TotalLevels       int              `json:"totalLevels"`
	CurrentLevelIndex int              `json:"currentLevelIndex"`
	LevelStatuses     []LevelStatusDTO `json:"levelStatuses,omitempty"`
}

type RoleStatusDTO struct {
	PrimaryCluster   string `json:"primaryCluster"`
	SecondaryCluster string `json:"secondaryCluster"`
}

type OperationDetailDTO struct {
	Name              string                `json:"name"`
	UID               string                `json:"uid"`
	Namespace         string                `json:"namespace"`
	OwnerKind         string                `json:"ownerKind,omitempty"`
	OwnerName         string                `json:"ownerName,omitempty"`
	OperationType     string                `json:"operationType"`
	State             string                `json:"state"`
	Reason            string                `json:"reason,omitempty"`
	CurrentStep       string                `json:"currentStep,omitempty"`
	Message           string                `json:"message,omitempty"`
	Steps             []StepStatusDTO       `json:"steps,omitempty"`
	AutoCancel        *AutoCancelSummaryDTO `json:"autoCancel,omitempty"`
	RoleStatus        *RoleStatusDTO        `json:"roleStatus,omitempty"`
	GroupStatus       *GroupStatusDTO       `json:"groupStatus,omitempty"`
	StartTime         *common.LocalTime     `json:"startTime,omitempty"`
	CompletionTime    *common.LocalTime     `json:"completionTime,omitempty"`
	CreationTimestamp common.LocalTime      `json:"creationTimestamp"`
}

func ConvertStepStatuses(steps []dapisv1.StepStatus) []StepStatusDTO {
	if len(steps) == 0 {
		return nil
	}

	out := make([]StepStatusDTO, len(steps))
	for i, step := range steps {
		out[i] = StepStatusDTO{
			Name:           step.Name,
			State:          step.State,
			StartTime:      common.NewLocalTimePtr(step.StartTime),
			CompletionTime: common.NewLocalTimePtr(step.CompletionTime),
			Message:        step.Message,
		}
	}
	return out
}

func ConvertGroupOperationStatus(groupStatus *dapisv1.GroupStatus) *GroupStatusDTO {
	if groupStatus == nil {
		return nil
	}

	out := &GroupStatusDTO{
		TotalLevels:       groupStatus.TotalLevels,
		CurrentLevelIndex: groupStatus.CurrentLevelIndex,
	}
	if len(groupStatus.LevelStatuses) == 0 {
		return out
	}

	out.LevelStatuses = make([]LevelStatusDTO, len(groupStatus.LevelStatuses))
	for i, level := range groupStatus.LevelStatuses {
		out.LevelStatuses[i] = LevelStatusDTO{
			Index:           level.Index,
			Instances:       append([]string(nil), level.Instances...),
			State:           level.State,
			StartTime:       common.NewLocalTimePtr(level.StartTime),
			CompletionTime:  common.NewLocalTimePtr(level.CompletionTime),
			FailedInstances: append([]string(nil), level.FailedInstances...),
		}
	}
	return out
}

func ConvertRoleStatus(roleStatus *dapisv1.RoleStatus) *RoleStatusDTO {
	if roleStatus == nil {
		return nil
	}

	return &RoleStatusDTO{
		PrimaryCluster:   roleStatus.PrimaryCluster,
		SecondaryCluster: roleStatus.SecondaryCluster,
	}
}

func InferOperationOwner(op *dapisv1.DisasterOperation) (string, string) {
	if op == nil {
		return "", ""
	}
	if op.Spec.GroupName != "" {
		return "DisasterGroup", op.Spec.GroupName
	}
	if op.Spec.InstanceName != "" {
		return "DisasterInstance", op.Spec.InstanceName
	}
	return "", ""
}

func ConvertToOperationDetailDTO(op *dapisv1.DisasterOperation) OperationDetailDTO {
	if op == nil {
		return OperationDetailDTO{}
	}

	ownerKind, ownerName := InferOperationOwner(op)

	return OperationDetailDTO{
		Name:              op.Name,
		UID:               string(op.UID),
		Namespace:         op.Namespace,
		OwnerKind:         ownerKind,
		OwnerName:         ownerName,
		OperationType:     string(op.Spec.OperationType),
		State:             string(op.Status.State),
		Reason:            op.Status.Reason,
		CurrentStep:       op.Status.CurrentStep,
		Message:           op.Status.Message,
		Steps:             ConvertStepStatuses(op.Status.Steps),
		AutoCancel:        ConvertToAutoCancelSummary(op),
		RoleStatus:        ConvertRoleStatus(op.Status.RoleStatus),
		GroupStatus:       ConvertGroupOperationStatus(op.Status.GroupStatus),
		StartTime:         common.NewLocalTimePtr(op.Status.StartTime),
		CompletionTime:    common.NewLocalTimePtr(op.Status.CompletionTime),
		CreationTimestamp: common.NewLocalTime(op.CreationTimestamp),
	}
}
