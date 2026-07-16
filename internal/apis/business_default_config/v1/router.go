package businessdefaultconfig

func (h *Handler) Register() {
	g := h.Rg.Group("v1").Use(h.Mw...)
	g.GET("/business-default-config", h.getSnapshot)
	g.GET("/business-default-config/fields", h.listFields)
	g.GET("/business-default-config/frontend-fields", h.listFrontendSpecFields)
	g.PATCH("/business-default-config", h.patchConfig)
}
