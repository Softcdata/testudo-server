package storage

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *StorageHandler) Register() {

	path := fmt.Sprintf("storage.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/storages", c.storages)
	g.GET("/storages/names", c.storageNames)
	g.GET("/storages/:name", c.storage)
	g.POST("/storages", c.createStorage)
	g.PUT("/storages/:name", c.updateStorage)
	g.PATCH("/storages/:name", c.patchStorage)
	g.DELETE("/storages/:name", c.deleteStorage)
	g.GET("/storages/:name/validate", c.validateStorage)

	g.POST("/storages/validate/connection", c.validateS3Connection)
	g.POST("/storages/connectivity/validate", c.validateBSLConnectivity)

	g.GET("/watch/storages", c.watchStorages)

}
