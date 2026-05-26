package instance

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	configv1 "github.com/softcdata/testudo-server/internal/apis/disaster_config/v1"
	storagev1 "github.com/softcdata/testudo-server/internal/apis/disaster_storage/v1"
	"github.com/softcdata/testudo-server/internal/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DisasterInstanceDTO 是返回给前端的聚合对象
type DisasterInstanceDTO struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name"`
	Namespace         string                    `json:"namespace"`
	Labels            map[string]string         `json:"labels,omitempty"`
	Description       string                    `json:"description,omitempty"`
	CreationTimestamp common.LocalTime          `json:"creationTimestamp"`
	Spec              DisasterInstanceSpecDTO   `json:"spec"`
	Status            DisasterInstanceStatusDTO `json:"status"`
	Conditions        []ConditionDTO            `json:"conditions,omitempty"`
	ConditionSummary  *ConditionSummaryDTO      `json:"conditionSummary,omitempty"`

	// 聚合字段 (从 Config 中获取)
	ConfigName string                        `json:"configName"`
	Config     *configv1.DisasterConfigDTO   `json:"config,omitempty"`
	Storage    *storagev1.DisasterStorageDTO `json:"storage,omitempty"`

	// 聚合状态 (UI用于展示)
	CurrentState string `json:"currentState"` // e.g. "Running", "Syncing", "Error"

	// Sync Summaries for List View
	DataSyncStatus     *SyncSummaryDTO       `json:"dataSyncStatus,omitempty"`
	ResourceSyncStatus *SyncSummaryDTO       `json:"resourceSyncStatus,omitempty"`
	AutoCancel         *AutoCancelSummaryDTO `json:"autoCancel,omitempty"`
}

type SyncSummaryDTO struct {
	State            string            `json:"state"` // Ready, InProgress, etc.
	Reason           string            `json:"reason"`
	Message          string            `json:"message"`
	LastSyncTime     *common.LocalTime `json:"lastSyncTime,omitempty"`
	SyncSuccessCount int               `json:"syncSuccessCount"`
	SyncFailureCount int               `json:"syncFailureCount"`
	Paused           bool              `json:"paused"`
}

type DisasterInstanceSpecDTO struct {
	Config                      string                `json:"config"`
	DataSyncPolicy              string                `json:"dataSyncPolicy,omitempty"`
	ResourceSyncPolicy          string                `json:"resourceSyncPolicy,omitempty"`
	EffectiveDataSyncPolicy     string                `json:"effectiveDataSyncPolicy,omitempty"`
	EffectiveResourceSyncPolicy string                `json:"effectiveResourceSyncPolicy,omitempty"`
	DataSyncPolicySource        string                `json:"dataSyncPolicySource,omitempty"`
	ResourceSyncPolicySource    string                `json:"resourceSyncPolicySource,omitempty"`
	Namespaces                  []string              `json:"namespaces,omitempty"`
	LabelSelector               *metav1.LabelSelector `json:"labelSelector,omitempty"`
	PodRestoreMethod            string                `json:"podRestoreMethod,omitempty"`
	RestorePolicy               *RestorePolicyDTO     `json:"restorePolicy,omitempty"`
	SkipPodReadyCheck           *bool                 `json:"skipPodReadyCheck,omitempty"`
	Description                 string                `json:"description,omitempty"`
}

// RestorePolicyDTO explicitly mirrors operator RestorePolicy fields and adds text echo for modifier rules and bulk actions.
// Explicit fields avoid serialization/schema loss on embedded external structs.
type RestorePolicyDTO struct {
	ResourceSelection           *dapisv1.RestoreResourceSelectionPolicy `json:"resourceSelection,omitempty"`
	Execution                   *dapisv1.RestoreExecutionPolicy         `json:"execution,omitempty"`
	StorageClassMapping         *dapisv1.RestoreClassMappingPolicy      `json:"storageClassMapping,omitempty"`
	IngressClassMapping         *dapisv1.RestoreClassMappingPolicy      `json:"ingressClassMapping,omitempty"`
	BulkModifierActions         []dapisv1.BulkModifierAction            `json:"bulkModifierActions,omitempty"`
	BulkModifierActionsText     string                                  `json:"bulkModifierActionsText,omitempty"`
	ModifierRules               []dapisv1.RestoreModifierRule           `json:"modifierRules,omitempty"`
	ModifierRuleSnapshot        []dapisv1.RestoreModifierRule           `json:"modifierRuleSnapshot,omitempty"`
	ModifierRuleSnapshotHash    string                                  `json:"modifierRuleSnapshotHash,omitempty"`
	UseUnifiedDirectionResolver *bool                                   `json:"useUnifiedDirectionResolver,omitempty"`
	ModifierRulesText           string                                  `json:"modifierRulesText,omitempty"`
}

