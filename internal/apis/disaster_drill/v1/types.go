package drill

import (
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	instancev1 "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	"github.com/softcdata/testudo-server/internal/common"
)

// DisasterDrillDTO 是返回给前端的演练对象
type DisasterDrillDTO struct {
	// 基础信息
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	InstanceName      string            `json:"instanceName,omitempty"`
	GroupName         string            `json:"groupName,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	CreationTimestamp common.LocalTime  `json:"creationTimestamp"`

	// 配置
	TargetCluster    string                       `json:"targetCluster,omitempty"`
	NamespaceMapping map[string]string            `json:"namespaceMapping,omitempty"`
	SkipValidation   bool                         `json:"skipValidation"`
	Confirmed        bool                         `json:"confirmed"`
	WaitUntilReady   bool                         `json:"waitUntilReady"`
	Cleanup          bool                         `json:"cleanup"`
	RestorePolicy    *instancev1.RestorePolicyDTO `json:"restorePolicy,omitempty"`

	// 状态（不扁平化，统一走 status.state / status.reason / status.message）
	Status DisasterDrillStatusDTO `json:"status"`

	// 校验结果
	ValidationResults *DrillValidationDTO `json:"validationResults,omitempty"`

	// 执行步骤
	Steps []DrillStepDTO `json:"steps,omitempty"`

	// 容灾组演练进度
	GroupProgress *DisasterGroupStatusDTO `json:"groupProgress,omitempty"`
}

// DisasterDrillStatusDTO 演练状态详情
type DisasterDrillStatusDTO struct {
	State          string            `json:"state"` // Pending, Ready, Executing, Completed, Failed
	Reason         string            `json:"reason,omitempty"`
	RestoreMode    string            `json:"restoreMode,omitempty"`   // FullRestore
	OperationName  string            `json:"operationName,omitempty"` // 关联的 DisasterOperation
	CurrentStep    string            `json:"currentStep,omitempty"`
	Message        string            `json:"message,omitempty"`
	StartTime      *common.LocalTime `json:"startTime,omitempty"`
	ReadyTime      *common.LocalTime `json:"readyTime,omitempty"`
	ExecutionTime  *common.LocalTime `json:"executionTime,omitempty"`
	CompletionTime *common.LocalTime `json:"completionTime,omitempty"`
}

// DisasterGroupStatusDTO 容灾组演练进度
type DisasterGroupStatusDTO struct {
	TotalLevels       int              `json:"totalLevels"`
	CurrentLevelIndex int              `json:"currentLevelIndex"`
	LevelStatuses     []LevelStatusDTO `json:"levelStatuses,omitempty"`
}

// LevelStatusDTO 容灾组层级状态
type LevelStatusDTO struct {
	Index     int      `json:"index"`
	State     string   `json:"state"`
	Instances []string `json:"instances,omitempty"`
}

// DrillValidationDTO 校验结果
type DrillValidationDTO struct {
	InstanceValid        bool              `json:"instanceValid"`
	ClusterReachable     bool              `json:"clusterReachable"`
	BackupAvailable      bool              `json:"backupAvailable"`
	LastDataSyncTime     *common.LocalTime `json:"lastDataSyncTime,omitempty"`
	LastResourceSyncTime *common.LocalTime `json:"lastResourceSyncTime,omitempty"`
}

// DrillStepDTO 执行步骤
type DrillStepDTO struct {
	Name           string            `json:"name"`
	State          string            `json:"state"` // Pending, Running, Completed, Failed
	StartTime      *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime *common.LocalTime `json:"completionTime,omitempty"`
	Message        string            `json:"message,omitempty"`
}

// CreateDrillRequest 创建演练请求
type CreateDrillRequest struct {
	// 二选一：关联的容灾实例名称 OR 容灾组名称
	InstanceName string `json:"instanceName,omitempty"`
	GroupName    string `json:"groupName,omitempty"`

	// 可选：自定义演练名称 (不指定则自动生成)
	Name string `json:"name,omitempty"`

	// 可选：演练目标集群 (不指定则使用 Instance 的 secondaryCluster)
	TargetCluster string `json:"targetCluster,omitempty"`

	// 可选：命名空间映射 (不指定则使用原始命名空间)
	NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`

	// 可选：跳过前置校验
	SkipValidation bool `json:"skipValidation,omitempty"`

	// 可选：命名空间 (CR 存储位置，默认 disaster-system)
	Namespace string `json:"namespace,omitempty"`

	// 可选：是否等待 Pod 就绪
	WaitUntilReady bool `json:"waitUntilReady,omitempty"`

	// 可选：演练级 restorePolicy 覆盖。
	// 当前主要用于资源定制化修改与 bulk 操作，未提供时默认继承实例配置。
	RestorePolicy *instancev1.RestorePolicyRequest `json:"restorePolicy,omitempty"`
}

