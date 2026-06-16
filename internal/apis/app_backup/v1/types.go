package appbackup

import (
	"encoding/json"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	velerohooks "github.com/softcdata/testudo-server/internal/apis/velero_hooks"
	"github.com/softcdata/testudo-server/internal/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AppBackupDescriptionAnnotation = "testudo.softcdata.com/app-backup-description"

// AppBackupDTO 备份应用传输对象
type AppBackupDTO struct {
	ID                string             `json:"id"`                 // 唯一标识
	Name              string             `json:"name"`               // 名称
	Namespace         string             `json:"namespace"`          // 命名空间
	Description       string             `json:"description"`        // 描述
	Labels            map[string]string  `json:"labels,omitempty"`   // 标签
	CreationTimestamp common.LocalTime   `json:"creation_timestamp"` // 创建时间
	Spec              AppBackupSpecDTO   `json:"spec"`               // 规格
	Status            AppBackupStatusDTO `json:"status"`             // 状态
}

type AppBackupSpecDTO struct {
	Cluster                  string                `json:"cluster"`                            // 集群名称
	Schedule                 string                `json:"schedule"`                           // 调度表达式
	DisasterPolicy           string                `json:"disasterPolicy,omitempty"`           // 灾难策略
	Paused                   bool                  `json:"paused,omitempty"`                   // 是否暂停
	SkipImmediately          *bool                 `json:"skipImmediately,omitempty"`          // 是否立即跳过
	IncludedNamespaces       []string              `json:"includedNamespaces,omitempty"`       // 包含的命名空间
	ExcludedNamespaces       []string              `json:"excludedNamespaces,omitempty"`       // 排除的命名空间
	IncludedResources        []string              `json:"includedResources,omitempty"`        // 包含的资源类型
	ExcludedResources        []string              `json:"excludedResources,omitempty"`        // 排除的资源类型
	LabelSelector            *metav1.LabelSelector `json:"labelSelector,omitempty"`            // 标签选择器
	SnapshotVolumes          *bool                 `json:"snapshotVolumes,omitempty"`          // 是否对卷进行快照
	StorageLocation          string                `json:"storageLocation,omitempty"`          // 备份存储位置
	TTL                      metav1.Duration       `json:"ttl,omitempty"`                      // 备份保留时间
	DefaultVolumesToFsBackup *bool                 `json:"defaultVolumesToFsBackup,omitempty"` // 默认是否使用文件系统备份卷
	VolumeSnapshotLocations  []string              `json:"volumeSnapshotLocations,omitempty"`  // 卷快照位置列表
	ParallelFilesUpload      int                   `json:"parallelFilesUpload,omitempty"`      // 并行文件上传数量

	IncludeClusterResources          *bool                 `json:"includeClusterResources,omitempty"`          // 包含集群资源
	Hooks                            *velerov1.BackupHooks `json:"hooks,omitempty"`                            // 备份钩子
	OrderedResources                 map[string]string     `json:"orderedResources,omitempty"`                 // 资源备份顺序
	CSISnapshotTimeout               metav1.Duration       `json:"csiSnapshotTimeout,omitempty"`               // CSI快照超时时间
	ItemOperationTimeout             metav1.Duration       `json:"itemOperationTimeout,omitempty"`             // 项目操作超时时间
	SnapshotMoveData                 *bool                 `json:"snapshotMoveData,omitempty"`                 // 快照移动数据
	Datamover                        string                `json:"datamover,omitempty"`                        // 数据移动器
	IncludedNamespaceScopedResources []string              `json:"includedNamespaceScopedResources,omitempty"` // 包含命名空间范围资源
	ExcludedNamespaceScopedResources []string              `json:"excludedNamespaceScopedResources,omitempty"` // 排除命名空间范围资源
	IncludedClusterScopedResources   []string              `json:"includedClusterScopedResources,omitempty"`   // 包含集群范围资源
	ExcludedClusterScopedResources   []string              `json:"excludedClusterScopedResources,omitempty"`   // 排除集群范围资源
}

type AppBackupStatusDTO struct {
	Phase              string                  `json:"phase"`                        // 阶段
	ScheduleStatus     VeleroScheduleStatusDTO `json:"scheduleStatus,omitempty"`     // 调度状态
	BackupStatus       VeleroBackupStatusDTO   `json:"backupStatus,omitempty"`       // 备份状态
	TotalBackups       int                     `json:"totalBackups,omitempty"`       // 总备份数
	History            []BackupRecordDTO       `json:"history,omitempty"`            // 历史记录 (Only populated in detail view)
	LastAction         *BackupActionDTO        `json:"lastAction,omitempty"`         // 最后一次操作
	LatestBackupStatus string                  `json:"latestBackupStatus,omitempty"` // 最新备份状态
	VeleroBackupName   string                  `json:"veleroBackupName,omitempty"`   // Velero备份名称
	Reason             string                  `json:"reason,omitempty"`             // 原因
	Message            string                  `json:"message,omitempty"`            // 消息
}

type VeleroScheduleStatusDTO struct {
	Phase            velerov1.SchedulePhase `json:"phase,omitempty"`
	LastBackup       *common.LocalTime      `json:"lastBackup,omitempty"`
	LastSkipped      *common.LocalTime      `json:"lastSkipped,omitempty"`
	ValidationErrors []string               `json:"validationErrors,omitempty"`
}

type VeleroBackupStatusDTO struct {
	Version                       int                      `json:"version,omitempty"`
	FormatVersion                 string                   `json:"formatVersion,omitempty"`
	Expiration                    *common.LocalTime        `json:"expiration,omitempty"`
	Phase                         velerov1.BackupPhase     `json:"phase,omitempty"`
	ValidationErrors              []string                 `json:"validationErrors,omitempty"`
	StartTimestamp                *common.LocalTime        `json:"startTimestamp,omitempty"`
	CompletionTimestamp           *common.LocalTime        `json:"completionTimestamp,omitempty"`
	VolumeSnapshotsAttempted      int                      `json:"volumeSnapshotsAttempted,omitempty"`
	VolumeSnapshotsCompleted      int                      `json:"volumeSnapshotsCompleted,omitempty"`
	FailureReason                 string                   `json:"failureReason,omitempty"`
	Warnings                      int                      `json:"warnings,omitempty"`
	Errors                        int                      `json:"errors,omitempty"`
	Progress                      *velerov1.BackupProgress `json:"progress,omitempty"`
	CSIVolumeSnapshotsAttempted   int                      `json:"csiVolumeSnapshotsAttempted,omitempty"`
	CSIVolumeSnapshotsCompleted   int                      `json:"csiVolumeSnapshotsCompleted,omitempty"`
	BackupItemOperationsAttempted int                      `json:"backupItemOperationsAttempted,omitempty"`
	BackupItemOperationsCompleted int                      `json:"backupItemOperationsCompleted,omitempty"`
	BackupItemOperationsFailed    int                      `json:"backupItemOperationsFailed,omitempty"`
	HookStatus                    *velerov1.HookStatus     `json:"hookStatus,omitempty"`
}

type BackupRecordDTO struct {
	Name                string                 `json:"name"`
	Phase               string                 `json:"phase"`
	ManagedStatus       string                 `json:"managedStatus,omitempty"`
	StartTimestamp      *common.LocalTime      `json:"startTimestamp,omitempty"`
	CompletionTimestamp *common.LocalTime      `json:"completionTimestamp,omitempty"`
	Errors              int                    `json:"errors,omitempty"`
	Warnings            int                    `json:"warnings,omitempty"`
	Expiration          *common.LocalTime      `json:"expiration,omitempty"`
	VeleroStatus        *VeleroBackupStatusDTO `json:"veleroStatus,omitempty"`
}

type BackupActionDTO struct {
	Type         string           `json:"type"`
	TargetBackup string           `json:"targetBackup,omitempty"`
	RequestAt    common.LocalTime `json:"requestAt"`
}

func ConvertToAppBackupDTO(item *dapisv1.AppBackup) AppBackupDTO {
	return AppBackupDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Description:       item.Annotations[AppBackupDescriptionAnnotation],
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Spec:              ConvertSpecToDTO(item.Spec),
		Status:            ConvertStatusToDTO(item.Status),
	}
}

