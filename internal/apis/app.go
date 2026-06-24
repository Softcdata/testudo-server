package apis

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	hc "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/router"
	"github.com/softcdata/testudo-server/internal/userstore"
)

type ApiServer struct {
	ServerOption     []hc.Option
	GlobalMiddleware []app.HandlerFunc
	Kc               kube.KubeClient
}

func (a *ApiServer) Run() error {
	s := server.New(a.ServerOption...)
	// Hertz's websocket upgrader uses hijacked connections. Keep the hijacked
	// connection owned by the websocket handler and avoid reusing its wrapper
	// while background watch goroutines are shutting down.
	s.KeepHijackedConns = true
	s.NoHijackConnPool = true

	s.Use(a.GlobalMiddleware...)

	store := userstore.NewKubeStore(a.Kc.K8sClient, common.DisasterSystemNamespace)
	initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := store.EnsureInitialized(initCtx); err != nil {
		return fmt.Errorf("initialize user secret: %w", err)
	}

	router.NewRouter(s, a.Kc)

	s.OnRun = append(s.OnRun, func(ctx context.Context) error {
		go func() {
			// Start InformerFactory
			a.Kc.InformerFactory.Start(ctx.Done())
			a.Kc.InformerFactory.WaitForCacheSync(ctx.Done())

			err := a.Kc.ClusterClient.Start(ctx)
			if err != nil {
				hlog.SystemLogger().Errorf("start cluster client err: %v", err)
			}
			a.Kc.ClusterClient.GetCache().WaitForCacheSync(ctx)
		}()
		return nil
	})

	s.Spin()
	return nil
}
