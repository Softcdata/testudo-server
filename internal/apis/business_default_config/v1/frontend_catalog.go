package businessdefaultconfig

import (
	"fmt"
	"sort"
	"strings"
)

type frontendSpecFieldDefinition struct {
	Key                 string
	Name                string
	Description         string
	ResourceKind        string
	SpecPath            string
	RequestPath         string
	ConfigGroupKey      string
	ConfigKey           string
	DataType            FieldValueType
	DefaultValue        interface{}
	Unit                string
	Required            bool
	Editable            bool
	ServerSupported     bool
	SupportedOperations []string
	Min                 interface{}
	Max                 interface{}
	EnumValues          []string
	Example             interface{}
	Note                string
}

var frontendConfigGroupNames = map[string]string{
	"backup":    "备份默认值",
	"restore":   "恢复默认值",
	"instance":  "实例默认值",
	"operation": "容灾操作默认值",
	"drill":     "演练默认值",
}

var frontendSpecFieldDefinitions = []frontendSpecFieldDefinition{
	{
		Key:             "backup.timeout",
		Name:            "备份超时",
		Description:     "AppBackup 等待单次 Velero Backup 完成的最长时间。前端读取业务默认值后应在创建或更新备份时写入 AppBackup.spec.timeout；当前 server AppBackup 请求 DTO 尚未接收该字段。",
		ResourceKind:    "AppBackup",
		SpecPath:        "spec.timeout",
		RequestPath:     "timeout",
		ConfigGroupKey:  "backup",
		ConfigKey:       "backup.timeout",
		DataType:        FieldValueTypeDuration,
		DefaultValue:    "2h",
		Unit:            "duration",
		Editable:        true,
		ServerSupported: false,
		Min:             "1m",
		Max:             "24h",
		Example:         "2h",
		Note:            "operator 已支持 CRD 字段；server AppBackup 创建/更新接口需要 companion change 后才能透传。",
	},
	{
		Key:             "backup.schedule",
		Name:            "备份调度表达式",
		Description:     "创建 AppBackup 时使用的默认 cron 表达式；手动备份可使用 @manual。",
		ResourceKind:    "AppBackup",
		SpecPath:        "spec.schedule",
		RequestPath:     "schedule",
		ConfigGroupKey:  "backup",
		ConfigKey:       "backup.schedule",
		DataType:        FieldValueTypeString,
		DefaultValue:    "0 2 * * *",
		Editable:        true,
		ServerSupported: true,
		Example:         "0 2 * * *",
	},
	{
		Key:             "backup.ttl",
		Name:            "备份保留时间",
		Description:     "Velero Backup 的保留时长，超过后由 Velero 按过期策略处理。",
		ResourceKind:    "AppBackup",
		SpecPath:        "spec.template.ttl",
		RequestPath:     "ttl",
		ConfigGroupKey:  "backup",
		ConfigKey:       "backup.ttl",
		DataType:        FieldValueTypeDuration,
		DefaultValue:    "720h",
		Unit:            "duration",
		Editable:        true,
		ServerSupported: true,
		Min:             "1h",
		Max:             "8760h",
		Example:         "720h",
	},
	{
		Key:             "backup.skipImmediately",
		Name:            "跳过首次立即备份",
		Description:     "创建定时备份后是否跳过首次立即执行，只保留后续调度执行。",
		ResourceKind:    "AppBackup",
		SpecPath:        "spec.skipImmediately",
		RequestPath:     "skipImmediately",
		ConfigGroupKey:  "backup",
		ConfigKey:       "backup.skipImmediately",
		DataType:        FieldValueTypeBool,
		DefaultValue:    false,
		Editable:        true,
		ServerSupported: true,
		Example:         false,
	},
	{
		Key:             "backup.defaultVolumesToFsBackup",
		Name:            "默认文件系统卷备份",
		Description:     "未单独声明卷备份方式时，是否默认使用文件系统备份卷数据。",
		ResourceKind:    "AppBackup",
		SpecPath:        "spec.template.defaultVolumesToFsBackup",
		RequestPath:     "defaultVolumesToFsBackup",
		ConfigGroupKey:  "backup",
		ConfigKey:       "backup.defaultVolumesToFsBackup",
		DataType:        FieldValueTypeBool,
		DefaultValue:    true,
		Editable:        true,
		ServerSupported: true,
		Example:         true,
	},
	{
		Key:             "restore.timeout",
		Name:            "恢复超时",
		Description:     "AppRestore 等待恢复完成的最长时间。当前 server 的 restore 请求字段 timeout 会写入 Velero Restore template.itemOperationTimeout。",
		ResourceKind:    "AppRestore",
		SpecPath:        "spec.timeout",
		RequestPath:     "timeout",
		ConfigGroupKey:  "restore",
		ConfigKey:       "restore.timeout",
		DataType:        FieldValueTypeDuration,
		DefaultValue:    "1h",
		Unit:            "duration",
		Editable:        true,
		ServerSupported: true,
		Min:             "1m",
		Max:             "24h",
		Example:         "1h",
		Note:            "当前 server 出于兼容原因把 request.timeout 映射到 spec.template.itemOperationTimeout。",
	},
	{
		Key:             "restore.itemOperationTimeout",
		Name:            "恢复单资源操作超时",
		Description:     "Velero Restore 单个资源恢复操作的最长等待时间，用于控制长时间卡住的 item operation。",
		ResourceKind:    "AppRestore",
		SpecPath:        "spec.template.itemOperationTimeout",
		RequestPath:     "timeout",
		ConfigGroupKey:  "restore",
		ConfigKey:       "restore.itemOperationTimeout",
		DataType:        FieldValueTypeDuration,
		DefaultValue:    "4h",
		Unit:            "duration",
		Editable:        true,
		ServerSupported: true,
		Min:             "1m",
		Max:             "24h",
		Example:         "4h",
	},
	{
		Key:             "restore.restorePVs",
		Name:            "恢复 PV",
		Description:     "创建 AppRestore 时是否恢复持久卷。",
		ResourceKind:    "AppRestore",
		SpecPath:        "spec.template.restorePVs",
		RequestPath:     "restorePVs",
		ConfigGroupKey:  "restore",
		ConfigKey:       "restore.restorePVs",
		DataType:        FieldValueTypeBool,
		DefaultValue:    true,
		Editable:        true,
		ServerSupported: true,
		Example:         true,
	},
	{
		Key:             "restore.existingResourcePolicy",
		Name:            "已有资源处理策略",
		Description:     "恢复目标集群已存在同名资源时的处理策略。",
		ResourceKind:    "AppRestore",
		SpecPath:        "spec.template.existingResourcePolicy",
		RequestPath:     "existingResourcePolicy",
		ConfigGroupKey:  "restore",
		ConfigKey:       "restore.existingResourcePolicy",
		DataType:        FieldValueTypeEnum,
		DefaultValue:    "Update",
		Editable:        true,
		ServerSupported: true,
		EnumValues:      []string{"None", "Update"},
		Example:         "Update",
	},
	{
		Key:             "restore.restorePolicy",
		Name:            "恢复策略",
		Description:     "实例、演练或恢复链路使用的资源选择、执行策略、StorageClass/IngressClass 映射和资源修改规则。",
		ResourceKind:    "DisasterInstance",
		SpecPath:        "spec.restorePolicy",
		RequestPath:     "restorePolicy",
		ConfigGroupKey:  "restore",
		ConfigKey:       "restore.restorePolicy",
		DataType:        FieldValueTypeObject,
		DefaultValue:    map[string]interface{}{},
		Editable:        true,
		ServerSupported: true,
		Example:         map[string]interface{}{},
		Note:            "AppRestore 直接创建接口当前不接收 restorePolicy；DisasterInstance 和 DisasterDrill 接口支持该结构。",
	},
	{
		Key:             "instance.operationTimeoutMinutes",
		Name:            "实例默认操作超时",
		Description:     "DisasterInstance 执行 failover、reprotect、同步等容灾操作时的默认超时时间，单次 DisasterOperation.timeoutMinutes 可覆盖。",
		ResourceKind:    "DisasterInstance",
		SpecPath:        "spec.operationTimeoutMinutes",
		RequestPath:     "operationTimeoutMinutes",
		ConfigGroupKey:  "instance",
		ConfigKey:       "instance.operationTimeoutMinutes",
		DataType:        FieldValueTypeInt,
		DefaultValue:    60,
		Unit:            "minutes",
		Editable:        true,
		ServerSupported: true,
		Min:             1,
		Max:             1440,
		Example:         60,
	},
	{
		Key:             "instance.podRestoreMethod",
		Name:            "Pod 恢复方式",
		Description:     "Standby 模式下 Pod 的处理方式，默认 replica。",
		ResourceKind:    "DisasterInstance",
		SpecPath:        "spec.podRestoreMethod",
		RequestPath:     "podRestoreMethod",
		ConfigGroupKey:  "instance",
		ConfigKey:       "instance.podRestoreMethod",
		DataType:        FieldValueTypeEnum,
		DefaultValue:    "replica",
		Editable:        true,
		ServerSupported: true,
		EnumValues:      []string{"replica", "initContainer"},
		Example:         "replica",
	},
	{
		Key:             "instance.skipPodReadyCheck",
		Name:            "默认跳过 Pod 就绪校验",
		Description:     "实例级默认策略，控制容灾切换时是否默认跳过 Pod readyReplicas 校验；单次操作参数可覆盖。",
		ResourceKind:    "DisasterInstance",
		SpecPath:        "spec.skipPodReadyCheck",
		RequestPath:     "skipPodReadyCheck",
		ConfigGroupKey:  "instance",
		ConfigKey:       "instance.skipPodReadyCheck",
		DataType:        FieldValueTypeBool,
		DefaultValue:    false,
		Editable:        true,
		ServerSupported: true,
		Example:         false,
	},
	{
		Key:                 "operation.timeoutMinutes",
		Name:                "单次操作超时",
		Description:         "本次 DisasterOperation 的超时时间，优先级高于实例默认 operationTimeoutMinutes。",
		ResourceKind:        "DisasterOperation",
		SpecPath:            "spec.timeoutMinutes",
		RequestPath:         "config.timeoutMinutes",
		ConfigGroupKey:      "operation",
		ConfigKey:           "operation.timeoutMinutes",
		DataType:            FieldValueTypeInt,
		DefaultValue:        60,
		Unit:                "minutes",
		Editable:            true,
		ServerSupported:     true,
		SupportedOperations: []string{"failover", "reprotect", "pause", "resume", "sync-data", "sync-resource", "cancel"},
		Min:                 1,
		Max:                 1440,
		Example:             60,
	},
	{
		Key:            "operation.retryPolicy",
		Name:           "单次操作重试策略",
		Description:    "DisasterOperation 的重试次数与重试间隔配置。",
		ResourceKind:   "DisasterOperation",
		SpecPath:       "spec.retryPolicy",
		RequestPath:    "config.retryPolicy",
		ConfigGroupKey: "operation",
		ConfigKey:      "operation.retryPolicy",
		DataType:       FieldValueTypeObject,
		DefaultValue: map[string]interface{}{
			"maxRetries":           0,
			"retryIntervalSeconds": 5,
		},
		Editable:        true,
		ServerSupported: false,
		Example: map[string]interface{}{
			"maxRetries":           2,
			"retryIntervalSeconds": 10,
		},
		Note: "当前实例 action 接口不解析 config.retryPolicy；组 action 会从 DisasterGroup policy 继承 retryPolicy。",
	},
	{
		Key:                 "operation.waitUntilReady",
		Name:                "等待 Pod 就绪",
		Description:         "failover 时是否等待 Pod readyReplicas 满足期望副本数。建议优先使用 skipPodReadyCheck 显式表达。",
		ResourceKind:        "DisasterOperation",
		SpecPath:            "spec.waitUntilReady",
		RequestPath:         "config.waitUntilReady",
		ConfigGroupKey:      "operation",
		ConfigKey:           "operation.waitUntilReady",
		DataType:            FieldValueTypeBool,
		DefaultValue:        false,
		Editable:            true,
		ServerSupported:     true,
		SupportedOperations: []string{"failover", "reprotect", "sync-data", "sync-resource", "cancel"},
		Example:             false,
		Note:                "server 会同时写入兼容字段 skipPodReadyCheck = !waitUntilReady。",
	},
	{
		Key:                 "operation.skipPodReadyCheck",
		Name:                "跳过 Pod 就绪校验",
		Description:         "单次操作级覆盖项，true 表示跳过 readyReplicas 校验，仅检查期望副本配置是否下发。",
		ResourceKind:        "DisasterOperation",
		SpecPath:            "spec.skipPodReadyCheck",
		RequestPath:         "config.skipPodReadyCheck",
		ConfigGroupKey:      "operation",
		ConfigKey:           "operation.skipPodReadyCheck",
		DataType:            FieldValueTypeBool,
		DefaultValue:        false,
		Editable:            true,
		ServerSupported:     true,
		SupportedOperations: []string{"failover", "reprotect", "sync-data", "sync-resource", "cancel"},
		Example:             false,
		Note:                "当 skipPodReadyCheck 与 waitUntilReady 同时提交时，server 以 skipPodReadyCheck 为准。",
	},
	{
		Key:                 "operation.skipFinalSync",
		Name:                "跳过最终同步",
		Description:         "failover 或 reprotect 前是否跳过最终同步步骤。",
		ResourceKind:        "DisasterOperation",
		SpecPath:            "spec.skipFinalSync",
		RequestPath:         "config.skipFinalSync",
		ConfigGroupKey:      "operation",
		ConfigKey:           "operation.skipFinalSync",
		DataType:            FieldValueTypeBool,
		DefaultValue:        false,
		Editable:            true,
		ServerSupported:     true,
		SupportedOperations: []string{"failover", "reprotect"},
		Example:             false,
	},
	{
		Key:                 "operation.skipScaleDownSource",
		Name:                "跳过源集群缩零",
		Description:         "failover 时是否跳过源集群工作负载缩零步骤，仅对 failover 生效。",
		ResourceKind:        "DisasterOperation",
		SpecPath:            "spec.skipScaleDownSource",
		RequestPath:         "config.skipScaleDownSource",
		ConfigGroupKey:      "operation",
		ConfigKey:           "operation.skipScaleDownSource",
		DataType:            FieldValueTypeBool,
		DefaultValue:        false,
		Editable:            true,
		ServerSupported:     true,
		SupportedOperations: []string{"failover"},
		Example:             false,
	},
	{
		Key:             "drill.waitUntilReady",
		Name:            "演练等待 Pod 就绪",
		Description:     "演练恢复后是否等待 Pod 就绪。",
		ResourceKind:    "DisasterDrill",
		SpecPath:        "spec.waitUntilReady",
		RequestPath:     "waitUntilReady",
		ConfigGroupKey:  "drill",
		ConfigKey:       "drill.waitUntilReady",
		DataType:        FieldValueTypeBool,
		DefaultValue:    false,
		Editable:        true,
		ServerSupported: true,
		Example:         false,
	},
	{
		Key:             "drill.skipValidation",
		Name:            "跳过演练前置校验",
		Description:     "创建演练时是否跳过实例、集群可达性、备份可用性等前置校验。",
		ResourceKind:    "DisasterDrill",
		SpecPath:        "spec.skipValidation",
		RequestPath:     "skipValidation",
		ConfigGroupKey:  "drill",
		ConfigKey:       "drill.skipValidation",
		DataType:        FieldValueTypeBool,
		DefaultValue:    false,
		Editable:        true,
		ServerSupported: true,
		Example:         false,
	},
	{
		Key:             "drill.namespaceMapping",
		Name:            "演练命名空间映射",
		Description:     "演练恢复时源命名空间到目标命名空间的默认映射。",
		ResourceKind:    "DisasterDrill",
		SpecPath:        "spec.namespaceMapping",
		RequestPath:     "namespaceMapping",
		ConfigGroupKey:  "drill",
		ConfigKey:       "drill.namespaceMapping",
		DataType:        FieldValueTypeObject,
		DefaultValue:    map[string]interface{}{},
		Editable:        true,
		ServerSupported: true,
		Example:         map[string]interface{}{"default": "default-drill"},
	},
	{
		Key:             "drill.restorePolicy",
		Name:            "演练恢复策略",
		Description:     "演练级 restorePolicy 覆盖；未提供时默认继承 DisasterInstance 的 restorePolicy。",
		ResourceKind:    "DisasterDrill",
		SpecPath:        "spec.restorePolicy",
		RequestPath:     "restorePolicy",
		ConfigGroupKey:  "drill",
		ConfigKey:       "drill.restorePolicy",
		DataType:        FieldValueTypeObject,
		DefaultValue:    map[string]interface{}{},
		Editable:        true,
		ServerSupported: true,
		Example:         map[string]interface{}{},
	},
	{
		Key:             "drill.cleanup",
		Name:            "演练清理",
		Description:     "触发演练资源清理的开关。该字段属于演练动作字段，通常不作为创建时默认值写入。",
		ResourceKind:    "DisasterDrill",
		SpecPath:        "spec.cleanup",
		RequestPath:     "cleanup",
		ConfigGroupKey:  "drill",
		ConfigKey:       "drill.cleanup",
		DataType:        FieldValueTypeBool,
		DefaultValue:    false,
		Editable:        true,
		ServerSupported: false,
		Example:         false,
		Note:            "server 当前通过独立 cleanup action 触发清理，不在创建请求中接收 cleanup。",
	},
}

