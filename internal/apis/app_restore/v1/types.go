package apprestore

import (
	"encoding/json"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/softcdata/testudo-server/internal/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AppRestoreDescriptionAnnotation = "testudo.softcdata.com/app-restore-description"

// CreateAppRestoreRequest defines the request body for creating an AppRestore
type CreateAppRestoreRequest struct {
	Name                string            `json:"name" binding:"required"`
	BackupSource        string            `json:"backupSource" binding:"required"`
	Cluster             string            `json:"cluster" binding:"required"`
	CleanVolumes        bool              `json:"cleanVolumes,omitempty"`
	StorageClassMapping map[string]string `json:"storageClassMapping,omitempty"`
	SCMapping           map[string]string `json:"scMapping,omitempty"`
	IngressClassMapping map[string]string `json:"ingressClassMapping,omitempty"`
	ScaleToZeroList     []string          `json:"scaleToZeroList,omitempty"`
	StandbyList         []string          `json:"standbyList,omitempty"`
	TrafficlessImage    string            `json:"trafficlessImage,omitempty"` // For Scheme A: sterilize pods with this image

	Description            string           `json:"description,omitempty"`            // 描述
	ExistingResourcePolicy string           `json:"existingResourcePolicy,omitempty"` // 资源冲突策略 (none, update)
	Timeout                *metav1.Duration `json:"timeout,omitempty"`                // 操作超时时间

	// Velero common parameters
	BackupName              string                `json:"backupName"`
	IncludedNamespaces      []string              `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string              `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string              `json:"includedResources,omitempty"`
	ExcludedResources       []string              `json:"excludedResources,omitempty"`
	LabelSelector           *metav1.LabelSelector `json:"labelSelector,omitempty"`
	NamespaceMapping        map[string]string     `json:"namespaceMapping,omitempty"`
	RestorePVs              *bool                 `json:"restorePVs,omitempty"`
	PreserveNodePorts       *bool                 `json:"preserveNodePorts,omitempty"`
	IncludeClusterResources *bool                 `json:"includeClusterResources,omitempty"`

	// UploaderConfig parameters
	WriteSparseFiles      *bool `json:"writeSparseFiles,omitempty"`
	ParallelFilesDownload *int  `json:"parallelFilesDownload,omitempty"`

	Hooks *velerov1.RestoreHooks `json:"hooks,omitempty"`
}

// ValidateRestorePreflightRequest defines the request body for restore preflight validation.
type ValidateRestorePreflightRequest struct {
	BackupSource  string `json:"backupSource" binding:"required"`
	TargetCluster string `json:"targetCluster" binding:"required"`
	WaitSeconds   int    `json:"waitSeconds,omitempty"`
}

// ResourceModifierDTO defines the DTO for resource modification rules

// ToCRD converts the request DTO to the Operator's AppRestoreSpec
func (r *CreateAppRestoreRequest) ToCRD() dapisv1.AppRestoreSpec {
	spec := dapisv1.AppRestoreSpec{
		BackupSource: r.BackupSource,
		Cluster:      r.Cluster,
		Template: velerov1.RestoreSpec{
			BackupName:              r.BackupName,
			IncludedNamespaces:      r.IncludedNamespaces,
			ExcludedNamespaces:      r.ExcludedNamespaces,
			IncludedResources:       r.IncludedResources,
			ExcludedResources:       r.ExcludedResources,
			LabelSelector:           r.LabelSelector,
			NamespaceMapping:        r.NamespaceMapping,
			RestorePVs:              r.RestorePVs,
			PreserveNodePorts:       r.PreserveNodePorts,
			IncludeClusterResources: r.IncludeClusterResources,
			ExistingResourcePolicy:  velerov1.PolicyType(r.ExistingResourcePolicy),
			Hooks:                   restoreHooksValue(r.Hooks),
			UploaderConfig: &velerov1.UploaderConfigForRestore{
				WriteSparseFiles: r.WriteSparseFiles,
				ParallelFilesDownload: func() int {
					if r.ParallelFilesDownload != nil {
						return *r.ParallelFilesDownload
					}
					return 0
				}(),
			},
		},
	}
	// 用户反馈: 超时是写入到	ItemOperationTimeout metav1.Duration `json:"itemOperationTimeout,omitempty"` 这里的
	// 修正: 将 Timeout 映射到 Template.ItemOperationTimeout，而不是 CRD 根级的 Timeout (如果有的话)
	// AppRestoreSpec 定义中可能有 Timeout 字段用于控制整体流程，但用户特别指出是 ItemOperationTimeout。
	// 这里我们优先满足用户需求，将 API 请求的 "timeout" 字段映射到 Velero RestoreSpec 的 "itemOperationTimeout"。
	// 同时，为了保持兼容性或完整性，如果 AppRestoreSpec 也有 Timeout，是否也需要设置？
	// 假设用户指的是 Velero 的 itemOperationTimeout。
	if r.Timeout != nil {
		spec.Template.ItemOperationTimeout = *r.Timeout
	}

	return spec
}

// UpdateAppRestoreRequest defines the request body for updating an AppRestore
type UpdateAppRestoreRequest struct {
	Name                string            `json:"name"`
	BackupSource        string            `json:"backupSource,omitempty"`
	Cluster             string            `json:"cluster,omitempty"`
	CleanVolumes        *bool             `json:"cleanVolumes,omitempty"`
	StorageClassMapping map[string]string `json:"storageClassMapping,omitempty"`
	SCMapping           map[string]string `json:"scMapping,omitempty"`
	IngressClassMapping map[string]string `json:"ingressClassMapping,omitempty"`
	ScaleToZeroList     []string          `json:"scaleToZeroList,omitempty"`
	StandbyList         []string          `json:"standbyList,omitempty"`

	Description            *string          `json:"description,omitempty"`            // 描述
	ExistingResourcePolicy string           `json:"existingResourcePolicy,omitempty"` // 资源冲突策略
	Timeout                *metav1.Duration `json:"timeout,omitempty"`                // 操作超时时间

	// Velero common parameters
	BackupName              string                `json:"backupName,omitempty"`
	IncludedNamespaces      []string              `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string              `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string              `json:"includedResources,omitempty"`
	ExcludedResources       []string              `json:"excludedResources,omitempty"`
	LabelSelector           *metav1.LabelSelector `json:"labelSelector,omitempty"`
	NamespaceMapping        map[string]string     `json:"namespaceMapping,omitempty"`
	RestorePVs              *bool                 `json:"restorePVs,omitempty"`
	PreserveNodePorts       *bool                 `json:"preserveNodePorts,omitempty"`
	IncludeClusterResources *bool                 `json:"includeClusterResources,omitempty"`

	// UploaderConfig parameters
	WriteSparseFiles      *bool `json:"writeSparseFiles,omitempty"`
	ParallelFilesDownload *int  `json:"parallelFilesDownload,omitempty"`

	Hooks *velerov1.RestoreHooks `json:"hooks,omitempty"`

	hooksSet   bool
	hooksClear bool
}

func (r *UpdateAppRestoreRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateAppRestoreRequest
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = UpdateAppRestoreRequest(decoded)
	if hooksRaw, ok := raw["hooks"]; ok {
		r.hooksSet = true
		if velerohooks.IsNullOrEmptyObject(hooksRaw) {
			r.Hooks = nil
			r.hooksClear = true
		}
	}
	return nil
}

// MergeToCRD updates the existing AppRestoreSpec with fields from the request
func (r *UpdateAppRestoreRequest) MergeToCRD(spec *dapisv1.AppRestoreSpec) {
	if r.BackupSource != "" {
		spec.BackupSource = r.BackupSource
	}
	if r.Cluster != "" {
		spec.Cluster = r.Cluster
	}
	if r.Timeout != nil {
		spec.Template.ItemOperationTimeout = *r.Timeout
	}

	// Update Template fields
	if r.BackupName != "" {
		spec.Template.BackupName = r.BackupName
	}
	if len(r.IncludedNamespaces) > 0 {
		spec.Template.IncludedNamespaces = r.IncludedNamespaces
	}
	if len(r.ExcludedNamespaces) > 0 {
		spec.Template.ExcludedNamespaces = r.ExcludedNamespaces
	}
	if len(r.IncludedResources) > 0 {
		spec.Template.IncludedResources = r.IncludedResources
	}
	if len(r.ExcludedResources) > 0 {
		spec.Template.ExcludedResources = r.ExcludedResources
	}
	if r.LabelSelector != nil {
		spec.Template.LabelSelector = r.LabelSelector
	}
	if len(r.NamespaceMapping) > 0 {
		spec.Template.NamespaceMapping = r.NamespaceMapping
	}
	if r.RestorePVs != nil {
		spec.Template.RestorePVs = r.RestorePVs
	}
	if r.PreserveNodePorts != nil {
		spec.Template.PreserveNodePorts = r.PreserveNodePorts
	}
	if r.IncludeClusterResources != nil {
		spec.Template.IncludeClusterResources = r.IncludeClusterResources
	}
	if r.ExistingResourcePolicy != "" {
		spec.Template.ExistingResourcePolicy = velerov1.PolicyType(r.ExistingResourcePolicy)
	}
	if r.hooksSet {
		if r.hooksClear || r.Hooks == nil {
			spec.Template.Hooks = velerov1.RestoreHooks{}
		} else {
			spec.Template.Hooks = *r.Hooks
		}
	}

	// UploaderConfig update
	if r.WriteSparseFiles != nil || r.ParallelFilesDownload != nil {
		if spec.Template.UploaderConfig == nil {
			spec.Template.UploaderConfig = &velerov1.UploaderConfigForRestore{}
		}
		if r.WriteSparseFiles != nil {
			spec.Template.UploaderConfig.WriteSparseFiles = r.WriteSparseFiles
		}
		if r.ParallelFilesDownload != nil {
			spec.Template.UploaderConfig.ParallelFilesDownload = *r.ParallelFilesDownload
		}
	}
}

// AppRestoreDTO is the data transfer object for AppRestore
type AppRestoreDTO struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Namespace         string              `json:"namespace"`
	Labels            map[string]string   `json:"labels,omitempty"`
	Description       string              `json:"description,omitempty"` // 描述
	CreationTimestamp common.LocalTime    `json:"creation_timestamp"`
	Spec              AppRestoreSpecDTO   `json:"spec"`
	Status            AppRestoreStatusDTO `json:"status"`
	TargetNamespaces  []string            `json:"targetNamespaces,omitempty"` // 辅助字段：目标命名空间列表
	BackupSourceType  string              `json:"backupSourceType,omitempty"` // 备份源类型 (Manual/Schedule)
}