type DisasterInstanceStatusDTO struct {
	FsmState             string            `json:"fsmState,omitempty"`
	Reason               string            `json:"reason,omitempty"`
	Message              string            `json:"message,omitempty"`
	PrimaryCluster       string            `json:"primaryCluster,omitempty"`
	SecondaryCluster     string            `json:"secondaryCluster,omitempty"`
	LastDataSyncTime     *common.LocalTime `json:"lastDataSyncTime,omitempty"`
	LastResourceSyncTime *common.LocalTime `json:"lastResourceSyncTime,omitempty"`
	AvailableOperations  []string          `json:"availableOperations,omitempty"`
	DataSyncName         string            `json:"dataSyncName,omitempty"`
	ResourceSyncName     string            `json:"resourceSyncName,omitempty"`
}

type ConditionDTO struct {
	Type               string           `json:"type"`
	Status             string           `json:"status"`
	Reason             string           `json:"reason,omitempty"`
	Message            string           `json:"message,omitempty"`
	LastTransitionTime common.LocalTime `json:"lastTransitionTime,omitempty"`
	ObservedGeneration int64            `json:"observedGeneration,omitempty"`
}

type ConditionSummaryDTO struct {
	RoleDrift *ConditionDTO `json:"roleDrift,omitempty"`
}

// SyncStatusDTO 用于 "资源与数据同步" Tab
type SyncStatusDTO struct {
	DataSync     *SubResourceStatusDTO `json:"dataSync,omitempty"`
	ResourceSync *SubResourceStatusDTO `json:"resourceSync,omitempty"`
}

type SubResourceStatusDTO struct {
	Name           string             `json:"name"`
	Status         string             `json:"status"`
	Reason         string             `json:"reason"`
	Message        string             `json:"message"`
	Paused         bool               `json:"paused"`
	LastBackupName string             `json:"lastBackupName,omitempty"`
	LastTime       *common.LocalTime  `json:"lastTime,omitempty"`
	LastSyncStatus *LastSyncStatusDTO `json:"lastSyncStatus,omitempty"`
	Duration       string             `json:"duration,omitempty"` // e.g. "2m 30s"
	ResourceCount  int                `json:"resourceCount,omitempty"`

	RunningTerminalCommands []string `json:"runningTerminalCommands,omitempty"` // Example? No.

	BackupResourceCount  int `json:"backupResourceCount"`
	RestoreResourceCount int `json:"restoreResourceCount"`
	// Last Sync Failure (Last Backup/Restore Errors)
	FailureCount int `json:"failureCount"`

	// Historical Sync Stats (From BackupRestoreStatistics)
	SyncSuccessCount int `json:"syncSuccessCount"`
	SyncFailureCount int `json:"syncFailureCount"`
}

type LastSyncStatusDTO struct {
	Status               string            `json:"status"`
	StartTime            *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime       *common.LocalTime `json:"completionTime,omitempty"`
	Duration             string            `json:"duration,omitempty"`
	BackupName           string            `json:"backupName,omitempty"`
	RestoreName          string            `json:"restoreName,omitempty"`
	BackupResourceCount  int               `json:"backupResourceCount,omitempty"`
	RestoreResourceCount int               `json:"restoreResourceCount,omitempty"`
}