// ProtectedNamespacesDTO 演练前置命名空间查询返回
type ProtectedNamespacesDTO struct {
	// Instance or Group
	Type string `json:"type"`

	// 二选一入参回显
	InstanceName string `json:"instanceName,omitempty"`
	GroupName    string `json:"groupName,omitempty"`

	// 聚合后的受保护命名空间（去重）
	Namespaces []string `json:"namespaces"`

	// 仅 Group 查询场景可选返回
	MissingInstances []string `json:"missingInstances,omitempty"`
}

// ConvertToDisasterDrillDTO 将 CRD 转换为 DTO
func ConvertToDisasterDrillDTO(item *dapisv1.DisasterDrill) DisasterDrillDTO {
	dto := DisasterDrillDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		InstanceName:      item.Spec.InstanceName,
		GroupName:         item.Spec.GroupName,
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		TargetCluster:     item.Status.TargetCluster,
		NamespaceMapping:  item.Spec.NamespaceMapping,
		SkipValidation:    item.Spec.SkipValidation,
		Confirmed:         item.Spec.Confirmed,
		WaitUntilReady:    item.Spec.WaitUntilReady,
		Cleanup:           item.Spec.CleanUp,
		RestorePolicy:     instancev1.ConvertRestorePolicyDTO(item.Spec.RestorePolicy),
		Status: DisasterDrillStatusDTO{
			State:          string(item.Status.State),
			Reason:         item.Status.Reason,
			RestoreMode:    string(item.Status.RestoreMode),
			OperationName:  item.Status.OperationName,
			CurrentStep:    item.Status.CurrentStep,
			Message:        item.Status.Message,
			StartTime:      common.NewLocalTimePtr(item.Status.StartTime),
			ReadyTime:      common.NewLocalTimePtr(item.Status.ReadyTime),
			ExecutionTime:  common.NewLocalTimePtr(item.Status.ExecutionTime),
			CompletionTime: common.NewLocalTimePtr(item.Status.CompletionTime),
		},
	}

	// 转换校验结果
	if item.Status.ValidationResults != nil {
		dto.ValidationResults = &DrillValidationDTO{
			InstanceValid:        item.Status.ValidationResults.InstanceValid,
			ClusterReachable:     item.Status.ValidationResults.ClusterReachable,
			BackupAvailable:      item.Status.ValidationResults.BackupAvailable,
			LastDataSyncTime:     common.NewLocalTimePtr(item.Status.ValidationResults.LastDataSyncTime),
			LastResourceSyncTime: common.NewLocalTimePtr(item.Status.ValidationResults.LastResourceSyncTime),
		}
	}

	// 转换执行步骤
	if len(item.Status.Steps) > 0 {
		dto.Steps = make([]DrillStepDTO, 0, len(item.Status.Steps))
		for _, step := range item.Status.Steps {
			dto.Steps = append(dto.Steps, DrillStepDTO{
				Name:           step.Name,
				State:          step.State,
				StartTime:      common.NewLocalTimePtr(step.StartTime),
				CompletionTime: common.NewLocalTimePtr(step.CompletionTime),
				Message:        step.Message,
			})
		}
	}

	// 转换 GroupProgress
	if item.Status.GroupProgress != nil {
		gp := item.Status.GroupProgress
		dto.GroupProgress = &DisasterGroupStatusDTO{
			TotalLevels:       gp.TotalLevels,
			CurrentLevelIndex: gp.CurrentLevelIndex,
		}
		if len(gp.LevelStatuses) > 0 {
			dto.GroupProgress.LevelStatuses = make([]LevelStatusDTO, 0, len(gp.LevelStatuses))
			for _, lvl := range gp.LevelStatuses {
				dto.GroupProgress.LevelStatuses = append(dto.GroupProgress.LevelStatuses, LevelStatusDTO{
					Index:     lvl.Index,
					State:     lvl.State,
					Instances: lvl.Instances,
				})
			}
		}
	}

	return dto
}

// ToCRD 将请求转换为 CRD Spec
func (r *CreateDrillRequest) ToCRD(restorePolicy *dapisv1.RestorePolicy) dapisv1.DisasterDrillSpec {
	return dapisv1.DisasterDrillSpec{
		InstanceName:     r.InstanceName,
		GroupName:        r.GroupName,
		TargetCluster:    r.TargetCluster,
		NamespaceMapping: r.NamespaceMapping,
		SkipValidation:   r.SkipValidation,
		WaitUntilReady:   r.WaitUntilReady,
		RestorePolicy:    restorePolicy,
		Confirmed:        false, // 创建时不自动确认
	}
}

// GenerateDrillName 生成演练名称
func GenerateDrillName(baseName string) string {
	// Truncate baseName if easier (keep last 30 chars or first 30 chars?)
	// Usually prefix is more meaningful, but suffix ensures uniqueness.
	// Let's cap baseName at 30 chars.
	if len(baseName) > 30 {
		baseName = baseName[:30]
	}
	// Format: drill-<base>-<short-timestamp>
	// timestamp: 01021504 (MMDDHHMM) - 8 chars
	// prefix: drill- (6 chars)
	// Total: 6 + 30 + 1 + 8 = 45 chars max. Safe for labels.
	return "drill-" + baseName + "-" + time.Now().Format("01021504")
}
