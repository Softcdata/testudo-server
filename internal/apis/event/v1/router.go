package event

func (h *EventHandler) Register() {
	path := "v1"
	g := h.Rg.Group(path).Use(h.Mw...)

	// 全局事件流
	// GET /apis/v1/events/watch
	g.GET("/watch/events", h.watchEvents)

	// 全局历史记录查询
	// GET /apis/v1/events
	g.GET("/events", h.listEvents)

	// 指定资源事件流
	// GET /apis/v1/:resource/:name/events/watch
	g.GET("/watch/:resource/:name/events", h.watchResourceEvents)

	// 指定资源历史记录查询
	// GET /apis/v1/:resource/:name/history
	g.GET("/:resource/:name/history", h.listResourceEvents)
}
