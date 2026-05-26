package instance

import (
	"fmt"
	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (h *InstanceHandler) Register() {
	path := fmt.Sprintf("disasterinstances.%s", v1.GroupVersion.String())

	g := h.Rg.Group(path).Use(h.Mw...)

	// Basic CRUD
	g.GET("/instances", h.listInstances)
	g.POST("/instances", h.createInstance)
	g.GET("/instances/:name", h.getInstance)
	g.PUT("/instances/:name", h.updateInstance)
	g.DELETE("/instances/:name", h.deleteInstance)

	// Extended Info
	g.GET("/instances/:name/sync-status", h.getSyncStatus)
	g.GET("/instances/:name/sync-history", h.getSyncHistory)
	g.GET("/instances/:name/groups", h.getInstanceGroups)
	g.GET("/instances/:name/validate-target", h.validateTarget)
	g.POST("/instances/:name/restore-classes/validate", h.validateRestoreClasses)
	g.GET("/instances/:name/history", h.getHistory)
	g.GET("/instances/:name/operations/:operationName", h.getOperationDetail)
	g.POST("/instances/:name/actions", h.executeAction)

	// Watchers (WS/SSE)
	g.GET("/watch/instances", h.watchInstances)
	g.GET("/watch/instances/operations/:operationName", h.watchOperation)
	g.GET("/watch/instances/:name", h.watchInstance)
}