func frontendSpecFieldDTOs() []FrontendSpecFieldDTO {
	out := make([]FrontendSpecFieldDTO, 0, len(frontendSpecFieldDefinitions))
	for _, field := range frontendSpecFieldDefinitions {
		out = append(out, field.toDTO(field.DefaultValue))
	}
	return out
}

func (f frontendSpecFieldDefinition) toDTO(value interface{}) FrontendSpecFieldDTO {
	return FrontendSpecFieldDTO{
		Key:                 f.Key,
		Value:               value,
		Name:                f.Name,
		Description:         f.Description,
		ResourceKind:        f.ResourceKind,
		RequestPath:         f.RequestPath,
		SpecPath:            f.SpecPath,
		ConfigGroupKey:      f.ConfigGroupKey,
		ConfigKey:           f.ConfigKey,
		DataType:            f.DataType,
		Editable:            f.Editable,
		ServerSupported:     f.ServerSupported,
		SupportedOperations: cloneStrings(f.SupportedOperations),
		KeySegments:         splitFieldPath(f.Key),
		RequestPathSegments: splitFieldPath(f.RequestPath),
		SpecPathSegments:    splitFieldPath(f.SpecPath),
		APIUsages:           frontendSpecAPIUsages(f),
		Note:                f.Note,
	}
}

