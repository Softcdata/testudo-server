package jobs

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *JobsHandler) Register() {

	path := fmt.Sprintf("disasterjobs.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/jobs", c.configs)
	g.GET("/jobs/:name", c.config)
	g.POST("/jobs", c.createConfig)
	g.DELETE("/jobs/:name", c.deleteConfig)

	// Watch 路由
	g.GET("/watch/jobs", c.watchJobs)
	g.GET("/watch/jobs/:name", c.watchJob)

}
