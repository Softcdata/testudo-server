package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCollectionResponse(t *testing.T) {
	baseURL := "/v1/clusters"
	resourceType := "disasterCluster"
	data := []string{"cluster1", "cluster2"}
	query := &Options{
		Limit: 10,
		Page:  1,
		Sort:  "name",
		Order: "asc",
	}

	dataResp, meta := BuildCollectionResponse(baseURL, resourceType, data, query, 100, map[string]string{
		"schemas": "/v1/schemas",
	}, nil)

	assert.Equal(t, "collection", meta.Type)
	assert.Equal(t, resourceType, meta.ResourceType)
	assert.Equal(t, data, dataResp.Items)
	assert.Equal(t, 10, meta.Pagination.Limit)
	assert.Equal(t, int64(100), meta.Pagination.Total)
	assert.True(t, meta.Pagination.Partial)
	assert.NotEmpty(t, meta.Pagination.First)
	assert.Contains(t, meta.Pagination.Next, "page=2")
	assert.Contains(t, meta.Pagination.Next, "limit=10")
	assert.Contains(t, meta.Pagination.Next, "sort=name")
	assert.Contains(t, meta.Pagination.Next, "order=asc")
	assert.Contains(t, meta.Links["self"], "limit=10")
	assert.Equal(t, "/v1/schemas", meta.Links["schemas"])
}

func TestBuildCollectionResponse_NoNextPage(t *testing.T) {
	baseURL := "/v1/clusters"
	resourceType := "disasterCluster"
	data := []string{"cluster1"}
	query := &Options{
		Limit: 10,
		Page:  1,
	}

	_, meta := BuildCollectionResponse(baseURL, resourceType, data, query, 1, nil, nil)

	assert.False(t, meta.Pagination.Partial)
	assert.Empty(t, meta.Pagination.Next)
}

func TestBuildCollectionResponse_WithItemLinks(t *testing.T) {
	baseURL := "/v1/clusters"
	resourceType := "disasterCluster"
	data := []struct {
		Name string `json:"name"`
	}{
		{Name: "cluster1"},
	}
	query := &Options{
		Limit: 10,
		Page:  1,
	}

	dataResp, meta := BuildCollectionResponse(baseURL, resourceType, data, query, 1, nil, func(item struct {
		Name string `json:"name"`
	}) map[string]string {
		return map[string]string{
			item.Name: "/v1/clusters/" + item.Name,
		}
	})

	assert.Len(t, dataResp.Items, 1)
	assert.Equal(t, "cluster1", dataResp.Items[0].Name)

	// 验证链接是否合并到了 Collection Links 中
	assert.Equal(t, "/v1/clusters/cluster1", meta.Links["cluster1"])
}
