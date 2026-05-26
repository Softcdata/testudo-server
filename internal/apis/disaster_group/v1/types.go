package v1

import (
	"reflect"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	instanceapi "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

// DisasterGroupDTO 响应对象 (含聚合状态)
type DisasterGroupDTO struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	Description       string                      `json:"description"` // 来自 Annotation
	CreationTimestamp common.LocalTime            `json:"creation_timestamp"`
	Levels            [][]string                  `json:"levels"`
	Policy            dapisv1.DisasterGroupPolicy `json:"policy"`
	Status            DisasterGroupStatusDTO      `json:"status"`
	// 新增：该组管理的容灾实例列表（详情接口返回）
	Instances []InstanceSummaryDTO `json:"instances,omitempty"`
}

type DisasterGroupStatusDTO struct {
	TotalInstances      int                     `json:"totalInstances"`
	ReadyInstances      int                     `json:"readyInstances"`
	Reason              string                  `json:"reason,omitempty"`
	Message             string                  `json:"message,omitempty"`
	Summary             string                  `json:"summary"` // e.g. "3 Levels, 10 Instances"
	FsmState            string                  `json:"fsmState"`
	AvailableOperations []string                `json:"availableOperations"`
	Conditions          []common.LocalCondition `json:"conditions,omitempty"`
}

// InstanceSummaryDTO 容灾实例摘要（用于组详情展示）
type InstanceSummaryDTO struct {
	Name              string   `json:"name"`
	Namespaces        []string `json:"namespaces"` // 保护的命名空间列表
	FsmState          string   `json:"fsmState"`   // Pending, Protected, Paused, Failed, etc.
	Reason            string   `json:"reason,omitempty"`
	Message           string   `json:"message,omitempty"`
	PrimaryCluster    string   `json:"primaryCluster"`
	SecondaryCluster  string   `json:"secondaryCluster"`
	StorageRepository string   `json:"storageRepository,omitempty"`
	Level             int      `json:"level"` // 该实例所在的层级 (1-indexed)
}

// CreateDisasterGroupRequest 创建请求
type CreateDisasterGroupRequest struct {
	Name        string                      `json:"name,required"`
	Description string                      `json:"description"`
	Levels      [][]string                  `json:"levels,required"`
	Policy      dapisv1.DisasterGroupPolicy `json:"policy"`
}

func (r *CreateDisasterGroupRequest) ToCRD() dapisv1.DisasterGroupSpec {
	return dapisv1.DisasterGroupSpec{
		Levels: r.Levels,
		Policy: r.Policy,
	}
}

// UpdateDisasterGroupRequest 更新请求
type UpdateDisasterGroupRequest struct {
	Description *string                      `json:"description,omitempty"`
	Levels      [][]string                   `json:"levels,omitempty"`
	Policy      *dapisv1.DisasterGroupPolicy `json:"policy,omitempty"`
}

// ConvertToDisasterGroupDTO 转换函数
func ConvertToDisasterGroupDTO(item *dapisv1.DisasterGroup) DisasterGroupDTO {
	desc := ""
	if item.Annotations != nil {
		desc = item.Annotations["testudo.softcdata.com/description"]
	}

	return DisasterGroupDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Description:       desc,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Levels:            item.Spec.Levels,
		Policy:            item.Spec.Policy,
		Status: DisasterGroupStatusDTO{
			TotalInstances: item.Status.TotalInstances,
			ReadyInstances: item.Status.ReadyInstances,
			Reason:         getGroupStatusStringField(&item.Status, "Reason"),
			Message:        getGroupStatusStringField(&item.Status, "Message"),
			Conditions:     common.NewLocalConditions(item.Status.Conditions),
			// Summary 可以在 Handler 中计算
		},
	}
}

func getGroupStatusStringField(status any, fieldName string) string {
	if status == nil {
		return ""
	}
	v := reflect.ValueOf(status)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return ""
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return ""
	}
	field := elem.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

// InstancePickerItemDTO 实例选择器条目（轻量 DTO，供前端构建选择器使用）
// 仅包含选择器场景必要字段，不含 Config / Storage / SyncStatus 等重型字段
// Description 来自 metadata.annotations["testudo.softcdata.com/description"]，而非 k8s labels
type InstancePickerItemDTO struct {
	Name        string   `json:"name"`
	Namespaces  []string `json:"namespaces,omitempty"`  // spec.namespaces，业务保护的应用命名空间
	Description string   `json:"description,omitempty"` // 说明标签（来自 Annotation）
	// Status 标准状态对象，前端统一读取 status.state/reason/message。
	Status GroupMemberStatusDTO `json:"status"`
	// FsmState 兼容字段，保留旧读取路径。
	FsmState string `json:"fsmState"`
}

