package backup

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *BackupHandler) Register() {

	path := fmt.Sprintf("disasterbackups.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/backups", c.backups)
	g.GET("/backups/:name", c.backup)
	g.POST("/backups", c.createBackup)
	g.DELETE("/backups/:name", c.deleteBackup)
	g.PUT("/backups/:name", c.updateBackup)

	// Watch 路由
	g.GET("/watch/backups", c.watchBackups)
	g.GET("/watch/backups/:name", c.watchBackup)

}
