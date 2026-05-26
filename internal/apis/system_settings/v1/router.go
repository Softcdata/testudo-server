package systemsettings

func (h *SystemSettingsHandler) Register() {
	h.registerRoutes(true, true)
}

func (h *SystemSettingsHandler) RegisterWithoutPublic() {
	h.registerRoutes(true, false)
}

func (h *SystemSettingsHandler) RegisterPublicOnly() {
	h.registerRoutes(false, true)
}

func (h *SystemSettingsHandler) registerRoutes(includeAdminRoutes bool, includePublicRoute bool) {
	g := h.Rg.Group("v1").Use(h.Mw...)
	if includeAdminRoutes {
		g.GET("/system-settings", h.listSettings)
		g.POST("/system-settings", h.createSetting)
		g.PUT("/system-settings/:config_key", h.updateSetting)
		g.DELETE("/system-settings/:config_key", h.deleteSetting)
		g.POST("/system-settings/assets/:config_key", h.uploadAsset)
		g.GET("/system-settings/assets/:config_key", h.getAsset)
	}
	if includePublicRoute {
		g.GET("/system-settings/public", h.listPublicSettings)
	}
}