// GroupMemberInstanceDTO 容灾组已选实例摘要（供编辑页展示用）
type GroupMemberInstanceDTO struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"` // 来自 metadata.annotations["testudo.softcdata.com/description"]
	Namespaces  []string `json:"namespaces,omitempty"`  // spec.namespaces，业务保护的应用命名空间
	// Status 标准状态对象，前端统一读取 status.state/reason/message。
	Status GroupMemberStatusDTO `json:"status"`
	// FsmState 兼容字段，保留旧读取路径。
	FsmState string `json:"fsmState"`
}

// GroupMemberStatusDTO 容灾组成员实例状态
type GroupMemberStatusDTO struct {
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// ============ 容灾组操作 Watch DTO ============

// StepStatusDTO 单个步骤状态
type StepStatusDTO struct {
	Name           string            `json:"name"`
	State          string            `json:"state"`
	StartTime      *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime *common.LocalTime `json:"completionTime,omitempty"`
	Message        string            `json:"message,omitempty"`
}

// LevelStatusDTO 组操作中单层的执行状态
type LevelStatusDTO struct {
	Index           int               `json:"index"`
	Instances       []string          `json:"instances"`
	State           string            `json:"state"`
	StartTime       *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime  *common.LocalTime `json:"completionTime,omitempty"`
	FailedInstances []string          `json:"failedInstances,omitempty"`
}

// GroupStatusDTO 组级别整体执行状态
type GroupStatusDTO struct {
	TotalLevels       int              `json:"totalLevels"`
	CurrentLevelIndex int              `json:"currentLevelIndex"`
	LevelStatuses     []LevelStatusDTO `json:"levelStatuses,omitempty"`
}

// DisasterOperationDTO 容灾操作 DTO，用于 Watch 事件推送
type DisasterOperationDTO struct {
	Name              string                            `json:"name"`
	Namespace         string                            `json:"namespace"`
	GroupName         string                            `json:"groupName"`
	OperationType     string                            `json:"operationType"`
	State             string                            `json:"state"`
	Reason            string                            `json:"reason,omitempty"`
	CurrentStep       string                            `json:"currentStep,omitempty"`
	Message           string                            `json:"message,omitempty"`
	Steps             []StepStatusDTO                   `json:"steps,omitempty"`
	AutoCancel        *instanceapi.AutoCancelSummaryDTO `json:"autoCancel,omitempty"`
	GroupStatus       *GroupStatusDTO                   `json:"groupStatus,omitempty"`
	StartTime         *common.LocalTime                 `json:"startTime,omitempty"`
	CompletionTime    *common.LocalTime                 `json:"completionTime,omitempty"`
	CreationTimestamp common.LocalTime                  `json:"creationTimestamp"`
}

// ConvertToDisasterOperationDTO 将 CRD 对象转换为 DTO
func ConvertToDisasterOperationDTO(op *dapisv1.DisasterOperation) DisasterOperationDTO {
	dto := DisasterOperationDTO{
		Name:              op.Name,
		Namespace:         op.Namespace,
		GroupName:         op.Spec.GroupName,
		OperationType:     string(op.Spec.OperationType),
		State:             string(op.Status.State),
		Reason:            op.Status.Reason,
		CurrentStep:       op.Status.CurrentStep,
		Message:           op.Status.Message,
		AutoCancel:        instanceapi.ConvertToAutoCancelSummary(op),
		StartTime:         common.NewLocalTimePtr(op.Status.StartTime),
		CompletionTime:    common.NewLocalTimePtr(op.Status.CompletionTime),
		CreationTimestamp: common.NewLocalTime(op.CreationTimestamp),
	}

	// 转换步骤状态
	if len(op.Status.Steps) > 0 {
		dto.Steps = make([]StepStatusDTO, len(op.Status.Steps))
		for i, s := range op.Status.Steps {
			dto.Steps[i] = StepStatusDTO{
				Name:           s.Name,
				State:          s.State,
				StartTime:      common.NewLocalTimePtr(s.StartTime),
				CompletionTime: common.NewLocalTimePtr(s.CompletionTime),
				Message:        s.Message,
			}
		}
	}

	// 转换组级别状态
	if op.Status.GroupStatus != nil {
		gs := op.Status.GroupStatus
		groupDTO := &GroupStatusDTO{
			TotalLevels:       gs.TotalLevels,
			CurrentLevelIndex: gs.CurrentLevelIndex,
		}
		if len(gs.LevelStatuses) > 0 {
			groupDTO.LevelStatuses = make([]LevelStatusDTO, len(gs.LevelStatuses))
			for i, ls := range gs.LevelStatuses {
				groupDTO.LevelStatuses[i] = LevelStatusDTO{
					Index:           ls.Index,
					Instances:       ls.Instances,
					State:           ls.State,
					StartTime:       common.NewLocalTimePtr(ls.StartTime),
					CompletionTime:  common.NewLocalTimePtr(ls.CompletionTime),
					FailedInstances: ls.FailedInstances,
				}
			}
		}
		dto.GroupStatus = groupDTO
	}

	return dto
}
