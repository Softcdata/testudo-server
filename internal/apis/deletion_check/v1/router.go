package deletioncheck

func (h *DeletionCheckHandler) Register() {
	g := h.Rg.Group("v1").Use(h.Mw...)
	g.POST("/deletion/check", h.check)
}
