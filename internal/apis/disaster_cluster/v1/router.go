package cluster

import (
	"fmt"

	v1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
)

func (c *ClusterHandler) Register() {

	path := fmt.Sprintf("cluster.%s", v1.GroupVersion.String())

	g := c.Rg.Group(path).Use(c.Mw...)

	g.GET("/clusters", c.clusters)
	g.GET("/clusters/names", c.clusterNames)
	g.GET("/clusters/:name", c.cluster)
	g.GET("/clusters/:name/protected-namespaces", c.protectedNamespaces)
	g.GET("/clusters/:name/restore-classes", c.listRestoreClasses)
	g.POST("/clusters", c.createCluster)
	g.POST("/clusters/:name/actions/refresh-namespaces", c.refreshNamespaces)
	g.PATCH("/clusters/:name", c.patchCluster)
	g.DELETE("/clusters/:name", c.deleteCluster)
	g.GET("/clusters/:name/validate", c.validateCluster)
	g.POST("/clusters/kubeconfig/validate", c.validateKubeConfig)

	// Watch 路由 - 使用 SSE 实时推送资源变化
	g.GET("/watch/clusters", c.watchClusters)      // 监听所有 Cluster 资源
	g.GET("/watch/clusters/:name", c.watchCluster) // 监听指定的 Cluster 资源

}