func compareFrontendSpecFieldDTO(a, b FrontendSpecFieldDTO, field string) int {
	switch strings.TrimSpace(field) {
	case "key":
		return strings.Compare(a.Key, b.Key)
	case "value":
		return strings.Compare(fmt.Sprint(a.Value), fmt.Sprint(b.Value))
	default:
		return strings.Compare(a.Key, b.Key)
	}
}

func sortFrontendSpecFieldsByKey(fields []FrontendSpecFieldDTO) {
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})
}

func frontendSpecFieldMatchesKeyword(field FrontendSpecFieldDTO, keyword string) bool {
	candidates := []string{
		field.Key,
		field.Name,
		field.Description,
		field.ResourceKind,
		field.RequestPath,
		field.SpecPath,
		fmt.Sprint(field.Value),
	}
	for _, usage := range field.APIUsages {
		candidates = append(candidates, usage.Method, usage.Path, usage.Operation, usage.Description)
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), keyword) {
			return true
		}
	}
	return false
}

func frontendSpecFieldMap(fields []FrontendSpecFieldDTO) map[string]FrontendSpecFieldDTO {
	out := make(map[string]FrontendSpecFieldDTO, len(fields))
	for _, field := range fields {
		out[field.Key] = field
	}
	return out
}

func splitFieldPath(path string) []string {
	parts := strings.Split(strings.TrimSpace(path), ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func frontendSpecAPIUsages(field frontendSpecFieldDefinition) []FrontendSpecAPIUsageDTO {
	key := strings.TrimSpace(field.Key)
	requestPath := strings.TrimSpace(field.RequestPath)
	resourceKind := strings.TrimSpace(field.ResourceKind)
	switch {
	case strings.HasPrefix(key, "backup."):
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/appbackups.testudo.softcdata.com/v1/appbackups", requestPath, resourceKind, "", "创建 AppBackup 时使用"),
			apiUsage("PUT", "/apis/appbackups.testudo.softcdata.com/v1/appbackups/:name", requestPath, resourceKind, "", "更新 AppBackup 时使用"),
		}
	case key == "restore.restorePolicy":
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/disasterinstances.testudo.softcdata.com/v1/instances", requestPath, "DisasterInstance", "", "创建容灾实例时写入实例恢复策略"),
			apiUsage("PUT", "/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name", requestPath, "DisasterInstance", "", "更新容灾实例时写入实例恢复策略"),
			apiUsage("POST", "/apis/disasterdrills.testudo.softcdata.com/v1/drills", requestPath, "DisasterDrill", "", "创建演练时写入演练级恢复策略"),
		}
	case strings.HasPrefix(key, "restore."):
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/apprestores.testudo.softcdata.com/v1/apprestores", requestPath, resourceKind, "", "创建 AppRestore 时使用"),
			apiUsage("PUT", "/apis/apprestores.testudo.softcdata.com/v1/apprestores/:name", requestPath, resourceKind, "", "更新 AppRestore 时使用"),
		}
	case strings.HasPrefix(key, "instance."):
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/disasterinstances.testudo.softcdata.com/v1/instances", requestPath, resourceKind, "", "创建容灾实例时使用"),
			apiUsage("PUT", "/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name", requestPath, resourceKind, "", "更新容灾实例时使用"),
		}
	case key == "operation.skipScaleDownSource":
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions", requestPath, resourceKind, "failover", "触发实例 failover 时使用"),
			apiUsage("POST", "/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions", requestPath, resourceKind, "failover", "触发容灾组 failover 时使用"),
		}
	case strings.HasPrefix(key, "operation."):
		return operationAPIUsages(field)
	case key == "drill.cleanup":
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/disasterdrills.testudo.softcdata.com/v1/drills/:name/cleanup", "", resourceKind, "", "触发演练清理动作；当前不在创建请求体中接收 cleanup 字段"),
		}
	case strings.HasPrefix(key, "drill."):
		return []FrontendSpecAPIUsageDTO{
			apiUsage("POST", "/apis/disasterdrills.testudo.softcdata.com/v1/drills", requestPath, resourceKind, "", "创建容灾演练时使用"),
		}
	default:
		return nil
	}
}