func ConvertSpecToDTO(spec dapisv1.AppBackupSpec) AppBackupSpecDTO {
	return AppBackupSpecDTO{
		Cluster:                  spec.Cluster,
		Schedule:                 spec.Schedule,
		DisasterPolicy:           spec.DisasterPolicy,
		Paused:                   spec.Paused,
		SkipImmediately:          spec.SkipImmediately,
		IncludedNamespaces:       spec.Template.IncludedNamespaces,
		ExcludedNamespaces:       spec.Template.ExcludedNamespaces,
		IncludedResources:        spec.Template.IncludedResources,
		ExcludedResources:        spec.Template.ExcludedResources,
		LabelSelector:            spec.Template.LabelSelector,
		SnapshotVolumes:          spec.Template.SnapshotVolumes,
		StorageLocation:          spec.Template.StorageLocation,
		TTL:                      spec.Template.TTL,
		DefaultVolumesToFsBackup: spec.Template.DefaultVolumesToFsBackup,
		// ParallelFilesUpload:      spec.Template.UploaderConfig.ParallelFilesUpload,
		// VolumeSnapshotLocations:  spec.Template.VolumeSnapshotLocations,
		IncludeClusterResources:          spec.Template.IncludeClusterResources,
		IncludedNamespaceScopedResources: spec.Template.IncludedNamespaceScopedResources,
		ExcludedNamespaceScopedResources: spec.Template.ExcludedNamespaceScopedResources,
		IncludedClusterScopedResources:   spec.Template.IncludedClusterScopedResources,
		ExcludedClusterScopedResources:   spec.Template.ExcludedClusterScopedResources,
		Hooks:                            backupHooksDTO(spec.Template.Hooks),
		// OrderedResources:         spec.Template.OrderedResources,
		// CSISnapshotTimeout:       spec.Template.CSISnapshotTimeout,
		// ItemOperationTimeout:     spec.Template.ItemOperationTimeout,
		// SnapshotMoveData:         spec.Template.SnapshotMoveData,
		// Datamover:                spec.Template.DataMover,
	}
}