// HistoryStatusDTO 历史记录的统一状态对象
type HistoryStatusDTO struct {
	State   string `json:"state"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type SyncHistoryItemDTO struct {
	ID                   string            `json:"id"`
	SyncType             string            `json:"syncType"`
	Source               string            `json:"source"`
	Status               HistoryStatusDTO  `json:"status"`
	StartTime            *common.LocalTime `json:"startTime,omitempty"`
	CompletionTime       *common.LocalTime `json:"completionTime,omitempty"`
	Duration             string            `json:"duration,omitempty"`
	BackupName           string            `json:"backupName,omitempty"`
	RestoreName          string            `json:"restoreName,omitempty"`
	BackupResourceCount  int               `json:"backupResourceCount,omitempty"`
	RestoreResourceCount int               `json:"restoreResourceCount,omitempty"`
	SubResourceName      string            `json:"subResourceName,omitempty"`
	OperationName        string            `json:"operationName,omitempty"`
	OperationUID         string            `json:"operationUID,omitempty"`
	OperationType        string            `json:"operationType,omitempty"`
	HasOperationDetail   bool              `json:"hasOperationDetail"`
}

// HistoryDTO 用于 "历史操作记录" Tab
type HistoryDTO struct {
	Time common.LocalTime `json:"time"`
	Type string           `json:"type"` // ResourceSync, DataSync, Failover, Drill
	// Status 标准状态结构，前端统一读取 status.state/reason/message。
	Status HistoryStatusDTO `json:"status"`
	// AutoCancel 自动补偿摘要
	AutoCancel *AutoCancelSummaryDTO `json:"autoCancel,omitempty"`
	// OperationName / OperationUID 供详情页定位具体的 DisasterOperation。
	OperationName string `json:"operationName,omitempty"`
	OperationUID  string `json:"operationUID,omitempty"`
	HasDetail     bool   `json:"hasDetail"`
	// 兼容字段：保留历史前端读取路径。
	Result   string `json:"result,omitempty"` // = status.state
	Reason   string `json:"reason,omitempty"` // = status.reason
	Operator string `json:"operator"`         // system, admin
	Note     string `json:"note,omitempty"`   // = status.message
}

type AutoCancelSummaryDTO struct {
	Triggered                  bool              `json:"triggered"`
	Status                     string            `json:"status"`
	Reason                     string            `json:"reason,omitempty"`
	TriggerStep                string            `json:"triggerStep,omitempty"`
	ManualInterventionRequired bool              `json:"manualInterventionRequired"`
	TriggeredAt                *common.LocalTime `json:"triggeredAt,omitempty"`
	CompletionTime             *common.LocalTime `json:"completionTime,omitempty"`
}

// Request DTOs

type CreateDisasterInstanceRequest struct {
	Name               string                `json:"name" binding:"required"`
	Namespace          string                `json:"namespace"` // CR Namespace
	Config             string                `json:"config" binding:"required"`
	DataSyncPolicy     *string               `json:"dataSyncPolicy,omitempty"`
	ResourceSyncPolicy *string               `json:"resourceSyncPolicy,omitempty"`
	Namespaces         []string              `json:"namespaces,omitempty"`
	LabelSelector      *metav1.LabelSelector `json:"labelSelector,omitempty"`
	PodRestoreMethod   string                `json:"podRestoreMethod,omitempty"`
	RestorePolicy      *RestorePolicyRequest `json:"restorePolicy,omitempty"`
	SkipPodReadyCheck  *bool                 `json:"skipPodReadyCheck,omitempty"`
	Description        string                `json:"description,omitempty"`
}

type UpdateDisasterInstanceRequest struct {
	DataSyncPolicy     *string               `json:"dataSyncPolicy,omitempty"`
	ResourceSyncPolicy *string               `json:"resourceSyncPolicy,omitempty"`
	Namespaces         []string              `json:"namespaces,omitempty"`
	LabelSelector      *metav1.LabelSelector `json:"labelSelector,omitempty"`
	PodRestoreMethod   *string               `json:"podRestoreMethod,omitempty"`
	RestorePolicy      *RestorePolicyRequest `json:"restorePolicy,omitempty"`
	SkipPodReadyCheck  *bool                 `json:"skipPodReadyCheck,omitempty"`
	Description        *string               `json:"description,omitempty"`
}

// RestorePolicyRequest extends operator RestorePolicy with text input for modifier rules and bulk actions.
// Server parses modifierRulesText into structured modifierRules and bulkModifierActionsText into structured bulkModifierActions before writing CRD.
type RestorePolicyRequest struct {
	ResourceSelection           *dapisv1.RestoreResourceSelectionPolicy `json:"resourceSelection,omitempty"`
	Execution                   *dapisv1.RestoreExecutionPolicy         `json:"execution,omitempty"`
	StorageClassMapping         *dapisv1.RestoreClassMappingPolicy      `json:"storageClassMapping,omitempty"`
	IngressClassMapping         *dapisv1.RestoreClassMappingPolicy      `json:"ingressClassMapping,omitempty"`
	BulkModifierActions         []dapisv1.BulkModifierAction            `json:"bulkModifierActions,omitempty"`
	BulkModifierActionsText     string                                  `json:"bulkModifierActionsText,omitempty"`
	ModifierRules               []dapisv1.RestoreModifierRule           `json:"modifierRules,omitempty"`
	UseUnifiedDirectionResolver *bool                                   `json:"useUnifiedDirectionResolver,omitempty"`
	ModifierRulesText           string                                  `json:"modifierRulesText,omitempty"`

	resourceSelectionSet           bool
	executionSet                   bool
	storageClassMappingSet         bool
	ingressClassMappingSet         bool
	bulkModifierActionsSet         bool
	bulkModifierActionsTextSet     bool
	modifierRulesSet               bool
	useUnifiedDirectionResolverSet bool
	modifierRulesTextSet           bool
}

func (r *RestorePolicyRequest) UnmarshalJSON(data []byte) error {
	type restorePolicyRequestAlias RestorePolicyRequest
	var decoded restorePolicyRequestAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = RestorePolicyRequest(decoded)
	_, r.resourceSelectionSet = raw["resourceSelection"]
	_, r.executionSet = raw["execution"]
	_, r.storageClassMappingSet = raw["storageClassMapping"]
	_, r.ingressClassMappingSet = raw["ingressClassMapping"]
	_, r.bulkModifierActionsSet = raw["bulkModifierActions"]
	_, r.bulkModifierActionsTextSet = raw["bulkModifierActionsText"]
	_, r.modifierRulesSet = raw["modifierRules"]
	_, r.useUnifiedDirectionResolverSet = raw["useUnifiedDirectionResolver"]
	_, r.modifierRulesTextSet = raw["modifierRulesText"]
	return nil
}

type ExecuteActionRequest struct {
	Operation string                 `json:"operation" binding:"required"` // failover, reprotect, pause, resume
	Config    map[string]interface{} `json:"config,omitempty"`
}

// InstanceGroupsDTO 描述实例所属的容灾组关系，供前端判断分组归属。
type InstanceGroupsDTO struct {
	Instance  string   `json:"instance"`
	Namespace string   `json:"namespace"`
	InGroup   bool     `json:"inGroup"`
	Groups    []string `json:"groups"`
}

// ValidateTargetDTO 描述实例作为操作目标时的校验结果，供前端在操作前触发。
type ValidateTargetDTO struct {
	TargetName          string   `json:"targetName"`
	Namespace           string   `json:"namespace"`
	Operation           string   `json:"operation,omitempty"`
	Valid               bool     `json:"valid"`
	Reason              string   `json:"reason,omitempty"`
	Message             string   `json:"message,omitempty"`
	FsmState            string   `json:"fsmState,omitempty"`
	AvailableOperations []string `json:"availableOperations,omitempty"`
	InGroup             bool     `json:"inGroup"`
	Groups              []string `json:"groups"`
}

// ValidateRestoreClassesRequest 定义恢复 Class 预检接口请求体。
type ValidateRestoreClassesRequest struct {
	TargetCluster       string                     `json:"targetCluster,omitempty"`
	StorageClassMapping *RestoreClassMappingPolicy `json:"storageClassMapping,omitempty"`
	IngressClassMapping *RestoreClassMappingPolicy `json:"ingressClassMapping,omitempty"`
}

type RestoreClassUnmatchedPolicy string

const (
	RestoreClassUnmatchedPolicyKeep RestoreClassUnmatchedPolicy = "Keep"
	RestoreClassUnmatchedPolicyFail RestoreClassUnmatchedPolicy = "Fail"
)

type RestoreClassMapping struct {
	SourceClass string   `json:"sourceClass"`
	TargetClass string   `json:"targetClass"`
	Namespaces  []string `json:"namespaces,omitempty"`
}

type RestoreClassMappingPolicy struct {
	Mappings               []RestoreClassMapping       `json:"mappings,omitempty"`
	UnmatchedPolicy        RestoreClassUnmatchedPolicy `json:"unmatchedPolicy,omitempty"`
	StrictTargetValidation bool                        `json:"strictTargetValidation,omitempty"`
}

// RestoreClassCheckDTO 表示单类映射检查结果。
type RestoreClassCheckDTO struct {
	Enabled                bool     `json:"enabled"`
	StrictTargetValidation bool     `json:"strictTargetValidation"`
	CheckedTargets         []string `json:"checkedTargets,omitempty"`
	MissingTargets         []string `json:"missingTargets,omitempty"`
}

// ValidateRestoreClassesDTO 描述恢复 Class 预检结果。
type ValidateRestoreClassesDTO struct {
	InstanceName      string               `json:"instanceName"`
	Namespace         string               `json:"namespace"`
	TargetCluster     string               `json:"targetCluster"`
	Valid             bool                 `json:"valid"`
	Code              string               `json:"code,omitempty"`
	Message           string               `json:"message,omitempty"`
	StorageClassCheck RestoreClassCheckDTO `json:"storageClassCheck"`
	IngressClassCheck RestoreClassCheckDTO `json:"ingressClassCheck"`
}

type ProtectedNamespaceConflictMeta struct {
	ConflictType       string                           `json:"conflictType"`
	SourceCluster      string                           `json:"sourceCluster"`
	ConflictNamespaces []string                         `json:"conflictNamespaces,omitempty"`
	Owners             []common.ProtectedNamespaceOwner `json:"owners,omitempty"`
}

// Converters

func ConvertToDisasterInstanceDTO(item *dapisv1.DisasterInstance, config *dapisv1.DisasterConfig, storage *dapisv1.StorageRepository) DisasterInstanceDTO {
	dto := DisasterInstanceDTO{
		ID:                string(item.UID),
		Name:              item.Name,
		Namespace:         item.Namespace,
		Labels:            item.Labels,
		CreationTimestamp: common.NewLocalTime(item.CreationTimestamp),
		Description:       item.Annotations["testudo.softcdata.com/description"],
		Spec: DisasterInstanceSpecDTO{
			Config:             item.Spec.Config,
			DataSyncPolicy:     item.Spec.DataSyncPolicy,
			ResourceSyncPolicy: item.Spec.ResourceSyncPolicy,
			Namespaces:         item.Spec.Namespaces,
			LabelSelector:      item.Spec.LabelSelector,
			PodRestoreMethod:   item.Spec.PodRestoreMethod,
			RestorePolicy:      convertRestorePolicyDTO(item.Spec.RestorePolicy),
			SkipPodReadyCheck:  item.Spec.SkipPodReadyCheck,
			Description:        item.Annotations["testudo.softcdata.com/description"],
		},
		Status: DisasterInstanceStatusDTO{
			FsmState:             item.Status.FsmState,
			Reason:               getInstanceStatusStringField(&item.Status, "Reason"),
			Message:              getInstanceStatusStringField(&item.Status, "Message"),
			PrimaryCluster:       item.Status.PrimaryCluster,
			SecondaryCluster:     item.Status.SecondaryCluster,
			LastDataSyncTime:     common.NewLocalTimePtr(item.Status.LastDataSyncTime),
			LastResourceSyncTime: common.NewLocalTimePtr(item.Status.LastResourceSyncTime),
			AvailableOperations:  item.Status.AvailableOperations,
			DataSyncName:         item.Status.DataSyncName,
			ResourceSyncName:     item.Status.ResourceSyncName,
		},
		ConfigName:   item.Spec.Config,
		CurrentState: determineCurrentState(item),
	}
	dto.Conditions = convertConditionDTOs(item.Status.Conditions)
	dto.ConditionSummary = buildConditionSummary(dto.Conditions)

	if config != nil {
		cfgDto := configv1.ConvertToDisasterConfigDTO(config)
		dto.Config = &cfgDto
	}
	populateInstanceSyncPolicyDTO(&dto.Spec, item, config)

	if storage != nil {
		sDto := storagev1.ConvertToDisasterStorageDTO(storage)
		dto.Storage = &sDto
	}

	return dto
}

func convertConditionDTOs(conditions []metav1.Condition) []ConditionDTO {
	if len(conditions) == 0 {
		return nil
	}
	out := make([]ConditionDTO, 0, len(conditions))
	for _, condition := range conditions {
		out = append(out, ConditionDTO{
			Type:               condition.Type,
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
			LastTransitionTime: common.NewLocalTime(condition.LastTransitionTime),
			ObservedGeneration: condition.ObservedGeneration,
		})
	}
	return out
}

func buildConditionSummary(conditions []ConditionDTO) *ConditionSummaryDTO {
	var summary ConditionSummaryDTO
	for i := range conditions {
		switch conditions[i].Type {
		case "RoleDrift":
			condition := conditions[i]
			summary.RoleDrift = &condition
		}
	}
	if summary.RoleDrift == nil {
		return nil
	}
	return &summary
}

func getInstanceStatusStringField(status any, fieldName string) string {
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

func determineCurrentState(item *dapisv1.DisasterInstance) string {
	if item == nil {
		return string(dapisv1.CurrentStateUnknown)
	}
	return string(dapisv1.CurrentStateFromFSM(item.Status.FsmState))
}

func ConvertToAutoCancelSummary(op *dapisv1.DisasterOperation) *AutoCancelSummaryDTO {
	if op == nil || !hasAutoCancelSummary(op) {
		return nil
	}
	return &AutoCancelSummaryDTO{
		Triggered:                  op.Status.AutoCancelTriggered,
		Status:                     string(op.Status.AutoCancelStatus),
		Reason:                     op.Status.AutoCancelReason,
		TriggerStep:                op.Status.AutoCancelTriggerStep,
		ManualInterventionRequired: op.Status.ManualInterventionRequired,
		TriggeredAt:                common.NewLocalTimePtr(op.Status.AutoCancelTriggeredAt),
		CompletionTime:             common.NewLocalTimePtr(op.Status.AutoCancelCompletionTime),
	}
}

func ConvertToHistoryDTO(op *dapisv1.DisasterOperation) HistoryDTO {
	state := ""
	reason := ""
	message := ""
	timestamp := metav1.Time{}
	opType := ""
	if op != nil {
		state = string(op.Status.State)
		reason = op.Status.Reason
		message = op.Status.Message
		timestamp = op.CreationTimestamp
		opType = string(op.Spec.OperationType)
	}
	return HistoryDTO{
		Time: common.NewLocalTime(timestamp),
		Type: opType,
		Status: HistoryStatusDTO{
			State:   state,
			Reason:  reason,
			Message: message,
		},
		AutoCancel: ConvertToAutoCancelSummary(op),
		OperationName: func() string {
			if op == nil {
				return ""
			}
			return op.Name
		}(),
		OperationUID: func() string {
			if op == nil {
				return ""
			}
			return string(op.UID)
		}(),
		HasDetail: op != nil && op.Name != "",
		Result:    state,
		Reason:    reason,
		Operator:  "admin",
		Note:      message,
	}
}

func hasAutoCancelSummary(op *dapisv1.DisasterOperation) bool {
	if op == nil {
		return false
	}
	return op.Status.AutoCancelTriggered ||
		op.Status.AutoCancelStatus != "" ||
		op.Status.AutoCancelReason != "" ||
		op.Status.AutoCancelTriggerStep != "" ||
		op.Status.AutoCancelTriggeredAt != nil ||
		op.Status.AutoCancelCompletionTime != nil ||
		op.Status.ManualInterventionRequired
}

func (r *CreateDisasterInstanceRequest) ToCRD() (dapisv1.DisasterInstanceSpec, error) {
	restorePolicy, err := r.ResolveRestorePolicy()
	if err != nil {
		return dapisv1.DisasterInstanceSpec{}, err
	}
	spec := dapisv1.DisasterInstanceSpec{
		Config:            r.Config,
		Namespaces:        r.Namespaces,
		LabelSelector:     r.LabelSelector,
		PodRestoreMethod:  r.PodRestoreMethod,
		RestorePolicy:     restorePolicy,
		SkipPodReadyCheck: r.SkipPodReadyCheck,
	}
	r.applyPolicyFields(&spec)
	return spec, nil
}

func (r *CreateDisasterInstanceRequest) ResolveRestorePolicy() (*dapisv1.RestorePolicy, error) {
	if r == nil || r.RestorePolicy == nil {
		return nil, nil
	}
	return r.RestorePolicy.ToCRD()
}

func (r *UpdateDisasterInstanceRequest) ResolveRestorePolicy() (*dapisv1.RestorePolicy, error) {
	if r == nil || r.RestorePolicy == nil {
		return nil, nil
	}
	return r.RestorePolicy.ToCRD()
}

func (r *RestorePolicyRequest) ToCRD() (*dapisv1.RestorePolicy, error) {
	if r == nil {
		return nil, nil
	}
	normalizeRestoreResourceSelection(r.ResourceSelection)
	if err := validateRestoreResourceSelection(r.ResourceSelection); err != nil {
		return nil, err
	}
	policy := &dapisv1.RestorePolicy{
		ResourceSelection:           r.ResourceSelection,
		Execution:                   r.Execution,
		StorageClassMapping:         r.StorageClassMapping,
		IngressClassMapping:         r.IngressClassMapping,
		BulkModifierActions:         cloneBulkModifierActions(r.BulkModifierActions),
		UseUnifiedDirectionResolver: r.UseUnifiedDirectionResolver,
	}

	bulkText := strings.TrimSpace(r.BulkModifierActionsText)
	bulkTextProvided := r.bulkModifierActionsTextSet || bulkText != ""
	bulkActionsProvided := r.bulkModifierActionsSet || len(r.BulkModifierActions) > 0
	if bulkTextProvided {
		var parsedActions []dapisv1.BulkModifierAction
		if bulkText != "" {
			var err error
			parsedActions, err = parseBulkModifierActionsText(bulkText)
			if err != nil {
				return nil, err
			}
		}
		if bulkActionsProvided {
			equal, err := bulkModifierActionsEqual(r.BulkModifierActions, parsedActions)
			if err != nil {
				return nil, err
			}
			if !equal {
				return nil, fmt.Errorf(
					"BulkModifierActionsInputConflict: restorePolicy.bulkModifierActions and restorePolicy.bulkModifierActionsText are both provided but not equal",
				)
			}
		}
		policy.BulkModifierActions = cloneBulkModifierActions(parsedActions)
	}

	text := strings.TrimSpace(r.ModifierRulesText)
	textProvided := r.modifierRulesTextSet || text != ""
	rulesProvided := r.modifierRulesSet || len(r.ModifierRules) > 0
	if !textProvided {
		policy.ModifierRules = cloneRestoreModifierRules(r.ModifierRules)
		return policy, nil
	}

	var parsedRules []dapisv1.RestoreModifierRule
	if text != "" {
		var err error
		parsedRules, err = parseModifierRulesText(text)
		if err != nil {
			return nil, err
		}
	}
	if rulesProvided && !restoreModifierRulesSemanticallyEqual(r.ModifierRules, parsedRules) {
		return nil, fmt.Errorf(
			"ModifierRulesInputConflict: restorePolicy.modifierRules and restorePolicy.modifierRulesText are both provided but not equal",
		)
	}
	policy.ModifierRules = parsedRules
	return policy, nil
}

func mergeRestorePolicyForUpdate(
	current *dapisv1.RestorePolicy,
	request *RestorePolicyRequest,
	parsed *dapisv1.RestorePolicy,
) *dapisv1.RestorePolicy {
	if request == nil || parsed == nil {
		if current == nil {
			return nil
		}
		return current.DeepCopy()
	}

	var merged *dapisv1.RestorePolicy
	if current != nil {
		merged = current.DeepCopy()
	} else {
		merged = &dapisv1.RestorePolicy{}
	}

	if request.resourceSelectionSet {
		if parsed.ResourceSelection == nil {
			merged.ResourceSelection = nil
		} else {
			merged.ResourceSelection = parsed.ResourceSelection.DeepCopy()
		}
	}
	if request.executionSet {
		if parsed.Execution == nil {
			merged.Execution = nil
		} else {
			merged.Execution = parsed.Execution.DeepCopy()
		}
	}
	if request.storageClassMappingSet {
		if parsed.StorageClassMapping == nil {
			merged.StorageClassMapping = nil
		} else {
			merged.StorageClassMapping = parsed.StorageClassMapping.DeepCopy()
		}
	}
	if request.ingressClassMappingSet {
		if parsed.IngressClassMapping == nil {
			merged.IngressClassMapping = nil
		} else {
			merged.IngressClassMapping = parsed.IngressClassMapping.DeepCopy()
		}
	}
	if request.bulkModifierActionsSet || request.bulkModifierActionsTextSet {
		merged.BulkModifierActions = cloneBulkModifierActions(parsed.BulkModifierActions)
	}
	if request.modifierRulesSet || request.modifierRulesTextSet {
		merged.ModifierRules = cloneRestoreModifierRules(parsed.ModifierRules)
	}
	if request.useUnifiedDirectionResolverSet {
		merged.UseUnifiedDirectionResolver = cloneBoolPtr(parsed.UseUnifiedDirectionResolver)
	}

	return merged
}

func restoreModifierRulesSemanticallyEqual(left []dapisv1.RestoreModifierRule, right []dapisv1.RestoreModifierRule) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}
	return reflect.DeepEqual(left, right)
}

func normalizeRestoreResourceSelection(selection *dapisv1.RestoreResourceSelectionPolicy) {
	if selection == nil {
		return
	}
	if hasRestoreScopedResourceFilters(selection) {
		selection.IncludeClusterResources = nil
	}
}

func validateRestoreResourceSelection(selection *dapisv1.RestoreResourceSelectionPolicy) error {
	if selection == nil {
		return nil
	}

	if selection.IncludeClusterResources != nil && *selection.IncludeClusterResources {
		return validateRestoreResourceFilterPair(
			"restorePolicy.resourceSelection.includedResources",
			"restorePolicy.resourceSelection.excludedResources",
			selection.IncludedResources,
			selection.ExcludedResources,
		)
	}

	if hasRestoreScopedResourceFilters(selection) {
		if err := validateRestoreResourceFilterPair(
			"restorePolicy.resourceSelection.includedNamespaceScopedResources",
			"restorePolicy.resourceSelection.excludedNamespaceScopedResources",
			selection.IncludedNamespaceScopedResources,
			selection.ExcludedNamespaceScopedResources,
		); err != nil {
			return err
		}
		return validateRestoreResourceFilterPair(
			"restorePolicy.resourceSelection.includedClusterScopedResources",
			"restorePolicy.resourceSelection.excludedClusterScopedResources",
			selection.IncludedClusterScopedResources,
			selection.ExcludedClusterScopedResources,
		)
	}

	return validateRestoreResourceFilterPair(
		"restorePolicy.resourceSelection.includedResources",
		"restorePolicy.resourceSelection.excludedResources",
		selection.IncludedResources,
		selection.ExcludedResources,
	)
}

func hasRestoreScopedResourceFilters(selection *dapisv1.RestoreResourceSelectionPolicy) bool {
	if selection == nil {
		return false
	}
	return len(trimRestoreResourceFilterValues(selection.IncludedNamespaceScopedResources)) > 0 ||
		len(trimRestoreResourceFilterValues(selection.ExcludedNamespaceScopedResources)) > 0 ||
		len(trimRestoreResourceFilterValues(selection.IncludedClusterScopedResources)) > 0 ||
		len(trimRestoreResourceFilterValues(selection.ExcludedClusterScopedResources)) > 0
}

func validateRestoreResourceFilterPair(includeField string, excludeField string, includeRaw []string, excludeRaw []string) error {
	include := trimRestoreResourceFilterValues(includeRaw)
	exclude := trimRestoreResourceFilterValues(excludeRaw)
	if len(include) == 0 || len(exclude) == 0 {
		return nil
	}

	includeSet := make(map[string]struct{}, len(include))
	for _, item := range include {
		includeSet[item] = struct{}{}
	}
	excludeSet := make(map[string]struct{}, len(exclude))
	for _, item := range exclude {
		excludeSet[item] = struct{}{}
	}

	if _, hasWildcard := includeSet["*"]; hasWildcard {
		return fmt.Errorf(
			"ResourceSelectionInvalid: %s contains '*' and cannot be combined with %s",
			includeField,
			excludeField,
		)
	}
	if _, hasWildcard := excludeSet["*"]; hasWildcard {
		return fmt.Errorf(
			"ResourceSelectionInvalid: %s contains '*' and cannot be combined with %s",
			excludeField,
			includeField,
		)
	}

	for _, item := range include {
		if _, conflict := excludeSet[item]; conflict {
			return fmt.Errorf(
				"ResourceSelectionInvalid: %s and %s conflict on resource %q",
				includeField,
				excludeField,
				item,
			)
		}
	}
	return nil
}

func trimRestoreResourceFilterValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseModifierRulesText(raw string) ([]dapisv1.RestoreModifierRule, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var rules []dapisv1.RestoreModifierRule
	if err := json.Unmarshal([]byte(trimmed), &rules); err != nil {
		return nil, fmt.Errorf("ModifierRulesTextInvalid: %w", err)
	}
	return cloneRestoreModifierRules(rules), nil
}

func parseBulkModifierActionsText(raw string) ([]dapisv1.BulkModifierAction, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, fmt.Errorf("BulkModifierActionsTextInvalid: %w", err)
	}

	switch payload.(type) {
	case []any:
		var actions []dapisv1.BulkModifierAction
		if err := json.Unmarshal([]byte(trimmed), &actions); err != nil {
			return nil, fmt.Errorf("BulkModifierActionsTextInvalid: %w", err)
		}
		return cloneBulkModifierActions(actions), nil
	case map[string]any:
		var action dapisv1.BulkModifierAction
		if err := json.Unmarshal([]byte(trimmed), &action); err != nil {
			return nil, fmt.Errorf("BulkModifierActionsTextInvalid: %w", err)
		}
		return []dapisv1.BulkModifierAction{action}, nil
	default:
		return nil, fmt.Errorf("BulkModifierActionsTextInvalid: expected JSON object or array")
	}
}

func convertRestorePolicyDTO(policy *dapisv1.RestorePolicy) *RestorePolicyDTO {
	if policy == nil {
		return nil
	}
	dto := &RestorePolicyDTO{
		ResourceSelection:           policy.ResourceSelection,
		Execution:                   policy.Execution,
		StorageClassMapping:         policy.StorageClassMapping,
		IngressClassMapping:         policy.IngressClassMapping,
		BulkModifierActions:         cloneBulkModifierActions(policy.BulkModifierActions),
		ModifierRules:               cloneRestoreModifierRules(policy.ModifierRules),
		ModifierRuleSnapshot:        cloneRestoreModifierRules(policy.ModifierRuleSnapshot),
		ModifierRuleSnapshotHash:    strings.TrimSpace(policy.ModifierRuleSnapshotHash),
		UseUnifiedDirectionResolver: cloneBoolPtr(policy.UseUnifiedDirectionResolver),
	}
	dto.BulkModifierActionsText = encodeBulkModifierActionsText(dto.BulkModifierActions)
	dto.ModifierRulesText = encodeModifierRulesText(dto.ModifierRules)
	return dto
}

// ConvertRestorePolicyDTO exposes restore-policy DTO conversion for other API modules
// that need identical echo semantics (for example drills reusing instance restore policy UX).
func ConvertRestorePolicyDTO(policy *dapisv1.RestorePolicy) *RestorePolicyDTO {
	return convertRestorePolicyDTO(policy)
}

func encodeBulkModifierActionsText(actions []dapisv1.BulkModifierAction) string {
	if len(actions) == 0 {
		return ""
	}
	b, err := json.Marshal(actions)
	if err != nil {
		return ""
	}
	return string(b)
}

func encodeModifierRulesText(rules []dapisv1.RestoreModifierRule) string {
	if len(rules) == 0 {
		return ""
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	return string(b)
}

func cloneBoolPtr(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneRestoreModifierRules(in []dapisv1.RestoreModifierRule) []dapisv1.RestoreModifierRule {
	if len(in) == 0 {
		return nil
	}
	out := make([]dapisv1.RestoreModifierRule, len(in))
	for i := range in {
		in[i].DeepCopyInto(&out[i])
	}
	return out
}

func cloneBulkModifierActions(in []dapisv1.BulkModifierAction) []dapisv1.BulkModifierAction {
	if len(in) == 0 {
		return nil
	}
	out := make([]dapisv1.BulkModifierAction, len(in))
	for i := range in {
		in[i].DeepCopyInto(&out[i])
	}
	return out
}

func bulkModifierActionsEqual(left []dapisv1.BulkModifierAction, right []dapisv1.BulkModifierAction) (bool, error) {
	normalizedLeft, err := normalizeBulkModifierActions(cloneBulkModifierActions(left))
	if err != nil {
		return false, err
	}
	normalizedRight, err := normalizeBulkModifierActions(cloneBulkModifierActions(right))
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(normalizedLeft, normalizedRight), nil
}
