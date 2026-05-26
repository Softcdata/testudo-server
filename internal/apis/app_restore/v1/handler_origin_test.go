package apprestore

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	"github.com/softcdata/testudo-operator/pkg/clientset/versioned/fake"
	informers "github.com/softcdata/testudo-operator/pkg/informers/externalversions"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func newMockRestoreHandler(objects ...runtime.Object) *AppRestoreHandler {
	fakeClient := fake.NewSimpleClientset(objects...)
	informerFactory := informers.NewSharedInformerFactory(fakeClient, 0)

	_ = informerFactory.Disaster().V1().AppRestores().Informer()

	stopCh := make(chan struct{})
	informerFactory.Start(stopCh)
	informerFactory.WaitForCacheSync(stopCh)

	kc := &kube.KubeClient{
		DisasterClient:  fakeClient,
		InformerFactory: informerFactory,
	}

	h := server.Default()
	rg := h.Group("/v1")

	return NewAppRestoreHandler(kc, rg)
}

func TestAppRestores_ListOriginFilter(t *testing.T) {
	controllerTrue := true
	userRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-restore",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	instanceRestore := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-restore",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				appResourceOriginLabelKey: appResourceOriginDisasterInstance,
			},
		},
	}
	instanceRestoreNoLabel := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance-restore-no-label",
			Namespace: common.DisasterSystemNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       "DataSync",
					Name:       "ds-1",
					Controller: &controllerTrue,
				},
			},
		},
	}
	drillRestoreWrongLabel := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "drr-drill-demo-123456",
			Namespace: common.DisasterSystemNamespace,
			Labels: map[string]string{
				appResourceOriginLabelKey:        appResourceOriginUser,
				"testudo.softcdata.com/drill":        "drill-demo",
				"testudo.softcdata.com/restore-type": "resource",
			},
		},
	}
	drillRestoreByPrefix := &dapisv1.AppRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ddr-drill-demo-654321",
			Namespace: common.DisasterSystemNamespace,
		},
	}
	h := newMockRestoreHandler(
		userRestore,
		instanceRestore,
		instanceRestoreNoLabel,
		drillRestoreWrongLabel,
		drillRestoreByPrefix,
	)

	type ListResponse struct {
		Code int `json:"code"`
		Data struct {
			Items []AppRestoreDTO `json:"items"`
		} `json:"data"`
	}

	testCases := []struct {
		name      string
		uri       string
		wantNames []string
	}{
		{
			name:      "default user filter",
			uri:       "/apprestores",
			wantNames: []string{"user-restore"},
		},
		{
			name:      "origin all",
			uri:       "/apprestores?origin=all",
			wantNames: []string{"user-restore", "instance-restore", "instance-restore-no-label", "drr-drill-demo-123456", "ddr-drill-demo-654321"},
		},
		{
			name:      "origin instance",
			uri:       "/apprestores?origin=instance",
			wantNames: []string{"instance-restore", "instance-restore-no-label", "drr-drill-demo-123456", "ddr-drill-demo-654321"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := app.NewContext(16)
			ctx.Request.SetRequestURI(tc.uri)

			h.appRestores(context.Background(), ctx)
			assert.Equal(t, consts.StatusOK, ctx.Response.StatusCode())

			var resp ListResponse
			err := json.Unmarshal(ctx.Response.Body(), &resp)
			assert.NoError(t, err)

			gotNames := make([]string, 0, len(resp.Data.Items))
			for _, item := range resp.Data.Items {
				gotNames = append(gotNames, item.Name)
			}
			assert.ElementsMatch(t, tc.wantNames, gotNames)
		})
	}
}
