package drill

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

// Register 注册路由
func (h *DrillHandler) Register() {
	// 使用 CRD GroupVersion 动态生成路径前缀
	// 例如: disasterdrills.testudo.softcdata.com/v1
	path := fmt.Sprintf("disasterdrills.%s", v1.GroupVersion.String())

	group := h.Rg.Group(path)
	group.Use(h.Mw...)

	// Basic CRUD
	group.GET("/drills", h.listDrills)
	group.GET("/drills/actions/protected-namespaces", h.getProtectedNamespaces)
	group.POST("/drills", h.createDrill)
	group.GET("/drills/:name", h.getDrill)
	group.DELETE("/drills/:name", h.deleteDrill)

	// Actions
	group.POST("/drills/:name/confirm", h.confirmDrill)
	group.POST("/drills/:name/restart", h.restartDrill)
	group.POST("/drills/:name/cleanup", h.cleanupDrill)

	// Watchers (WS)
	group.GET("/watch/drills", h.watchDrills)
	group.GET("/watch/drills/:name", h.watchDrill)
}
