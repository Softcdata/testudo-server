package instance

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	metadata "github.com/softcdata/testudo-operator/pkg/metadata"
	"github.com/softcdata/testudo-server/internal/transport"
	"k8s.io/apimachinery/pkg/api/errors"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const skipScaleDownSourceAnnotation = "testudo.softcdata.com/skip-scale-down-source"

// 9. Execute Action (Failover, Reprotect, Pause, etc.)
func (h *InstanceHandler) executeAction(c context.Context, ctx *app.RequestContext) {
	name := ctx.Param("name")
	var req ExecuteActionRequest
	if err := ctx.BindJSON(&req); err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// Namespace resolution
	ns := ctx.QueryArgs().Peek("namespace")
	namespace := string(ns)
	if namespace == "" {
		var err error
		namespace, err = h.findNamespace(c, name)
		if err != nil {
			if errors.IsNotFound(err) {
				transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
				return
			}
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
	}

	instance, err := h.DisasterClient.DisasterV1().DisasterInstances(namespace).Get(c, name, matev1.GetOptions{})
	if err != nil {
		transport.WriteError(ctx, transport.CodeNotFound, err.Error(), nil)
		return
	}
	// Dispatch based on operation type
	switch req.Operation {
	// Valid operations: failover, reprotect, undo, pause, resume, synconce, syncdata, syncresource
	case "failover", "reprotect", "undo", "pause", "resume", "synconce", "sync-data", "sync-resource", "cancel":
		available, valid, reason, message := validateInstanceOperationAllowed(instance, req.Operation)
		if !valid {
			transport.WriteError(ctx, transport.CodeConflict, message, ValidateTargetDTO{
				TargetName:          name,
				Namespace:           namespace,
				Operation:           req.Operation,
				Valid:               false,
				Reason:              reason,
				Message:             message,
				FsmState:            instance.Status.FsmState,
				AvailableOperations: available,
			})
			return
		}

		// Create DisasterOperation CR
		opName := fmt.Sprintf("%s-%s-%d", req.Operation, name, time.Now().UnixNano())

		// Map frontend action names to CRD enum types
		var opType dapisv1.OperationType
		switch req.Operation {
		case "sync-data":
			opType = dapisv1.OperationTypeSyncData
		case "sync-resource":
			opType = dapisv1.OperationTypeSyncResource
		default:
			opType = dapisv1.OperationType(req.Operation)
		}

		op := dapisv1.DisasterOperation{
			ObjectMeta: matev1.ObjectMeta{
				Name:      opName,
				Namespace: namespace, // In same namespace as instance
				Labels:    map[string]string{"testudo.softcdata.com/instance": name},
			},
			Spec: dapisv1.DisasterOperationSpec{
				InstanceName: name, OperationType: opType,
			},
		}

		if req.Operation == "failover" || req.Operation == "reprotect" {
			force, _ := req.Config["force"].(bool)
			if !force {
				force, _ = req.Config["Force"].(bool)
			}
			op.Spec.Force = force

			skipSync, _ := req.Config["skipFinalSync"].(bool)
			if !skipSync {
				skipSync, _ = req.Config["SkipFinalSync"].(bool)
			}
			op.Spec.SkipFinalSync = skipSync
		}
		if req.Operation == "failover" {
			skipScaleDownSource, _ := req.Config["skipScaleDownSource"].(bool)
			if !skipScaleDownSource {
				skipScaleDownSource, _ = req.Config["SkipScaleDownSource"].(bool)
			}
			if skipScaleDownSource {
				if op.Annotations == nil {
					op.Annotations = make(map[string]string)
				}
				op.Annotations[skipScaleDownSourceAnnotation] = "true"
			}
			setSkipScaleDownSourceCompat(&op.Spec, skipScaleDownSource)
		}

		// Extract timeout if present (JSON numbers are float64 unmarshaled into interface{})
		if tm, ok := req.Config["timeoutMinutes"].(float64); ok {
			op.Spec.TimeoutMinutes = int32(tm)
		}
		skipPodReadyCheck, skipProvided := parseBoolConfig(req.Config, "skipPodReadyCheck", "SkipPodReadyCheck")
		waitUntilReady, waitProvided := parseBoolConfig(req.Config, "waitUntilReady", "WaitUntilReady")
		switch {
		case skipProvided:
			op.Spec.SkipPodReadyCheck = boolPtr(skipPodReadyCheck)
			op.Spec.WaitUntilReady = !skipPodReadyCheck
		case waitProvided:
			op.Spec.SkipPodReadyCheck = boolPtr(!waitUntilReady)
			op.Spec.WaitUntilReady = waitUntilReady
		}

		transport.SetTraceAnnotation(&op.ObjectMeta, ctx, metadata.AnnotationTraceID)
		createdOp, err := h.DisasterClient.DisasterV1().DisasterOperations(namespace).Create(c, &op, matev1.CreateOptions{})
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		transport.WriteSuccess(ctx, consts.StatusAccepted, utils.H{"operationID": createdOp.Name, "status": "Processing"}, nil)

	default:
		transport.WriteError(ctx, transport.CodeBadRequest, fmt.Sprintf("Unknown operation type: %s", req.Operation), nil)
	}
}

// setSkipScaleDownSourceCompat 兼容不同版本的 disaster-operator 依赖。
// 当 Spec 存在 SkipScaleDownSource 字段时通过反射写入；旧版本则安全跳过。
func setSkipScaleDownSourceCompat(spec interface{}, value bool) {
	v := reflect.ValueOf(spec)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return
	}
	elem := v.Elem()
	if !elem.IsValid() || elem.Kind() != reflect.Struct {
		return
	}
	f := elem.FieldByName("SkipScaleDownSource")
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
		f.SetBool(value)
	}
}

func parseBoolConfig(config map[string]interface{}, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := config[key].(bool); ok {
			return value, true
		}
	}
	return false, false
}

func boolPtr(v bool) *bool {
	out := v
	return &out
}
