package transport

import (
	"github.com/cloudwego/hertz/pkg/app"
	matev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetTraceAnnotation writes current request trace_id into object annotations.
// It initializes the annotations map if nil and ignores empty trace ids.
func SetTraceAnnotation(meta *matev1.ObjectMeta, ctx *app.RequestContext, key string) {
	if meta == nil || ctx == nil {
		return
	}
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	if tid, ok := ctx.Get("trace_id"); ok {
		if s, ok2 := tid.(string); ok2 && s != "" {
			meta.Annotations[key] = s
		}
	}
}
