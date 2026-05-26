# Capability: 路由实现标准

## Description
定义 `disaster-server` 中 API 路由和处理程序（Handler）的标准实现模式，以确保代码的一致性和可维护性。

### Requirement: Handler 结构定义
所有 API 资源的处理程序必须定义为一个结构体，该结构体必须包含 Kubernetes 客户端、路由组和中间件列表。

#### Scenario: 定义标准 Handler 结构
- **WHEN** 开发一个新的 API 资源（如 `Backup`）
- **THEN** 必须定义一个包含 `*kube.KubeClient`、`*route.RouterGroup` 和 `[]app.HandlerFunc` 的结构体
- **AND** 必须提供一个标准的构造函数 `New<Resource>Handler`

```go
type BackupHandler struct {
	*kube.KubeClient
	Rg *route.RouterGroup
	Mw []app.HandlerFunc
}

func NewBackupHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *BackupHandler {
	return &BackupHandler{
		KubeClient: kc,
		Rg:         rg,
		Mw:         mw,
	}
}
```

### Requirement: 路由注册接口
每个处理程序必须实现 `Register` 方法，用于将具体的路由路径绑定到处理函数。

#### Scenario: 实现 Register 方法
- **WHEN** 注册资源的路由时
- **THEN** 必须使用 `Register` 方法
- **AND** 路由路径应包含 API 组和版本信息（如 `disasterbackups.testudo.softcdata.com/v1`）
- **AND** 必须支持标准的 CRUD 操作（GET, POST, PUT, DELETE）
- **AND** 必须支持 Watch 操作（如果适用）

```go
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
```

### Requirement: 处理函数实现
处理函数（Handler Functions）必须遵循 Hertz 框架的签名，并统一处理响应和错误。

#### Scenario: 实现列表查询 (List)
- **WHEN** 处理获取资源列表的请求时
- **THEN** 必须使用 `internal/transport` 包提供的工具函数
- **AND** 必须支持分页、排序、过滤和 HATEOAS 链接
- **AND** 必须使用 Informer Lister 获取数据以支持高效的内存分页和总数统计
- **AND** 必须将 CRD 对象转换为 DTO 对象，隐藏内部实现细节
- **AND** 必须使用 `transport.WriteSuccess` 返回标准响应

```go
func (h *BackupHandler) backups(c context.Context, ctx *app.RequestContext) {
	// 1. 解析通用查询参数
	qParams := transport.ParseOptions(c, ctx)

	// 2. 构建 Label Selector
	selector := transport.BuildLabelSelector(qParams)

	// 3. 调用 Lister 获取过滤后的数据 (从本地缓存)
	filteredItems, err := h.BackupLister.List(selector)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	// 4. 内存排序逻辑
	sortedItems := transport.Sort(filteredItems, qParams, func(a, b *dapisv1.DisasterBackup, field string) int {
		switch field {
		case "name":
			return strings.Compare(a.Name, b.Name)
		case "creationTimestamp":
			if a.CreationTimestamp.Before(&b.CreationTimestamp) {
				return -1
			}
			if a.CreationTimestamp.After(b.CreationTimestamp.Time) {
				return 1
			}
			return 0
		default:
			return 0
		}
	})

	// 5. 内存分页逻辑
	pagedItems, total := transport.Paginate(sortedItems, qParams)

	// 6. 转换为 DTO
	dtos := make([]DisasterBackupDTO, len(pagedItems))
	for i, item := range pagedItems {
		dtos[i] = ConvertToDisasterBackupDTO(item)
	}

	// 7. 构建标准响应
	requestPath := string(ctx.URI().Path())
	data, meta := transport.BuildCollectionResponse(
		requestPath,
		"disasterBackup",
		dtos,
		qParams,
		total,
		nil,
		func(item DisasterBackupDTO) map[string]string {
			return map[string]string{
				item.Name: fmt.Sprintf("%s/%s", strings.TrimRight(requestPath, "/"), item.Name),
			}
		},
	)

	transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}
```

#### Scenario: 实现资源监听 (Watch)
- **WHEN** 处理 Watch 请求时
- **THEN** 应使用 `watchutils.StreamWatch` 工具函数
- **AND** 传入一个返回 `watch.Interface` 的闭包函数

```go
func (cluster *BackupHandler) watchBackups(c context.Context, ctx *app.RequestContext) {
	watcherFunc := func(ctx context.Context) (watch.Interface, error) {
		return cluster.DisasterClient.DisasterV1().DisasterBackups(common.DisasterSystemNamespace).Watch(ctx, matev1.ListOptions{})
	}
	watchutils.StreamWatch(c, ctx, watcherFunc)
}
```
