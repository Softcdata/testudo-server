package deletioncheck

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func newMockHandler(objects ...runtime.Object) *DeletionCheckHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	kc := &kube.KubeClient{
		DisasterClient: fakeClient,
	}

	h := server.Default()
	rg := h.Group("/apis")
	return NewDeletionCheckHandler(kc, rg)
}

func TestDeletionCheck_NotFound(t *testing.T) {
	h := newMockHandler()

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{
		ResourceKind: "StorageRepository",
		Name:         "not-exist",
		Namespace:    common.DisasterSystemNamespace,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusNotFound, ctx.Response.StatusCode())
}

func TestDeletionCheck_UpstreamPresent_CanDeleteFalse(t *testing.T) {
	ns := common.DisasterSystemNamespace
	targetToken := "fake-token"

	target := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-1",
			Namespace: ns,
			UID:       types.UID("repo-uid-1"),
			Labels: map[string]string{
				labelDependencyToken: targetToken,
			},
		},
	}

	upstream := &dapisv1.DisasterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-1",
			Namespace: ns,
			UID:       types.UID("policy-uid-1"),
			Labels: map[string]string{
				labelDependencyToken:              "1111111111111111",
				dependencyToLabelKey(targetToken): "label.storageRepositoryName",
			},
		},
	}

	h := newMockHandler(target, upstream)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{
		ResourceKind: "StorageRepository",
		Name:         "repo-1",
		Namespace:    ns,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                   `json:"code"`
		Data DeletionCheckResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.False(t, resp.Data.CanDelete)
	assert.Len(t, resp.Data.Upstream, 1)
	assert.Equal(t, "DisasterPolicy", resp.Data.Upstream[0].Kind)
	assert.Equal(t, "policy-1", resp.Data.Upstream[0].Name)
	assert.Equal(t, ns, resp.Data.Upstream[0].Namespace)
	assert.Equal(t, "label.storageRepositoryName", resp.Data.Upstream[0].RelationCode)
}

func TestDeletionCheck_TargetMissingTokenLabel_FallbackStillFindsUpstream(t *testing.T) {
	ns := common.DisasterSystemNamespace
	targetUID := types.UID("repo-uid-missing-token")
	targetToken := buildDependencyToken(string(targetUID))

	target := &dapisv1.StorageRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "repo-2",
			Namespace: ns,
			UID:       targetUID,
			Labels:    map[string]string{}, // dependency-token missing (backfill window)
		},
	}
	upstream := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "config-1",
			UID:  types.UID("config-uid-1"),
			Labels: map[string]string{
				dependencyToLabelKey(targetToken): "spec.storageRepository",
			},
		},
	}

	h := newMockHandler(target, upstream)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{
		ResourceKind: "StorageRepository",
		Name:         "repo-2",
		Namespace:    ns,
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                   `json:"code"`
		Data DeletionCheckResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.False(t, resp.Data.CanDelete)
	assert.Len(t, resp.Data.Upstream, 1)
	assert.Equal(t, "DisasterConfig", resp.Data.Upstream[0].Kind)
	assert.Equal(t, "config-1", resp.Data.Upstream[0].Name)
}

func TestDeletionCheck_DownstreamResolveByToken(t *testing.T) {
	clusterToken := "aaaaaaaaaaaaaaaa"
	configToken := "bbbbbbbbbbbbbbbb"

	cluster := &dapisv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-1",
			UID:  types.UID("cluster-uid-1"),
			Labels: map[string]string{
				labelDependencyToken: clusterToken,
			},
		},
	}

	config := &dapisv1.DisasterConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dc-1",
			UID:  types.UID("dc-uid-1"),
			Labels: map[string]string{
				labelDependencyToken:               configToken,
				dependencyToLabelKey(clusterToken): "spec.sourceCluster",
			},
		},
	}

	h := newMockHandler(cluster, config)

	ctx := app.NewContext(16)
	req := DeletionCheckRequest{
		ResourceKind: "DisasterConfig",
		Name:         "dc-1",
	}
	body, _ := json.Marshal(req)
	ctx.Request.SetBody(body)
	ctx.Request.Header.SetContentTypeBytes([]byte("application/json"))

	h.check(context.Background(), ctx)
	assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

	var resp struct {
		Code int                   `json:"code"`
		Data DeletionCheckResponse `json:"data"`
	}
	err := json.Unmarshal(ctx.Response.Body(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
	assert.True(t, resp.Data.CanDelete) // no upstream
	assert.Len(t, resp.Data.Downstream, 1)
	assert.False(t, resp.Data.Downstream[0].Unresolved)
	assert.Equal(t, "Cluster", resp.Data.Downstream[0].Kind)
	assert.Equal(t, "cluster-1", resp.Data.Downstream[0].Name)
	assert.Equal(t, "spec.sourceCluster", resp.Data.Downstream[0].RelationCode)
}
