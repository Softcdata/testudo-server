package transport

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"k8s.io/apimachinery/pkg/labels"
)

// Options 定义标准的查询参数，用于 Hertz Bind（原 BaseQuery）
type Options struct {
	Limit   int               `query:"limit"`
	Page    int               `query:"page"`
	Sort    string            `query:"sort"`
	Order   string            `query:"order"`
	Keyword string            `query:"keyword"`
	Filters map[string]string `query:"-"`
}

type PaginationMeta struct {
	Limit    int    `json:"limit"`
	Total    int64  `json:"total,omitempty"`
	Partial  bool   `json:"partial"`
	First    string `json:"first,omitempty"`
	Previous string `json:"previous,omitempty"`
	Next     string `json:"next,omitempty"`
	Last     string `json:"last,omitempty"`
}

type SortMeta struct {
	Name    string `json:"name,omitempty"`
	Order   string `json:"order,omitempty"`
	Reverse string `json:"reverse,omitempty"`
}

type CollectionMeta struct {
	Type         string                 `json:"type"`
	ResourceType string                 `json:"resourceType"`
	Links        map[string]string      `json:"links,omitempty"`
	Pagination   *PaginationMeta        `json:"pagination,omitempty"`
	Sort         *SortMeta              `json:"sort,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	Summary      map[string]int         `json:"summary,omitempty"`
}

type CollectionData[T any] struct {
	Items []T `json:"items"`
}

// ParseOptions 解析 HTTP 请求参数
func ParseOptions(c context.Context, ctx *app.RequestContext) *Options {
	var q Options
	_ = ctx.BindAndValidate(&q)
	// 默认为 10，但允许用户传 -1 来获取全部数据
	if q.Limit == 0 {
		q.Limit = 10
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	q.Filters = make(map[string]string)
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		v := string(value)
		switch k {
		case "limit", "page", "sort", "order", "keyword":
		default:
			q.Filters[k] = v
		}
	})
	return &q
}

// BuildCollectionResponse 构造标准集合响应
func BuildCollectionResponse[T any](
	baseURL string,
	resourceType string,
	data []T,
	query *Options,
	total int64,
	extraLinks map[string]string,
	itemLinkGenerator func(T) map[string]string,
) (*CollectionData[T], *CollectionMeta) {
	links := map[string]string{
		"self": buildSelfLink(baseURL, query),
	}
	for k, v := range extraLinks {
		links[k] = v
	}
	if itemLinkGenerator != nil {
		for _, item := range data {
			itemLinks := itemLinkGenerator(item)
			for k, v := range itemLinks {
				links[k] = v
			}
		}
	}

	partial := false
	if query.Limit > 0 {
		partial = total > int64(query.Limit)
	}

	meta := &CollectionMeta{
		Type:         "collection",
		ResourceType: resourceType,
		Links:        links,
		Pagination: &PaginationMeta{
			Limit:    query.Limit,
			Total:    total,
			Partial:  partial,
			First:    buildPageLink(baseURL, query, 1),
			Previous: buildPreviousLink(baseURL, query),
			Next:     buildNextLink(baseURL, query, total),
			Last:     buildLastLink(baseURL, query, total),
		},
		Sort: &SortMeta{
			Name:  query.Sort,
			Order: query.Order,
		},
		Filters: make(map[string]interface{}),
	}

	for k, v := range query.Filters {
		meta.Filters[k] = v
	}

	return &CollectionData[T]{Items: data}, meta
}

func buildPreviousLink(baseURL string, q *Options) string {
	if q.Page > 1 {
		return buildPageLink(baseURL, q, q.Page-1)
	}
	return ""
}

func buildNextLink(baseURL string, q *Options, total int64) string {
	if q.Limit <= 0 {
		return ""
	}
	lastPage := int((total + int64(q.Limit) - 1) / int64(q.Limit))
	if lastPage < 1 {
		lastPage = 1
	}
	if q.Page < lastPage {
		return buildPageLink(baseURL, q, q.Page+1)
	}
	return ""
}

func buildLastLink(baseURL string, q *Options, total int64) string {
	if q.Limit <= 0 {
		return buildPageLink(baseURL, q, 1)
	}
	lastPage := int((total + int64(q.Limit) - 1) / int64(q.Limit))
	if lastPage < 1 {
		lastPage = 1
	}
	return buildPageLink(baseURL, q, lastPage)
}

func buildPageLink(baseURL string, q *Options, page int) string {
	u, _ := url.Parse(baseURL)
	values := u.Query()
	values.Set("limit", fmt.Sprintf("%d", q.Limit))
	values.Set("page", fmt.Sprintf("%d", page))
	if q.Sort != "" {
		values.Set("sort", q.Sort)
		values.Set("order", q.Order)
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func buildSelfLink(baseURL string, q *Options) string {
	u, _ := url.Parse(baseURL)
	values := u.Query()
	values.Set("limit", fmt.Sprintf("%d", q.Limit))
	values.Set("page", fmt.Sprintf("%d", q.Page))
	if q.Sort != "" {
		values.Set("sort", q.Sort)
		values.Set("order", q.Order)
	}
	u.RawQuery = values.Encode()
	return u.String()
}

// Paginate 内存分页辅助函数
func Paginate[T any](items []T, q *Options) ([]T, int64) {
	total := int64(len(items))
	if q.Limit < 0 {
		return items, total
	}
	start := (q.Page - 1) * q.Limit
	if start < 0 {
		start = 0
	}
	if int64(start) >= total {
		return []T{}, total
	}
	end := start + q.Limit
	if int64(end) > total {
		end = int(total)
	}
	return items[start:end], total
}

// BuildLabelSelector 返回一个空的 Label Selector。
// 根据 API 标准 "Filtering: 一律模糊匹配"，我们不再尝试将过滤参数转换为 K8s 的精确匹配 Selector。
// 所有的过滤逻辑都统一交由各 Handler 在内存中通过 MatchFuzzy 函数执行。
// 这样做是为了避免合法的 Label Value 被 K8s Lister 进行精确匹配，从而导致模糊搜索失效。
// 例如：用户搜索 name="backup" 时，应该能搜到 "my-backup-01"，而不是仅搜到 "backup"。
func BuildLabelSelector(q *Options) labels.Selector {
	return labels.NewSelector()
}

// MatchFuzzy 执行包含匹配逻辑，实现“一律模糊搜索”
func MatchFuzzy(actualValue, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	// 统一转为包含匹配，不要求传 *
	cleanPattern := strings.Trim(pattern, "*")
	return strings.Contains(actualValue, cleanPattern)
}

// Sort 内存排序辅助函数
func Sort[T any](items []T, q *Options, compareFunc func(a, b T, field string) int) []T {
	if q.Sort == "" || compareFunc == nil {
		return items
	}
	sortedItems := make([]T, len(items))
	copy(sortedItems, items)
	sort.SliceStable(sortedItems, func(i, j int) bool {
		result := compareFunc(sortedItems[i], sortedItems[j], q.Sort)
		if q.Order == "desc" {
			return result > 0
		}
		return result < 0
	})
	return sortedItems
}