func ConvertStatusToDTO(status dapisv1.AppBackupStatus) AppBackupStatusDTO {
	dto := AppBackupStatusDTO{
		Phase:              status.Status,
		ScheduleStatus:     ConvertVeleroScheduleStatusToDTO(status.ScheduleStatus),
		BackupStatus:       ConvertVeleroBackupStatusToDTO(status.BackupStatus),
		TotalBackups:       status.TotalBackups,
		LastAction:         ConvertBackupActionToDTO(status.LastAction),
		LatestBackupStatus: status.LatestBackupStatus,
		Reason:             status.Reason,
		Message:            status.Message,
	}

	if len(status.History) > 0 {
		dto.VeleroBackupName = status.History[len(status.History)-1].Name
	}

	return dto
}

func ConvertVeleroScheduleStatusToDTO(status velerov1.ScheduleStatus) VeleroScheduleStatusDTO {
	return VeleroScheduleStatusDTO{
		Phase:            status.Phase,
		LastBackup:       common.NewLocalTimePtr(status.LastBackup),
		LastSkipped:      common.NewLocalTimePtr(status.LastSkipped),
		ValidationErrors: status.ValidationErrors,
	}
}

func ConvertVeleroBackupStatusToDTO(status velerov1.BackupStatus) VeleroBackupStatusDTO {
	return VeleroBackupStatusDTO{
		Version:                       status.Version,
		FormatVersion:                 status.FormatVersion,
		Expiration:                    common.NewLocalTimePtr(status.Expiration),
		Phase:                         status.Phase,
		ValidationErrors:              status.ValidationErrors,
		StartTimestamp:                common.NewLocalTimePtr(status.StartTimestamp),
		CompletionTimestamp:           common.NewLocalTimePtr(status.CompletionTimestamp),
		VolumeSnapshotsAttempted:      status.VolumeSnapshotsAttempted,
		VolumeSnapshotsCompleted:      status.VolumeSnapshotsCompleted,
		FailureReason:                 status.FailureReason,
		Warnings:                      status.Warnings,
		Errors:                        status.Errors,
		Progress:                      status.Progress,
		CSIVolumeSnapshotsAttempted:   status.CSIVolumeSnapshotsAttempted,
		CSIVolumeSnapshotsCompleted:   status.CSIVolumeSnapshotsCompleted,
		BackupItemOperationsAttempted: status.BackupItemOperationsAttempted,
		BackupItemOperationsCompleted: status.BackupItemOperationsCompleted,
		BackupItemOperationsFailed:    status.BackupItemOperationsFailed,
		HookStatus:                    status.HookStatus,
	}
}