type AppRestoreSpecDTO struct {
	BackupSource            string                 `json:"backupSource"`
	Cluster                 string                 `json:"cluster"`
	BackupName              string                 `json:"backupName"`
	IncludedNamespaces      []string               `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces      []string               `json:"excludedNamespaces,omitempty"`
	IncludedResources       []string               `json:"includedResources,omitempty"`
	ExcludedResources       []string               `json:"excludedResources,omitempty"`
	LabelSelector           *metav1.LabelSelector  `json:"labelSelector,omitempty"`
	NamespaceMapping        map[string]string      `json:"namespaceMapping,omitempty"`
	RestorePVs              *bool                  `json:"restorePVs,omitempty"`
	PreserveNodePorts       *bool                  `json:"preserveNodePorts,omitempty"`
	IncludeClusterResources *bool                  `json:"includeClusterResources,omitempty"`
	ExistingResourcePolicy  string                 `json:"existingResourcePolicy,omitempty"`
	Timeout                 string                 `json:"timeout,omitempty"`
	Hooks                   *velerov1.RestoreHooks `json:"hooks,omitempty"`

	// UploaderConfig
	WriteSparseFiles      *bool `json:"writeSparseFiles,omitempty"`
	ParallelFilesDownload int   `json:"parallelFilesDownload,omitempty"`
}

type AppRestoreStatusDTO struct {
	Phase         dapisv1.AppRestorePhase `json:"phase"`
	RestoreStatus VeleroRestoreStatusDTO  `json:"restoreStatus,omitempty"`
	LastAction    *RestoreActionDTO       `json:"lastAction,omitempty"`
	Reason        string                  `json:"reason,omitempty"`
	Message       string                  `json:"message,omitempty"`
}

type VeleroRestoreStatusDTO struct {
	Phase                          velerov1.RestorePhase     `json:"phase,omitempty"`
	ValidationErrors               []string                  `json:"validationErrors,omitempty"`
	Warnings                       int                       `json:"warnings,omitempty"`
	Errors                         int                       `json:"errors,omitempty"`
	FailureReason                  string                    `json:"failureReason,omitempty"`
	StartTimestamp                 *common.LocalTime         `json:"startTimestamp,omitempty"`
	CompletionTimestamp            *common.LocalTime         `json:"completionTimestamp,omitempty"`
	Progress                       *velerov1.RestoreProgress `json:"progress,omitempty"`
	RestoreItemOperationsAttempted int                       `json:"restoreItemOperationsAttempted,omitempty"`
	RestoreItemOperationsCompleted int                       `json:"restoreItemOperationsCompleted,omitempty"`
	RestoreItemOperationsFailed    int                       `json:"restoreItemOperationsFailed,omitempty"`
	HookStatus                     *velerov1.HookStatus      `json:"hookStatus,omitempty"`
}

type RestoreActionDTO struct {
	Type      string           `json:"type"`
	RequestAt common.LocalTime `json:"requestAt"`
}

func ConvertToAppRestoreDTO(item *dapisv1.AppRestore) AppRestoreDTO {
	dto := AppRestoreDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Labels:            item.Labels,
		Description:       item.Annotations[AppRestoreDescriptionAnnotation],
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
		BackupSourceType:  item.Labels[metadata.LabelAppRestoreSourceType],
	}

	// 计算 TargetNamespaces
	// 按照用户指示：优先使用 Operator 写入的 Status，如有映射则使用映射后的值。
	// 若 Operator 尚未写入 Status，则降级使用 Spec.IncludedNamespaces 并应用 Mapping (Client 端预计算)
	if len(item.Status.TargetNamespaces) > 0 {
		dto.TargetNamespaces = item.Status.TargetNamespaces
	} else {
		// 降级：Operator 尚未处理，Client 端预计算映射结果，提升响应速度
		baseNamespaces := item.Spec.Template.IncludedNamespaces
		if len(item.Spec.Template.NamespaceMapping) > 0 {
			finalTargets := make([]string, 0, len(baseNamespaces))
			for _, ns := range baseNamespaces {
				if mapped, ok := item.Spec.Template.NamespaceMapping[ns]; ok {
					finalTargets = append(finalTargets, mapped)
				} else {
					finalTargets = append(finalTargets, ns)
				}
			}
			dto.TargetNamespaces = finalTargets
		} else {
			dto.TargetNamespaces = baseNamespaces
		}
	}

	return dto
}

func ConvertSpecToDTO(spec dapisv1.AppRestoreSpec) AppRestoreSpecDTO {
	dto := AppRestoreSpecDTO{
		BackupSource:            spec.BackupSource,
		Cluster:                 spec.Cluster,
		BackupName:              spec.Template.BackupName,
		IncludedNamespaces:      spec.Template.IncludedNamespaces,
		ExcludedNamespaces:      spec.Template.ExcludedNamespaces,
		IncludedResources:       spec.Template.IncludedResources,
		ExcludedResources:       spec.Template.ExcludedResources,
		LabelSelector:           spec.Template.LabelSelector,
		NamespaceMapping:        spec.Template.NamespaceMapping,
		RestorePVs:              spec.Template.RestorePVs,
		PreserveNodePorts:       spec.Template.PreserveNodePorts,
		IncludeClusterResources: spec.Template.IncludeClusterResources,
		ExistingResourcePolicy:  string(spec.Template.ExistingResourcePolicy),
		Hooks:                   restoreHooksDTO(spec.Template.Hooks),
	}
	if spec.Template.ItemOperationTimeout.Duration > 0 {
		dto.Timeout = spec.Template.ItemOperationTimeout.Duration.String()
	} else if spec.Timeout != nil {
		// Fallback to Spec.Timeout if present (backward compatibility)
		dto.Timeout = spec.Timeout.Duration.String()
	}
	if spec.Template.UploaderConfig != nil {
		dto.WriteSparseFiles = spec.Template.UploaderConfig.WriteSparseFiles
		dto.ParallelFilesDownload = spec.Template.UploaderConfig.ParallelFilesDownload
	}
	return dto
}

func ConvertStatusToDTO(status dapisv1.AppRestoreStatus) AppRestoreStatusDTO {
	return AppRestoreStatusDTO{
		Phase:         status.Status,
		RestoreStatus: ConvertVeleroRestoreStatusToDTO(status.RestoreStatus),
		LastAction:    ConvertRestoreActionToDTO(status.LastAction),
		Reason:        status.Reason,
		Message:       status.Message,
	}
}

func ConvertVeleroRestoreStatusToDTO(status velerov1.RestoreStatus) VeleroRestoreStatusDTO {
	return VeleroRestoreStatusDTO{
		Phase:                          status.Phase,
		ValidationErrors:               status.ValidationErrors,
		Warnings:                       status.Warnings,
		Errors:                         status.Errors,
		FailureReason:                  status.FailureReason,
		StartTimestamp:                 common.NewLocalTimePtr(status.StartTimestamp),
		CompletionTimestamp:            common.NewLocalTimePtr(status.CompletionTimestamp),
		Progress:                       status.Progress,
		RestoreItemOperationsAttempted: status.RestoreItemOperationsAttempted,
		RestoreItemOperationsCompleted: status.RestoreItemOperationsCompleted,
		RestoreItemOperationsFailed:    status.RestoreItemOperationsFailed,
		HookStatus:                     status.HookStatus,
	}
}

func restoreHooksValue(hooks *velerov1.RestoreHooks) velerov1.RestoreHooks {
	if hooks == nil {
		return velerov1.RestoreHooks{}
	}
	return *hooks
}

func restoreHooksDTO(hooks velerov1.RestoreHooks) *velerov1.RestoreHooks {
	if len(hooks.Resources) == 0 {
		return nil
	}
	return &hooks
}

func ConvertRestoreActionToDTO(action *dapisv1.RestoreAction) *RestoreActionDTO {
	if action == nil {
		return nil
	}
	return &RestoreActionDTO{
		Type:      action.Type,
		RequestAt: common.NewLocalTime(action.RequestAt),
	}
}
