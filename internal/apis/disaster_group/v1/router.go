package v1

import (
	"fmt"

	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (h *GroupHandler) Register() {
	path := fmt.Sprintf("disastergroups.%s", dapisv1.GroupVersion.String())
	g := h.Rg.Group(path).Use(h.Mw...)

	// Basic CRUD
	g.GET("/groups", h.listGroups)
	g.POST("/groups", h.createGroup)
	g.GET("/groups/:name", h.getGroup)
	g.PUT("/groups/:name", h.updateGroup)   // Full update
	g.PATCH("/groups/:name", h.updateGroup) // Partial update
	g.DELETE("/groups/:name", h.deleteGroup)

	// Actions
	g.POST("/groups/:name/actions", h.executeAction)

	// History
	g.GET("/groups/:name/history", h.getHistory)
	g.GET("/groups/:name/operations/:operationName", h.getOperationDetail)

	// Instance Picker（专供前端"选择容灾实例"UI 使用）
	g.GET("/groups/instance-picker", h.instancePicker)

	// Group Members（容灾组已选实例列表）
	g.GET("/groups/:name/instances", h.listGroupInstances)

	// Watch 事件流（WebSocket）
	g.GET("/watch/groups/status", h.watchGroupStatuses)
	g.GET("/watch/groups/status/:name", h.watchGroupStatus)
	g.GET("/watch/groups/operations", h.watchGroupOperations)
	g.GET("/watch/groups/operations/:operationName", h.watchGroupOperation)
}
