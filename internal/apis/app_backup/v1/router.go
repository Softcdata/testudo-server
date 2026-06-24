package appbackup

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *AppBackupHandler) Register() {
	path := fmt.Sprintf("appbackups.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/appbackups", c.appBackups)
	g.GET("/appbackups/clusters", c.getAppBackupClusters)
	g.GET("/appbackups/:name", c.appBackup)
	g.GET("/appbackups/:name/history", c.getBackupHistory)
	g.POST("/appbackups", c.createAppBackup)
	g.DELETE("/appbackups/:name", c.deleteAppBackup)
	g.PUT("/appbackups/:name", c.updateAppBackup)
	g.POST("/appbackups/:name/actions/:type", c.executeAction)
	g.GET("/appbackups/:name/backups/:backupName/download", c.downloadBackup)
	g.GET("/velero/backups/:backupName/includes", c.getVeleroBackupIncludes)

	// Watch 路由
	g.GET("/watch/appbackups", c.watchAppBackups)
	g.GET("/watch/appbackups/:name", c.watchAppBackup)
}

func (c *AppBackupHandler) RegisterDownloadStream() {
	path := fmt.Sprintf("appbackups.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)
	g.GET("/appbackups/:name/backups/:backupName/download/stream", c.downloadBackupStream)
}