func ConvertVeleroBackupStatusPtrToDTO(status *velerov1.BackupStatus) *VeleroBackupStatusDTO {
	if status == nil {
		return nil
	}
	dto := ConvertVeleroBackupStatusToDTO(*status)
	return &dto
}

func ConvertBackupRecordsToDTO(records []dapisv1.BackupRecord) []BackupRecordDTO {
	if len(records) == 0 {
		return nil
	}
	out := make([]BackupRecordDTO, 0, len(records))
	for _, record := range records {
		out = append(out, ConvertBackupRecordToDTO(record))
	}
	return out
}

func ConvertBackupRecordToDTO(record dapisv1.BackupRecord) BackupRecordDTO {
	return BackupRecordDTO{
		Name:                record.Name,
		Phase:               record.Phase,
		ManagedStatus:       record.ManagedStatus,
		StartTimestamp:      common.NewLocalTimePtr(record.StartTimestamp),
		CompletionTimestamp: common.NewLocalTimePtr(record.CompletionTimestamp),
		Errors:              record.Errors,
		Warnings:            record.Warnings,
		Expiration:          common.NewLocalTimePtr(record.Expiration),
		VeleroStatus:        ConvertVeleroBackupStatusPtrToDTO(record.VeleroStatus),
	}
}

func ConvertBackupActionToDTO(action *dapisv1.BackupAction) *BackupActionDTO {
	if action == nil {
		return nil
	}
	return &BackupActionDTO{
		Type:         action.Type,
		TargetBackup: action.TargetBackup,
		RequestAt:    common.NewLocalTime(action.RequestAt),
	}
}

// CreateAppBackupRequest defines the request body for creating an AppBackup
type CreateAppBackupRequest struct {
	Name            string `json:"name" binding:"required"`     // 名称
	Cluster         string `json:"cluster" binding:"required"`  // 集群
	Schedule        string `json:"schedule" binding:"required"` // 调度表达式
	DisasterPolicy  string `json:"disasterPolicy,omitempty"`    // 灾难策略
	Paused          bool   `json:"paused,omitempty"`            // 是否暂停
	SkipImmediately *bool  `json:"skipImmediately,omitempty"`   // 是否立即跳过
	Description     string `json:"description,omitempty"`       // 描述
	// Velero Template fields
	IncludedNamespaces       []string              `json:"includedNamespaces,omitempty"`       // 包含的命名空间
	ExcludedNamespaces       []string              `json:"excludedNamespaces,omitempty"`       // 排除的命名空间
	IncludedResources        []string              `json:"includedResources,omitempty"`        // 包含的资源类型
	ExcludedResources        []string              `json:"excludedResources,omitempty"`        // 排除的资源类型
	LabelSelector            *metav1.LabelSelector `json:"labelSelector,omitempty"`            // 标签选择器
	SnapshotVolumes          *bool                 `json:"snapshotVolumes,omitempty"`          // 是否对卷进行快照
	StorageLocation          string                `json:"storageLocation,omitempty"`          // 备份存储位置
	TTL                      metav1.Duration       `json:"ttl,omitempty"`                      // 备份保留时间
	DefaultVolumesToFsBackup *bool                 `json:"defaultVolumesToFsBackup,omitempty"` // 默认是否使用文件系统备份卷
	VolumeSnapshotLocations  []string              `json:"volumeSnapshotLocations,omitempty"`  // 卷快照位置列表
	// ParallelFilesUpload      int                   `json:"parallelFilesUpload,omitempty"`      // 并行文件上传数量

	IncludeClusterResources *bool                 `json:"includeClusterResources,omitempty"` // 包含集群资源
	Hooks                   *velerov1.BackupHooks `json:"hooks,omitempty"`                   // 备份钩子
	// OrderedResources        map[string]string    `json:"orderedResources,omitempty"`        // 资源备份顺序
	// CSISnapshotTimeout      metav1.Duration      `json:"csiSnapshotTimeout,omitempty"`      // CSI快照超时时间
	// ItemOperationTimeout    metav1.Duration      `json:"itemOperationTimeout,omitempty"`    // 项目操作超时时间
	// SnapshotMoveData        *bool                `json:"snapshotMoveData,omitempty"`        // 快照移动数据
	// Datamover               string               `json:"datamover,omitempty"`               // 数据移动器
	IncludedNamespaceScopedResources []string `json:"includedNamespaceScopedResources,omitempty"` // 包含命名空间范围资源
	ExcludedNamespaceScopedResources []string `json:"excludedNamespaceScopedResources,omitempty"` // 排除命名空间范围资源
	IncludedClusterScopedResources   []string `json:"includedClusterScopedResources,omitempty"`   // 包含集群范围资源
	ExcludedClusterScopedResources   []string `json:"excludedClusterScopedResources,omitempty"`   // 排除集群范围资源
}

