package businessdefaultconfig

import (
	"fmt"
	"sort"
	"strings"
)

type groupDefinition struct {
	Key         string
	Name        string
	Description string
}

type fieldDefinition struct {
	Key          string
	GroupKey     string
	Name         string
	Description  string
	DefaultValue interface{}
	DataType     FieldValueType
	Editable     bool
	EffectMode   EffectMode
	Min          interface{}
	Max          interface{}
	EnumValues   []string
}

var groupDefinitions = []groupDefinition{
	{Key: "backupRuntime", Name: "备份运行时", Description: "控制 AppBackup 与 Velero Backup 观察、等待和轮询的默认参数。"},
	{Key: "restoreRuntime", Name: "恢复运行时", Description: "控制 AppRestore 与 Velero Restore 状态观察、卡顿保护和自动重试的默认参数。"},
	{Key: "operationRuntime", Name: "容灾操作运行时", Description: "控制 DisasterOperation 步骤超时、步骤重试和状态重排队的默认参数。"},
	{Key: "instanceRuntime", Name: "实例运行时", Description: "控制 DisasterInstance 状态转换看门狗和不同状态下重排队的默认参数。"},
	{Key: "syncRuntime", Name: "同步运行时", Description: "控制 DataSync、ResourceSync 的观察间隔、恢复观察和历史保留数量。"},
	{Key: "storageRuntime", Name: "存储运行时", Description: "控制 StorageRepository 校验、统计和状态刷新重排队间隔。"},
	{Key: "clusterRuntime", Name: "集群运行时", Description: "控制 DisasterCluster 对账、删除重试、Velero 安装和 Helm 锁保护的默认参数。"},
}

