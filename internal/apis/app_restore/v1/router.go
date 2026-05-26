package apprestore

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *AppRestoreHandler) Register() {
	path := fmt.Sprintf("apprestores.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/apprestores", c.appRestores)
	g.GET("/apprestores/:name", c.appRestore)
	g.POST("/apprestores/preflight/validate", c.validateRestorePreflight)
	g.POST("/apprestores", c.createAppRestore)
	g.DELETE("/apprestores/:name", c.deleteAppRestore)
	g.PUT("/apprestores/:name", c.updateAppRestore)
	g.POST("/apprestores/:name/actions/:type", c.executeAction)

	// Watch 路由
	g.GET("/watch/apprestores", c.watchAppRestores)
	g.GET("/watch/apprestores/:name", c.watchAppRestore)
}