func (r *CreateAppBackupRequest) hasScopedResourceFilters() bool {
	if r == nil {
		return false
	}
	return hasScopedAppBackupResourceFilters(
		r.IncludedNamespaceScopedResources,
		r.ExcludedNamespaceScopedResources,
		r.IncludedClusterScopedResources,
		r.ExcludedClusterScopedResources,
	)
}

func (r *CreateAppBackupRequest) normalizeScopedResourceFilters() {
	if !r.hasScopedResourceFilters() {
		return
	}
	r.IncludeClusterResources = nil
}

// ToCRD converts the request DTO to the Operator's AppBackupSpec
func (r *CreateAppBackupRequest) ToCRD() dapisv1.AppBackupSpec {
	r.normalizeScopedResourceFilters()
	spec := dapisv1.AppBackupSpec{
		Cluster:         r.Cluster,
		Schedule:        r.Schedule,
		DisasterPolicy:  r.DisasterPolicy,
		Paused:          r.Paused,
		SkipImmediately: r.SkipImmediately,
		Template: velerov1.BackupSpec{
			IncludedNamespaces:       r.IncludedNamespaces,
			ExcludedNamespaces:       r.ExcludedNamespaces,
			IncludedResources:        r.IncludedResources,
			ExcludedResources:        r.ExcludedResources,
			LabelSelector:            r.LabelSelector,
			SnapshotVolumes:          r.SnapshotVolumes,
			StorageLocation:          r.StorageLocation,
			TTL:                      r.TTL,
			DefaultVolumesToFsBackup: r.DefaultVolumesToFsBackup,
			// VolumeSnapshotLocations:  r.VolumeSnapshotLocations,
			IncludeClusterResources: r.IncludeClusterResources,
			Hooks:                   backupHooksValue(r.Hooks),
			// OrderedResources:         r.OrderedResources,
			// CSISnapshotTimeout:       r.CSISnapshotTimeout,
			// ItemOperationTimeout:     r.ItemOperationTimeout,
			// SnapshotMoveData:         r.SnapshotMoveData,
			// DataMover:                r.Datamover,
			IncludedNamespaceScopedResources: r.IncludedNamespaceScopedResources,
			ExcludedNamespaceScopedResources: r.ExcludedNamespaceScopedResources,
			IncludedClusterScopedResources:   r.IncludedClusterScopedResources,
			ExcludedClusterScopedResources:   r.ExcludedClusterScopedResources,
		},
	}
	// if r.ParallelFilesUpload > 0 {
	// 	spec.Template.UploaderConfig.ParallelFilesUpload = r.ParallelFilesUpload
	// }
	return spec
}