var fieldDefinitions = []fieldDefinition{
	durationField("backupRuntime", "inProgressMaxWait", "备份进行中最大等待", "Velero Backup 处于 InProgress 时允许等待的最长时间，超过后由 operator 按失败路径处理。", "2h", "1m", "24h"),
	durationField("backupRuntime", "unknownMaxWait", "备份未知状态最大等待", "Velero Backup 状态未知时允许等待的最长时间，超过后由 operator 按失败路径处理。", "10m", "1m", "24h"),
	durationField("backupRuntime", "pollInterval", "备份状态轮询间隔", "AppBackup 观察 Velero Backup 状态的默认重排队间隔，过小会增加 controller 负载。", "10s", "1s", "5m"),

	durationField("restoreRuntime", "inProgressMaxWait", "恢复进行中最大等待", "Velero Restore 处于 InProgress 时允许等待的最长时间，AppRestore.spec.timeout 存在时优先使用资源级超时。", "1h", "1m", "24h"),
	durationField("restoreRuntime", "unknownMaxWait", "恢复未知状态最大等待", "Velero Restore 状态未知时允许等待的最长时间，AppRestore.spec.timeout 存在时优先使用资源级超时。", "1h", "1m", "24h"),
	durationField("restoreRuntime", "inProgressPollInterval", "恢复进行中轮询间隔", "AppRestore 观察 InProgress Restore 的默认重排队间隔，过小会增加 controller 负载。", "5s", "1s", "5m"),
	durationField("restoreRuntime", "unknownPollInterval", "恢复未知状态轮询间隔", "AppRestore 观察状态未知 Restore 的默认重排队间隔。", "10s", "1s", "5m"),
	durationField("restoreRuntime", "progressCompleteGrace", "进度完成宽限期", "Restore 已接近完成但状态尚未结束时的宽限时间，用于避免短暂状态滞后被误判失败。", "5m", "30s", "24h"),
	durationField("restoreRuntime", "startupGrace", "恢复启动宽限期", "Restore 创建后尚未出现有效进度时的宽限时间，超过后进入启动卡顿判断。", "5m", "30s", "24h"),
	durationField("restoreRuntime", "missingGrace", "恢复对象缺失宽限期", "Velero Restore 短暂查询不到时的容忍时间，超过后按缺失处理。", "90s", "30s", "24h"),
	durationField("restoreRuntime", "emptyStatusGrace", "空状态宽限期", "Velero Restore status 为空时的容忍时间，超过后按异常状态处理。", "5m", "30s", "24h"),
	durationField("restoreRuntime", "podVolumeRestorePendingMaxWait", "PodVolumeRestore Pending 最大等待", "PodVolumeRestore 长时间停留在 Pending 时允许等待的最长时间。", "10m", "1m", "24h"),
	durationField("restoreRuntime", "retryBackoff", "恢复自动重试间隔", "AppRestore 遇到可重试异常后的默认自动重试等待时间。", "15s", "1s", "1h"),
	intField("restoreRuntime", "retryLimit", "恢复自动重试次数", "兼容总重试次数；当具体失败类型重试次数未配置时作为 fallback，0 表示不自动重试。", 1, 0, 10),
	intField("restoreRuntime", "retryLimitProgress", "进度卡顿重试次数", "Restore 进度卡顿时允许自动重试的次数，0 表示不自动重试。", 1, 0, 10),
	intField("restoreRuntime", "retryLimitStartup", "启动卡顿重试次数", "Restore 启动阶段卡顿时允许自动重试的次数，0 表示不自动重试。", 1, 0, 10),
	intField("restoreRuntime", "retryLimitMissing", "Restore 缺失重试次数", "Velero Restore 缺失时允许自动重试的次数，0 表示不自动重试。", 2, 0, 10),
	intField("restoreRuntime", "retryLimitEmpty", "空状态重试次数", "Velero Restore status 为空时允许自动重试的次数，0 表示不自动重试。", 2, 0, 10),

	intField("operationRuntime", "defaultTimeoutMinutes", "操作默认超时分钟数", "DisasterOperation 步骤未显式设置 timeout 时使用的默认超时时间，单位为分钟。", 60, 1, 1440),
	durationField("operationRuntime", "stepStartRequeue", "步骤启动重排队间隔", "容灾操作步骤刚启动后再次检查状态的默认间隔。", "1s", "1s", "5m"),
	durationField("operationRuntime", "stepRunningRequeue", "步骤运行中重排队间隔", "容灾操作步骤运行中再次检查状态的默认间隔。", "5s", "1s", "5m"),
	durationField("operationRuntime", "defaultRetryInterval", "操作默认重试间隔", "DisasterOperation 未显式设置 retryPolicy.retryIntervalSeconds 时的默认重试间隔。", "5s", "1s", "1h"),

	durationField("instanceRuntime", "transitionWatchdogTimeout", "状态转换看门狗超时", "DisasterInstance 状态转换超过该时间仍未推进时触发看门狗保护；该值不得小于状态转换看门狗下限。", "2m", "30s", "24h"),
	durationField("instanceRuntime", "minTransitionWatchdogTimeout", "状态转换看门狗下限", "状态转换看门狗允许配置的最小超时，用于防止过小值导致误判。", "30s", "10s", "1h"),
	durationField("instanceRuntime", "initializingRequeue", "初始化状态重排队间隔", "DisasterInstance 处于初始化阶段时的默认重排队间隔。", "10s", "1s", "10m"),
	durationField("instanceRuntime", "steadyRequeue", "稳态重排队间隔", "DisasterInstance 处于稳定状态时的默认重排队间隔。", "1m", "5s", "30m"),
	durationField("instanceRuntime", "failedRequeue", "失败状态重排队间隔", "DisasterInstance 处于失败状态时的默认重排队间隔。", "1m", "5s", "30m"),

	durationField("syncRuntime", "schedulerUpdateTimeout", "调度器更新超时", "DataSync 调度器更新 cron 表达式和同步策略时使用的上下文超时。", "30s", "1s", "10m"),
	durationField("syncRuntime", "backupObserveRequeue", "同步备份创建观察间隔", "DataSync 创建备份后第一次观察备份状态的默认重排队间隔。", "2s", "1s", "5m"),
	durationField("syncRuntime", "backupInProgressRequeue", "同步备份进行中观察间隔", "DataSync 观察进行中备份的默认重排队间隔。", "5s", "1s", "5m"),
	durationField("syncRuntime", "historyMissingRequeue", "同步历史缺失观察间隔", "DataSync 或 ResourceSync 未找到期望历史记录时再次检查的默认间隔。", "5s", "1s", "5m"),
	durationField("syncRuntime", "restoreObserveRequeue", "同步恢复观察间隔", "ResourceSync 创建恢复后观察恢复状态的默认重排队间隔。", "10s", "1s", "5m"),
	intField("syncRuntime", "historyRetention", "同步历史保留条数", "DataSync 和 ResourceSync 历史记录保留数量，超过该数量后由 controller 清理旧历史。", 20, 1, 500),

	durationField("storageRuntime", "requeueInterval", "存储状态重排队间隔", "StorageRepository 执行 S3 校验、统计刷新和状态更新后的默认重排队间隔。", "10s", "5s", "1h"),

	durationField("clusterRuntime", "reconcileInterval", "集群对账间隔", "DisasterCluster 常规对账完成后的默认重排队间隔。", "1m", "10s", "1h"),
	durationField("clusterRuntime", "deletionRetryInterval", "集群删除重试间隔", "DisasterCluster 删除、Velero 卸载或清理失败后的默认重试间隔。", "10s", "1s", "10m"),
	durationField("clusterRuntime", "veleroInstallTimeout", "Velero 安装超时", "后续 Velero Helm install 或 upgrade 调用的默认超时，不会修改已经运行中的 Helm 命令。", "10m", "1m", "2h"),
	readOnlyDurationField("clusterRuntime", "veleroZombieLockThreshold", "Velero Helm 锁保护阈值", "Velero Helm release 长时间停留在 pending 状态时判定为 zombie lock 的阈值，只允许后端展示，不允许页面直接修改。", "10m", "5m", "24h"),
}

