package app

import (
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	hc "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzzap "github.com/hertz-contrib/logger/zap"
	"github.com/softcdata/testudo-server/configs"
	"github.com/softcdata/testudo-server/internal/apis"
	"github.com/softcdata/testudo-server/internal/kube"
	mw "github.com/softcdata/testudo-server/internal/middleware"
	"github.com/spf13/cobra"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type ServerOptions struct {
}

func NewServerConfig() *ServerOptions {
	return &ServerOptions{}
}

func NewServerCommand() *cobra.Command {
	c := NewServerConfig()

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := configs.LoadConfig()
			if err != nil {
				return fmt.Errorf("load config err: %v", err)
			}
			if configs.Validate() != nil {
				return fmt.Errorf("validate config: %v", err)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.newApiServer()
		},
	}

	return cmd
}

func (c *ServerOptions) newApiServer() error {

	logger := hertzzap.NewLogger(hertzzap.WithZapOptions())

	logger.SetLevel(configs.GetLogLevel())
	hlog.SetLogger(logger)

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	serverOption := []hc.Option{
		server.WithHostPorts(fmt.Sprintf("%s:%d", configs.Cfg.Server.Host, configs.Cfg.Server.Port)),
		server.WithExitWaitTime(10 * time.Minute),
	}

	mwFc := []app.HandlerFunc{
		mw.LocaleMiddleware(),
		mw.TraceMiddleware(),
		mw.RequestID(),
		mw.AccessLog(),
		mw.Recovery(),
	}

	kc, err := kube.NewClient()
	if err != nil {
		return err
	}

	app := apis.ApiServer{
		ServerOption:     serverOption,
		GlobalMiddleware: mwFc,
		Kc:               kc,
	}

	return app.Run()

}