// UpdateAppBackupRequest defines the request body for updating an AppBackup
type UpdateAppBackupRequest struct {
	Name            string  `json:"name" binding:"required"`   // 名称
	Cluster         string  `json:"cluster,omitempty"`         // 集群
	Schedule        string  `json:"schedule,omitempty"`        // 调度表达式
	DisasterPolicy  string  `json:"disasterPolicy,omitempty"`  // 灾难策略
	Paused          *bool   `json:"paused,omitempty"`          // 是否暂停
	SkipImmediately *bool   `json:"skipImmediately,omitempty"` // 是否立即跳过
	Description     *string `json:"description,omitempty"`     // 描述
	// Velero Template fields
	IncludedNamespaces       []string              `json:"includedNamespaces,omitempty"`       // 包含的命名空间
	ExcludedNamespaces       []string              `json:"excludedNamespaces,omitempty"`       // 排除的命名空间
	IncludedResources        []string              `json:"includedResources,omitempty"`        // 包含的资源类型
	ExcludedResources        []string              `json:"excludedResources,omitempty"`        // 排除的资源类型
	LabelSelector            *metav1.LabelSelector `json:"labelSelector,omitempty"`            // 标签选择器
	SnapshotVolumes          *bool                 `json:"snapshotVolumes,omitempty"`          // 是否对卷进行快照
	StorageLocation          string                `json:"storageLocation,omitempty"`          // 备份存储位置
	TTL                      *metav1.Duration      `json:"ttl,omitempty"`                      // 备份保留时间
	DefaultVolumesToFsBackup *bool                 `json:"defaultVolumesToFsBackup,omitempty"` // 默认是否使用文件系统备份卷
	// VolumeSnapshotLocations  []string              `json:"volumeSnapshotLocations,omitempty"`  // 卷快照位置列表
	// ParallelFilesUpload      *int                  `json:"parallelFilesUpload,omitempty"`      // 并行文件上传数量

	IncludeClusterResources *bool                 `json:"includeClusterResources,omitempty"` // 包含集群资源
	Hooks                   *velerov1.BackupHooks `json:"hooks,omitempty"`                   // 备份钩子
	// OrderedResources        map[string]string     `json:"orderedResources,omitempty"`        // 资源备份顺序
	// CSISnapshotTimeout      *metav1.Duration      `json:"csiSnapshotTimeout,omitempty"`      // CSI快照超时时间
	// ItemOperationTimeout    *metav1.Duration      `json:"itemOperationTimeout,omitempty"`    // 项目操作超时时间
	// SnapshotMoveData        *bool                 `json:"snapshotMoveData,omitempty"`        // 快照移动数据
	// Datamover               string                `json:"datamover,omitempty"`               // 数据移动器
	IncludedNamespaceScopedResources []string `json:"includedNamespaceScopedResources,omitempty"` // 包含命名空间范围资源
	ExcludedNamespaceScopedResources []string `json:"excludedNamespaceScopedResources,omitempty"` // 排除命名空间范围资源
	IncludedClusterScopedResources   []string `json:"includedClusterScopedResources,omitempty"`   // 包含集群范围资源
	ExcludedClusterScopedResources   []string `json:"excludedClusterScopedResources,omitempty"`   // 排除集群范围资源

	hooksSet   bool
	hooksClear bool
}

func (r *UpdateAppBackupRequest) UnmarshalJSON(data []byte) error {
	type alias UpdateAppBackupRequest
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = UpdateAppBackupRequest(decoded)
	if hooksRaw, ok := raw["hooks"]; ok {
		r.hooksSet = true
		if velerohooks.IsNullOrEmptyObject(hooksRaw) {
			r.Hooks = nil
			r.hooksClear = true
		}
	}
	return nil
}

func (r *UpdateAppBackupRequest) hasScopedResourceFilters() bool {
	if r == nil {
		return false
	}
	return hasScopedAppBackupResourceFilters(
		r.IncludedNamespaceScopedResources,
		r.ExcludedNamespaceScopedResources,
		r.IncludedClusterScopedResources,
		r.ExcludedClusterScopedResources,
	)
}

func (r *UpdateAppBackupRequest) normalizeScopedResourceFilters() {
	if !r.hasScopedResourceFilters() {
		return
	}
	r.IncludeClusterResources = nil
}