func operationAPIUsages(field frontendSpecFieldDefinition) []FrontendSpecAPIUsageDTO {
	ops := field.SupportedOperations
	if len(ops) == 0 {
		ops = []string{""}
	}
	out := make([]FrontendSpecAPIUsageDTO, 0, len(ops)*2)
	for _, op := range ops {
		description := "触发实例容灾操作时使用"
		if op != "" {
			description = "触发实例 " + op + " 操作时使用"
		}
		out = append(out, apiUsage("POST", "/apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/actions", field.RequestPath, field.ResourceKind, op, description))
		description = "触发容灾组容灾操作时使用"
		if op != "" {
			description = "触发容灾组 " + op + " 操作时使用"
		}
		out = append(out, apiUsage("POST", "/apis/disastergroups.testudo.softcdata.com/v1/groups/:name/actions", field.RequestPath, field.ResourceKind, op, description))
	}
	return out
}

func apiUsage(method, path, requestPath, resourceKind, operation, description string) FrontendSpecAPIUsageDTO {
	return FrontendSpecAPIUsageDTO{
		Method:              method,
		Path:                path,
		RequestPath:         requestPath,
		RequestPathSegments: splitFieldPath(requestPath),
		ResourceKind:        resourceKind,
		Operation:           operation,
		Description:         description,
	}
}
