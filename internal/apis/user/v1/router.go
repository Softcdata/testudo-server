package user

func (h *UserHandler) Register() {
	g := h.Rg.Group("v1").Use(h.Mw...)
	g.POST("/users", h.createUser)
	g.GET("/users", h.listUsers)
	g.DELETE("/users/:username", h.deleteUser)
	g.PATCH("/users/:username/password", h.patchUserPassword)
	g.PATCH("/users/:username/status", h.patchUserStatus)
}