var (
	groupsByKey = buildGroupsByKey()
	fieldsByKey = buildFieldsByKey()
)

func durationField(groupKey, name, displayName, description, defaultValue, min, max string) fieldDefinition {
	return fieldDefinition{
		Key:          groupKey + "." + name,
		GroupKey:     groupKey,
		Name:         displayName,
		Description:  description,
		DefaultValue: defaultValue,
		DataType:     FieldValueTypeDuration,
		Editable:     true,
		EffectMode:   EffectModeHot,
		Min:          min,
		Max:          max,
	}
}

func readOnlyDurationField(groupKey, name, displayName, description, defaultValue, min, max string) fieldDefinition {
	field := durationField(groupKey, name, displayName, description, defaultValue, min, max)
	field.Editable = false
	return field
}

func intField(groupKey, name, displayName, description string, defaultValue, min, max int) fieldDefinition {
	return fieldDefinition{
		Key:          groupKey + "." + name,
		GroupKey:     groupKey,
		Name:         displayName,
		Description:  description,
		DefaultValue: defaultValue,
		DataType:     FieldValueTypeInt,
		Editable:     true,
		EffectMode:   EffectModeHot,
		Min:          min,
		Max:          max,
	}
}

func buildGroupsByKey() map[string]groupDefinition {
	out := make(map[string]groupDefinition, len(groupDefinitions))
	for _, group := range groupDefinitions {
		out[group.Key] = group
	}
	return out
}

func buildFieldsByKey() map[string]fieldDefinition {
	out := make(map[string]fieldDefinition, len(fieldDefinitions))
	for _, field := range fieldDefinitions {
		out[field.Key] = field
	}
	return out
}

func fieldDefinitionByKey(key string) (fieldDefinition, bool) {
	field, ok := fieldsByKey[key]
	return field, ok
}

func fieldDTOs(doc *configDocument) []FieldDTO {
	values := map[string]interface{}{}
	if doc != nil && doc.Values != nil {
		values = doc.Values
	}

	out := make([]FieldDTO, 0, len(fieldDefinitions))
	for _, field := range fieldDefinitions {
		group := groupsByKey[field.GroupKey]
		value, exists := values[field.Key]
		if !exists {
			value = field.DefaultValue
		}
		out = append(out, field.toDTO(group, value))
	}
	return out
}

func groupedSnapshot(doc *configDocument) SnapshotDTO {
	if doc == nil {
		doc = defaultConfigDocument()
	}

	values := map[string]interface{}{}
	if doc.Values != nil {
		values = doc.Values
	}

	groups := make([]GroupDTO, 0, len(groupDefinitions))
	for _, group := range groupDefinitions {
		dto := GroupDTO{
			Key:         group.Key,
			Name:        group.Name,
			Description: group.Description,
			Fields:      []FieldDTO{},
		}
		for _, field := range fieldDefinitions {
			if field.GroupKey != group.Key {
				continue
			}
			value, exists := values[field.Key]
			if !exists {
				value = field.DefaultValue
			}
			dto.Fields = append(dto.Fields, field.toDTO(group, value))
		}
		groups = append(groups, dto)
	}

	return SnapshotDTO{
		SchemaVersion: doc.SchemaVersion,
		UpdatedAt:     doc.UpdatedAt,
		UpdatedBy:     doc.UpdatedBy,
		Groups:        groups,
	}
}

func (f fieldDefinition) toDTO(group groupDefinition, value interface{}) FieldDTO {
	return FieldDTO{
		Key:          f.Key,
		Name:         f.Name,
		Description:  f.Description,
		Value:        normalizeStoredValueForDTO(f, value),
		DefaultValue: f.DefaultValue,
		DataType:     f.DataType,
		Editable:     f.Editable,
		EffectMode:   f.EffectMode,
		Min:          f.Min,
		Max:          f.Max,
		EnumValues:   append([]string(nil), f.EnumValues...),
		GroupKey:     group.Key,
		GroupName:    group.Name,
	}
}

func compareFieldDTO(a, b FieldDTO, field string) int {
	switch strings.TrimSpace(field) {
	case "key":
		return strings.Compare(a.Key, b.Key)
	case "name":
		return strings.Compare(a.Name, b.Name)
	case "groupKey":
		return strings.Compare(a.GroupKey, b.GroupKey)
	case "groupName":
		return strings.Compare(a.GroupName, b.GroupName)
	case "dataType":
		return strings.Compare(string(a.DataType), string(b.DataType))
	case "effectMode":
		return strings.Compare(string(a.EffectMode), string(b.EffectMode))
	case "editable":
		return compareBool(a.Editable, b.Editable)
	case "value":
		return strings.Compare(fmt.Sprint(a.Value), fmt.Sprint(b.Value))
	default:
		return strings.Compare(a.Key, b.Key)
	}
}

func compareBool(a, b bool) int {
	if a == b {
		return 0
	}
	if !a && b {
		return -1
	}
	return 1
}

func sortFieldsByKey(fields []FieldDTO) {
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})
}