// MergeToCRD updates the existing AppBackupSpec with fields from the request
func (r *UpdateAppBackupRequest) MergeToCRD(spec *dapisv1.AppBackupSpec) {
	r.normalizeScopedResourceFilters()

	if r.Cluster != "" {
		spec.Cluster = r.Cluster
	}
	if r.Schedule != "" {
		spec.Schedule = r.Schedule
	}
	if r.DisasterPolicy != "" {
		spec.DisasterPolicy = r.DisasterPolicy
	}
	if r.Paused != nil {
		spec.Paused = *r.Paused
	}
	if r.SkipImmediately != nil {
		spec.SkipImmediately = r.SkipImmediately
	}

	// Update Template fields
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
	if r.SnapshotVolumes != nil {
		spec.Template.SnapshotVolumes = r.SnapshotVolumes
	}
	if r.StorageLocation != "" {
		spec.Template.StorageLocation = r.StorageLocation
	}
	if r.TTL != nil {
		spec.Template.TTL = *r.TTL
	}
	if r.DefaultVolumesToFsBackup != nil {
		spec.Template.DefaultVolumesToFsBackup = r.DefaultVolumesToFsBackup
	}
	// if len(r.VolumeSnapshotLocations) > 0 {
	// 	spec.Template.VolumeSnapshotLocations = r.VolumeSnapshotLocations
	// }
	// if r.ParallelFilesUpload != nil {
	// 	spec.Template.UploaderConfig.ParallelFilesUpload = *r.ParallelFilesUpload
	// }
	if r.hasScopedResourceFilters() {
		spec.Template.IncludeClusterResources = nil
	} else if r.IncludeClusterResources != nil {
		spec.Template.IncludeClusterResources = r.IncludeClusterResources
	}
	if len(r.IncludedNamespaceScopedResources) > 0 {
		spec.Template.IncludedNamespaceScopedResources = r.IncludedNamespaceScopedResources
	}
	if len(r.ExcludedNamespaceScopedResources) > 0 {
		spec.Template.ExcludedNamespaceScopedResources = r.ExcludedNamespaceScopedResources
	}
	if len(r.IncludedClusterScopedResources) > 0 {
		spec.Template.IncludedClusterScopedResources = r.IncludedClusterScopedResources
	}
	if len(r.ExcludedClusterScopedResources) > 0 {
		spec.Template.ExcludedClusterScopedResources = r.ExcludedClusterScopedResources
	}
	if r.hooksSet {
		if r.hooksClear || r.Hooks == nil {
			spec.Template.Hooks = velerov1.BackupHooks{}
		} else {
			spec.Template.Hooks = *r.Hooks
		}
	}
	// if len(r.OrderedResources) > 0 {
	// 	spec.Template.OrderedResources = r.OrderedResources
	// }
	// if r.CSISnapshotTimeout != nil {
	// 	spec.Template.CSISnapshotTimeout = *r.CSISnapshotTimeout
	// }
	// if r.ItemOperationTimeout != nil {
	// 	spec.Template.ItemOperationTimeout = *r.ItemOperationTimeout
	// }
	// if r.SnapshotMoveData != nil {
	// 	spec.Template.SnapshotMoveData = r.SnapshotMoveData
	// }
	// if r.Datamover != "" {
	// 	spec.Template.DataMover = r.Datamover
	// }
}

func backupHooksValue(hooks *velerov1.BackupHooks) velerov1.BackupHooks {
	if hooks == nil {
		return velerov1.BackupHooks{}
	}
	return *hooks
}

func backupHooksDTO(hooks velerov1.BackupHooks) *velerov1.BackupHooks {
	if len(hooks.Resources) == 0 {
		return nil
	}
	return &hooks
}

// BackupDownloadResponse 备份下载响应
type BackupDownloadResponse struct {
	DownloadURL string               `json:"download_url,omitempty"` // 下载链接 (resource 类型使用)
	Files       []BackupFileDownload `json:"files,omitempty"`        // 文件列表 (data/all 类型使用)
	ExpiresAt   common.LocalTime     `json:"expires_at"`             // 过期时间
}

// BackupFileDownload 单个备份文件的下载信息
type BackupFileDownload struct {
	Key         string `json:"key"`          // S3 对象 key
	DownloadURL string `json:"download_url"` // 预签名下载链接
	Size        int64  `json:"size"`         // 文件大小 (bytes)
}

// AppBackupActionRequest defines the request body for executing an action
type AppBackupActionRequest struct {
	TargetBackup string `json:"targetBackup,omitempty"` // 目标备份
}
