package velerohooks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const SensitiveParameterErrorCode = "VeleroHookSensitiveParameter"

var (
	maxBackupExecTimeout  = 10 * time.Minute
	maxRestoreExecTimeout = 10 * time.Minute
	maxRestoreWaitTimeout = 30 * time.Minute
	maxRestoreInitTimeout = 30 * time.Minute
)

type ValidationError struct {
	Code      string
	FieldPath string
	Message   string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func ErrorMeta(err error) map[string]any {
	validationErr, ok := err.(*ValidationError)
	if !ok || validationErr == nil {
		return nil
	}
	meta := map[string]any{}
	if validationErr.Code != "" {
		meta["errorCode"] = validationErr.Code
	}
	if validationErr.FieldPath != "" {
		meta["fieldPath"] = validationErr.FieldPath
	}
	return meta
}

func IsNullOrEmptyObject(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed == "" || trimmed == "null" || trimmed == "{}"
}

type DisasterVeleroHooksRequest struct {
	DataBackup  *velerov1.BackupHooks  `json:"dataBackup,omitempty"`
	DataRestore *velerov1.RestoreHooks `json:"dataRestore,omitempty"`

	DataBackupSet    bool `json:"-"`
	DataBackupClear  bool `json:"-"`
	DataRestoreSet   bool `json:"-"`
	DataRestoreClear bool `json:"-"`
}

func (r *DisasterVeleroHooksRequest) UnmarshalJSON(data []byte) error {
	type alias DisasterVeleroHooksRequest
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*r = DisasterVeleroHooksRequest(decoded)
	if dataBackupRaw, ok := raw["dataBackup"]; ok {
		r.DataBackupSet = true
		if IsNullOrEmptyObject(dataBackupRaw) {
			r.DataBackup = nil
			r.DataBackupClear = true
		}
	}
	if dataRestoreRaw, ok := raw["dataRestore"]; ok {
		r.DataRestoreSet = true
		if IsNullOrEmptyObject(dataRestoreRaw) {
			r.DataRestore = nil
			r.DataRestoreClear = true
		}
	}
	return nil
}

func (r *DisasterVeleroHooksRequest) ToCRD() *dapisv1.DisasterVeleroHooks {
	if r == nil {
		return nil
	}
	return &dapisv1.DisasterVeleroHooks{
		DataBackup:  r.DataBackup,
		DataRestore: r.DataRestore,
	}
}

type DisasterVeleroHooksPatch struct {
	Set   bool
	Clear bool
	Value *DisasterVeleroHooksRequest
}

func DecodeDisasterVeleroHooksPatch(raw json.RawMessage) (DisasterVeleroHooksPatch, error) {
	if raw == nil {
		return DisasterVeleroHooksPatch{}, nil
	}
	if IsNullOrEmptyObject(raw) {
		return DisasterVeleroHooksPatch{Set: true, Clear: true}, nil
	}
	var value DisasterVeleroHooksRequest
	if err := json.Unmarshal(raw, &value); err != nil {
		return DisasterVeleroHooksPatch{}, err
	}
	return DisasterVeleroHooksPatch{Set: true, Value: &value}, nil
}

func ApplyDisasterVeleroHooksPatch(target **dapisv1.DisasterVeleroHooks, patch DisasterVeleroHooksPatch) {
	if !patch.Set {
		return
	}
	if patch.Clear {
		*target = nil
		return
	}
	if *target == nil {
		*target = &dapisv1.DisasterVeleroHooks{}
	}
	if patch.Value == nil {
		return
	}
	if patch.Value.DataBackupSet {
		if patch.Value.DataBackupClear {
			(*target).DataBackup = nil
		} else {
			(*target).DataBackup = patch.Value.DataBackup
		}
	}
	if patch.Value.DataRestoreSet {
		if patch.Value.DataRestoreClear {
			(*target).DataRestore = nil
		} else {
			(*target).DataRestore = patch.Value.DataRestore
		}
	}
	if (*target).DataBackup == nil && (*target).DataRestore == nil {
		*target = nil
	}
}

func ValidateBackupHooks(hooks *velerov1.BackupHooks, fieldPath string) error {
	if hooks == nil {
		return nil
	}
	for i := range hooks.Resources {
		resource := hooks.Resources[i]
		resourcePath := fmt.Sprintf("%s.resources[%d]", fieldPath, i)
		if err := validatePodResourceScope(resource.IncludedResources, resourcePath+".includedResources"); err != nil {
			return err
		}
		for j := range resource.PreHooks {
			if err := validateBackupResourceHook(resource.PreHooks[j], fmt.Sprintf("%s.pre[%d]", resourcePath, j)); err != nil {
				return err
			}
		}
		for j := range resource.PostHooks {
			if err := validateBackupResourceHook(resource.PostHooks[j], fmt.Sprintf("%s.post[%d]", resourcePath, j)); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidateRestoreHooks(hooks *velerov1.RestoreHooks, fieldPath string) error {
	if hooks == nil {
		return nil
	}
	for i := range hooks.Resources {
		resource := hooks.Resources[i]
		resourcePath := fmt.Sprintf("%s.resources[%d]", fieldPath, i)
		if err := validatePodResourceScope(resource.IncludedResources, resourcePath+".includedResources"); err != nil {
			return err
		}
		for j := range resource.PostHooks {
			if err := validateRestoreResourceHook(resource.PostHooks[j], fmt.Sprintf("%s.postHooks[%d]", resourcePath, j)); err != nil {
				return err
			}
		}
	}
	return nil
}

func ValidateDisasterVeleroHooks(hooks *dapisv1.DisasterVeleroHooks, fieldPath string) error {
	if hooks == nil {
		return nil
	}
	if err := ValidateBackupHooks(hooks.DataBackup, fieldPath+".dataBackup"); err != nil {
		return err
	}
	return ValidateRestoreHooks(hooks.DataRestore, fieldPath+".dataRestore")
}

func validateBackupResourceHook(hook velerov1.BackupResourceHook, fieldPath string) error {
	if hook.Exec == nil {
		return validation(fieldPath+".exec", "backup hook exec is required")
	}
	return validateBackupExecHook(hook.Exec, fieldPath+".exec")
}

func validateBackupExecHook(exec *velerov1.ExecHook, fieldPath string) error {
	if len(exec.Command) == 0 {
		return validation(fieldPath+".command", "backup hook exec.command is required")
	}
	if err := validateCommandStrings(exec.Command, fieldPath+".command"); err != nil {
		return err
	}
	if err := validateOnError(exec.OnError, fieldPath+".onError"); err != nil {
		return err
	}
	return validateDuration(exec.Timeout, maxBackupExecTimeout, "Backup exec hook timeout 最大值为 10m", fieldPath+".timeout")
}

func validateRestoreResourceHook(hook velerov1.RestoreResourceHook, fieldPath string) error {
	if hook.Exec != nil {
		if err := validateRestoreExecHook(hook.Exec, fieldPath+".exec"); err != nil {
			return err
		}
	}
	if hook.Init != nil {
		if err := validateRestoreInitHook(hook.Init, fieldPath+".init"); err != nil {
			return err
		}
	}
	if hook.Exec == nil && hook.Init == nil {
		return validation(fieldPath, "restore hook exec or init is required")
	}
	return nil
}

func validateRestoreExecHook(exec *velerov1.ExecRestoreHook, fieldPath string) error {
	if len(exec.Command) == 0 {
		return validation(fieldPath+".command", "restore hook exec.command is required")
	}
	if err := validateCommandStrings(exec.Command, fieldPath+".command"); err != nil {
		return err
	}
	if err := validateOnError(exec.OnError, fieldPath+".onError"); err != nil {
		return err
	}
	if err := validateDuration(exec.ExecTimeout, maxRestoreExecTimeout, "Restore exec hook execTimeout 最大值为 10m", fieldPath+".execTimeout"); err != nil {
		return err
	}
	return validateDuration(exec.WaitTimeout, maxRestoreWaitTimeout, "Restore exec hook waitTimeout 最大值为 30m", fieldPath+".waitTimeout")
}

func validateRestoreInitHook(init *velerov1.InitRestoreHook, fieldPath string) error {
	if err := validateDuration(init.Timeout, maxRestoreInitTimeout, "Restore init hook timeout 最大值为 30m", fieldPath+".timeout"); err != nil {
		return err
	}
	for i := range init.InitContainers {
		raw := init.InitContainers[i]
		if len(raw.Raw) == 0 {
			continue
		}
		var container corev1.Container
		if err := json.Unmarshal(raw.Raw, &container); err != nil {
			return validation(fmt.Sprintf("%s.initContainers[%d]", fieldPath, i), fmt.Sprintf("invalid initContainer: %v", err))
		}
		base := fmt.Sprintf("%s.initContainers[%d]", fieldPath, i)
		if err := validateCommandStrings(container.Command, base+".command"); err != nil {
			return err
		}
		if err := validateCommandStrings(container.Args, base+".args"); err != nil {
			return err
		}
		for j := range container.Env {
			if container.Env[j].Value == "" {
				continue
			}
			if err := validateString(container.Env[j].Value, fmt.Sprintf("%s.env[%d].value", base, j)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePodResourceScope(resources []string, fieldPath string) error {
	if len(resources) == 0 {
		return nil
	}
	for _, resource := range resources {
		normalized := strings.ToLower(strings.TrimSpace(resource))
		if normalized == "pod" || normalized == "pods" {
			return nil
		}
	}
	return validation(fieldPath, "Velero Hook 仅支持 Pod 目标，includedResources 为空或包含 pods")
}

func validateCommandStrings(values []string, fieldPath string) error {
	for i, value := range values {
		if err := validateString(value, fmt.Sprintf("%s[%d]", fieldPath, i)); err != nil {
			return err
		}
	}
	return nil
}

func validateString(value, fieldPath string) error {
	if strings.Contains(value, "${testudo.") || strings.Contains(value, "{{") {
		return validation(fieldPath, "第一阶段不支持平台占位符渲染")
	}
	if containsSensitiveParameter(value) {
		return &ValidationError{
			Code:      SensitiveParameterErrorCode,
			FieldPath: fieldPath,
			Message:   "sensitive hook parameter is not allowed; use Secret env, Secret volume, or valueFrom instead",
		}
	}
	return nil
}

func containsSensitiveParameter(value string) bool {
	normalized := strings.ToLower(value)
	for _, token := range []string{"password=", "passwd=", "token=", "access_key=", "access-key=", "secret=", "api_key=", "apikey="} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func validateOnError(onError velerov1.HookErrorMode, fieldPath string) error {
	if onError == "" || onError == velerov1.HookErrorModeFail || onError == velerov1.HookErrorModeContinue {
		return nil
	}
	return validation(fieldPath, "onError must be Fail or Continue")
}

func validateDuration(duration metav1.Duration, max time.Duration, message, fieldPath string) error {
	if duration.Duration == 0 {
		return nil
	}
	if duration.Duration < 0 {
		return validation(fieldPath, "hook timeout must be positive")
	}
	if duration.Duration > max {
		return validation(fieldPath, message)
	}
	return nil
}

func validation(fieldPath, message string) error {
	return &ValidationError{FieldPath: fieldPath, Message: message}
}
