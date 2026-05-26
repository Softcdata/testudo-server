package config

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *ConfigHandler) Register() {

	path := fmt.Sprintf("disasterconfigs.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/configs", c.configs)
	g.GET("/configs/names", c.configNames)
	g.GET("/configs/:name", c.config)
	g.POST("/configs", c.createConfig)
	g.DELETE("/configs/:name", c.deleteConfig)
	g.PUT("/configs/:name", c.updateConfig)

	// Watch 路由 - 使用 SSE 实时推送资源变化
	g.GET("/watch/configs", c.watchConfigs)      // 监听所有 Config 资源
	g.GET("/watch/configs/:name", c.watchConfig) // 监听指定的 Config 资源

}
