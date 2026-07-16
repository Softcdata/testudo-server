package router

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/softcdata/testudo-server/configs"
	appbackupv1 "github.com/softcdata/testudo-server/internal/apis/app_backup/v1"
	apprestorev1 "github.com/softcdata/testudo-server/internal/apis/app_restore/v1"
	businessdefaultconfigv1 "github.com/softcdata/testudo-server/internal/apis/business_default_config/v1"
	deletioncheckv1 "github.com/softcdata/testudo-server/internal/apis/deletion_check/v1"
	backupv1 "github.com/softcdata/testudo-server/internal/apis/disaster_backup/v1"
	clusterv1 "github.com/softcdata/testudo-server/internal/apis/disaster_cluster/v1"
	configv1 "github.com/softcdata/testudo-server/internal/apis/disaster_config/v1"
	drillv1 "github.com/softcdata/testudo-server/internal/apis/disaster_drill/v1"
	groupv1 "github.com/softcdata/testudo-server/internal/apis/disaster_group/v1"
	instancev1 "github.com/softcdata/testudo-server/internal/apis/disaster_instance/v1"
	jobsv1 "github.com/softcdata/testudo-server/internal/apis/disaster_jobs/v1"
	policyv1 "github.com/softcdata/testudo-server/internal/apis/disaster_policy/v1"
	storagev1 "github.com/softcdata/testudo-server/internal/apis/disaster_storage/v1"
	eventv1 "github.com/softcdata/testudo-server/internal/apis/event/v1"
	kubernetesresources "github.com/softcdata/testudo-server/internal/apis/kubernetes_resources"
	platformlicensev1 "github.com/softcdata/testudo-server/internal/apis/platform_license/v1"
	statisticsv1 "github.com/softcdata/testudo-server/internal/apis/statistics/v1"
	systemsettingsv1 "github.com/softcdata/testudo-server/internal/apis/system_settings/v1"
	userv1 "github.com/softcdata/testudo-server/internal/apis/user/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	"github.com/softcdata/testudo-server/internal/middleware"
	"github.com/softcdata/testudo-server/internal/openapi"
	"github.com/softcdata/testudo-server/internal/storage"
	"github.com/softcdata/testudo-server/internal/userstore"
)

type RegisterInterface interface {
	Register()
}

func NewRouter(sh *server.Hertz, kc kube.KubeClient) {

	store := userstore.NewKubeStore(kc.K8sClient, common.DisasterSystemNamespace)
	jwtMiddleware := middleware.NewJWT(store)

	// Health endpoints
	sh.GET("/healthz", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})
	sh.GET("/readyz", func(ctx context.Context, c *app.RequestContext) {
		c.String(200, "ok")
	})

	sh.POST("/login", jwtMiddleware.LoginHandler)
	sh.POST("/refresh_token", middleware.RefreshTokenHandler)

	if configs.Cfg.Swagger.Enabled {
		openapi.Register(sh)
	}

	bashPath := sh.Group("/apis")
	bashPath.Use(middleware.WebSocketTokenAdapter())
	if configs.Cfg.Server.Env != "dev" {
		bashPath.Use(jwtMiddleware.MiddlewareFunc())
	}
	// Apply JWT and Trace middleware globally
	bashPath.Use(recovery.Recovery())
	bashPath.Use(middleware.TraceMiddleware())
	// bashPath.Use(middleware.TenantID())

	apiPath := sh.Group("/api")
	apiPath.Use(middleware.WebSocketTokenAdapter())
	if configs.Cfg.Server.Env != "dev" {
		apiPath.Use(jwtMiddleware.MiddlewareFunc())
	}
	apiPath.Use(recovery.Recovery())
	apiPath.Use(middleware.TraceMiddleware())

	apiPublicPath := sh.Group("/api")
	apiPublicPath.Use(middleware.WebSocketTokenAdapter())
	apiPublicPath.Use(recovery.Recovery())
	apiPublicPath.Use(middleware.TraceMiddleware())

	apisPublicPath := sh.Group("/apis")
	apisPublicPath.Use(recovery.Recovery())
	apisPublicPath.Use(middleware.TraceMiddleware())

	minioStorage := storage.NewMinIOStorage()

	handler := []RegisterInterface{
		clusterv1.NewClusterHandler(&kc, bashPath),
		backupv1.NewBackupHandler(&kc, bashPath),
		appbackupv1.NewAppBackupHandler(&kc, bashPath, minioStorage),
		apprestorev1.NewAppRestoreHandler(&kc, bashPath),
		configv1.NewConfigHandler(&kc, bashPath),
		policyv1.NewPolicyHandler(&kc, bashPath),
		instancev1.NewInstanceHandler(&kc, bashPath),
		groupv1.NewGroupHandler(&kc, bashPath),
		drillv1.NewDrillHandler(&kc, bashPath),
		jobsv1.NewJobsHandler(&kc, bashPath),
		storagev1.NewStorageHandler(&kc, bashPath),
		eventv1.NewEventHandler(&kc, bashPath),
		kubernetesresources.NewKubernetesResourcesHandler(&kc, bashPath),
		platformlicensev1.NewHandler(&kc, bashPath, configs.Cfg.License.Namespace, configs.Cfg.License.CAPath),
		statisticsv1.NewStatisticsHandler(&kc, bashPath),
		deletioncheckv1.NewDeletionCheckHandler(&kc, bashPath),
		systemsettingsv1.NewSystemSettingsHandler(&kc, bashPath),
		businessdefaultconfigv1.NewHandler(&kc, bashPath),
		userv1.NewUserHandler(store, bashPath),
	}

	for _, h := range handler {
		h.Register()
	}

	appbackupv1.NewAppBackupHandler(&kc, apisPublicPath, minioStorage).RegisterDownloadStream()

	// Alias path for clients that use /api prefix instead of /apis.
	deletioncheckv1.NewDeletionCheckHandler(&kc, apiPath).Register()
	systemsettingsv1.NewSystemSettingsHandler(&kc, apiPath).RegisterWithoutPublic()
	businessdefaultconfigv1.NewHandler(&kc, apiPath).Register()
	systemsettingsv1.NewSystemSettingsHandler(&kc, apiPublicPath).RegisterPublicOnly()
	userv1.NewUserHandler(store, apiPath).Register()

}
