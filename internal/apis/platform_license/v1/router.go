package platformlicenseapi

func (h *Handler) Register() {
	g := h.Rg.Group("v1").Use(h.Mw...)
	g.GET("/platform-license/status", h.status)
	g.POST("/platform-license/install", h.install)
}
