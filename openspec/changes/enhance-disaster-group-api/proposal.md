---
name: enhance-disaster-group-api
description: 增强容灾组 API，添加删除接口、详情页实例列表、以及列表搜索/筛选功能
status: draft
---

# 增强容灾组 API

## 背景

当前容灾组 (`DisasterGroup`) API 存在以下问题：
1. **删除接口已存在但未在路由中注册** - `deleteGroup` 方法已实现但路由表中有 `DELETE` 路由
2. **详情查询缺少实例信息** - 查询容灾组详情时，未返回该组管理的容灾实例列表及其状态
3. **列表查询缺少搜索和筛选功能** - 无法根据关键字搜索，无法按实例状态筛选
4. **列表汇总缺少异常容灾组个数** - 前端列表页只能展示实例总数，无法直接展示当前过滤条件下的异常容灾组数量

## 目标

1. 确认删除接口正常工作
2. 增强 `getGroup` 详情接口，返回该组管理的所有 `DisasterInstance` 及其状态
3. 增强 `listGroups` 列表接口，支持：
   - **关键字搜索**: 匹配容灾组名称、标签、描述、或包含的实例命名空间
   - **状态筛选**: 全部、运行中 (Protected)、暂停 (Paused)、异常 (Failed/Error)
4. 增强 `listGroups` 响应汇总，在 `meta.summary` 中返回：
   - `instanceCount`: 当前过滤结果中的实例总数
   - `abnormalCount`: 当前过滤结果中的异常容灾组个数

## 设计

### 1. 确认删除接口

检查 `router.go`，确认 `DELETE /groups/:name` 路由已注册且指向 `deleteGroup` 方法。

**当前状态**: ✅ 已存在 (Line 19: `g.DELETE("/groups/:name", h.deleteGroup)`)

### 2. 增强详情接口 - 返回实例列表

#### 2.1 修改 `DisasterGroupDTO`

在 `types.go` 中添加 `Instances` 字段：

```go
type DisasterGroupDTO struct {
    // ... existing fields ...
    
    // 新增：该组管理的容灾实例列表
    Instances []InstanceSummaryDTO `json:"instances,omitempty"`
}

// InstanceSummaryDTO 容灾实例摘要（用于组详情展示）
type InstanceSummaryDTO struct {
    Name             string `json:"name"`
    Namespace        string `json:"namespace"`
    FsmState         string `json:"fsmState"`    // Pending, Protected, Paused, Failed, etc.
    PrimaryCluster   string `json:"primaryCluster"`
    SecondaryCluster string `json:"secondaryCluster"`
    Level            int    `json:"level"`       // 该实例所在的层级 (1-indexed)
}
```

#### 2.2 修改 `getGroup` Handler

在 `handler.go` 的 `getGroup` 方法中：
1. 获取 `DisasterGroup` 对象
2. 遍历 `Spec.Levels`，提取所有实例名称
3. 批量查询这些 `DisasterInstance` 对象
4. 构建 `InstanceSummaryDTO` 列表并附加到响应

```go
func (h *GroupHandler) getGroup(c context.Context, ctx *app.RequestContext) {
    name := ctx.Param("name")
    
    item, err := h.DisasterClient.DisasterV1().DisasterGroups("disaster-system").Get(c, name, matev1.GetOptions{})
    if err != nil {
        // ... error handling ...
    }
    
    dto := ConvertToDisasterGroupDTO(item)
    
    // 新增：聚合实例信息
    dto.Instances = h.collectInstanceSummaries(c, item.Spec.Levels)
    
    transport.WriteSuccess(ctx, consts.StatusOK, dto, nil)
}

func (h *GroupHandler) collectInstanceSummaries(c context.Context, levels [][]string) []InstanceSummaryDTO {
    var result []InstanceSummaryDTO
    
    for levelIdx, instanceNames := range levels {
        for _, instName := range instanceNames {
            inst, err := h.DisasterClient.DisasterV1().DisasterInstances("disaster-system").Get(c, instName, matev1.GetOptions{})
            if err != nil {
                // 实例不存在或获取失败，添加占位条目
                result = append(result, InstanceSummaryDTO{
                    Name:     instName,
                    FsmState: "Unknown",
                    Level:    levelIdx + 1,
                })
                continue
            }
            
            result = append(result, InstanceSummaryDTO{
                Name:             inst.Name,
                Namespace:        inst.Namespace,
                FsmState:         inst.Status.FsmState,
                PrimaryCluster:   inst.Status.PrimaryCluster,
                SecondaryCluster: inst.Status.SecondaryCluster,
                Level:            levelIdx + 1,
            })
        }
    }
    
    return result
}
```

### 3. 增强列表接口 - 搜索和筛选

#### 3.1 支持的查询参数

| 参数名 | 类型 | 说明 |
|--------|------|------|
| `keyword` | string | 模糊匹配：组名称、描述、标签值、包含的实例命名空间 |
| `status` | string | 状态筛选：`all`, `running`, `paused`, `error` |
| `limit` | int | 分页大小，默认 10，-1 表示全部 |
| `offset` | int | 分页偏移 |

#### 3.2 状态映射

| API `status` 参数 | 匹配的 FsmState |
|-------------------|-----------------|
| `all` / 空 | 不过滤 |
| `running` | `Protected`, `Initializing` |
| `paused` | `Paused` |
| `error` | `Failed`, `ConfigError`, `NotFound`, `FailingOver`, `FailingBack` (异常/操作中) |

**注意**: 状态筛选基于组内实例的聚合状态。只要组内有任意一个实例匹配该状态，该组就会被返回。

#### 3.3 列表汇总统计

`GET /apis/disastergroups.testudo.softcdata.com/v1/groups` 必须在标准 collection 响应的 `meta.summary` 中返回以下字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `instanceCount` | int | 过滤后、分页前的所有容灾组中，`instances[]` 条目的总数 |
| `abnormalCount` | int | 过滤后、分页前的异常容灾组个数，每个容灾组最多计数一次 |

异常容灾组的判定口径必须表达组自身健康异常，不得把纯操作中状态计入异常。满足以下任一条件时，该容灾组计入 `abnormalCount`：

- `DisasterGroup.status.reason` 非空，例如 `InstanceNotFound`、`InstanceFailed`
- `DisasterGroup.status.conditions` 中存在 `type=Error` 且 `status=True`
- server 推导的组展示态 `status.fsmState` 为 `Degraded`
- 组内成员实例展示状态为 `Failed`、`ConfigError`、`NotFound`
- 组内成员实例存在非空 `reason`

`FailingOver`、`FailingBack` 只表示切换动作进行中，单独出现时不得计入 `abnormalCount`。如果同一容灾组已经因为成员失败、成员缺失、配置异常、组级 `reason` 非空满足异常条件，则仍计入一次。

统计顺序必须固定为：
1. 读取容灾组并补齐成员实例摘要
2. 应用 `keyword` 与 `status` 过滤
3. 计算 `meta.summary.instanceCount` 与 `meta.summary.abnormalCount`
4. 执行分页并返回当前页数据

因此，当 `page`、`limit` 只返回当前页的一部分容灾组时，`meta.summary` 仍表示过滤后完整结果集的汇总。

#### 3.4 修改 `listGroups` Handler

```go
func (h *GroupHandler) listGroups(c context.Context, ctx *app.RequestContext) {
    qParams := transport.ParseOptions(c, ctx)
    keyword := string(ctx.Query("keyword"))
    statusFilter := string(ctx.Query("status"))
    
    list, err := h.DisasterClient.DisasterV1().DisasterGroups("disaster-system").List(c, matev1.ListOptions{})
    if err != nil {
        transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
        return
    }
    
    // 预加载所有实例用于筛选
    allInstances := h.preloadInstances(c)
    
    var filtered []DisasterGroupDTO
    for i := range list.Items {
        group := &list.Items[i]
        dto := ConvertToDisasterGroupDTO(group)
        
        // 聚合实例信息
        dto.Instances = h.collectInstanceSummariesWithCache(group.Spec.Levels, allInstances)
        
        // 关键字搜索
        if !h.matchKeyword(dto, keyword) {
            continue
        }
        
        // 状态筛选
        if !h.matchStatus(dto.Instances, statusFilter) {
            continue
        }
        
        dto.Status.Summary = fmt.Sprintf("%d Levels, %d Instances", len(group.Spec.Levels), group.Status.TotalInstances)
        filtered = append(filtered, dto)
    }

    summary := summarizeDisasterGroupList(filtered)
    
    // 分页
    pagedDtos, total := transport.Paginate(filtered, qParams)
    
    // 构建响应
    requestPath := string(ctx.URI().Path())
    data, meta := transport.BuildCollectionResponse(
        requestPath,
        "disasterGroup",
        pagedDtos,
        qParams,
        total,
        nil,
        nil,
    )
    meta.Summary = summary
    transport.WriteSuccess(ctx, consts.StatusOK, data, meta)
}

func (h *GroupHandler) matchKeyword(dto DisasterGroupDTO, keyword string) bool {
    if keyword == "" {
        return true
    }
    kw := strings.ToLower(keyword)
    
    // 匹配名称
    if strings.Contains(strings.ToLower(dto.Name), kw) {
        return true
    }
    
    // 匹配描述
    if strings.Contains(strings.ToLower(dto.Description), kw) {
        return true
    }
    
    // 匹配标签值
    // (如果有 Labels 字段...)
    
    // 匹配实例命名空间
    for _, inst := range dto.Instances {
        if strings.Contains(strings.ToLower(inst.Namespace), kw) {
            return true
        }
    }
    
    return false
}

func (h *GroupHandler) matchStatus(instances []InstanceSummaryDTO, statusFilter string) bool {
    if statusFilter == "" || statusFilter == "all" {
        return true
    }
    
    for _, inst := range instances {
        switch statusFilter {
        case "running":
            if inst.FsmState == "Protected" || inst.FsmState == "Initializing" {
                return true
            }
        case "paused":
            if inst.FsmState == "Paused" {
                return true
            }
        case "error":
            if inst.FsmState == "Failed" || inst.FsmState == "ConfigError" || inst.FsmState == "NotFound" || inst.FsmState == "FailingOver" || inst.FsmState == "FailingBack" {
                return true
            }
        }
    }
    
    return false
}
```

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `disaster-server/internal/apis/disaster_group/v1/types.go` | 修改 | 添加 `Instances` 字段和 `InstanceSummaryDTO` 类型 |
| `disaster-server/internal/apis/disaster_group/v1/handler.go` | 修改 | 增强 `getGroup` 和 `listGroups` 方法，补充 `meta.summary.instanceCount` 与 `meta.summary.abnormalCount` |
| `disaster-server/openspec/specs/disaster-server-openapi.yaml` | 修改 | 更新 `GET /groups` 的 Swagger/OpenAPI 字段说明和响应示例 |
| RunAPI/Apipost `GET /groups` 接口 | 修改 | 更新 `meta.summary.abnormalCount` 字段说明和响应示例 |

## 测试计划

1. **删除接口测试**: 确认 `DELETE /groups/:name` 正常删除容灾组
2. **详情接口测试**: 验证返回的实例列表包含正确的状态信息
3. **列表搜索测试**: 使用关键字搜索，验证匹配逻辑
4. **列表筛选测试**: 按状态筛选，验证返回结果
5. **列表汇总测试**: 验证 `instanceCount` 与 `abnormalCount` 基于过滤后、分页前的完整结果集计算，且每个异常容灾组只计数一次
6. **文档同步测试**: 验证 Swagger/OpenAPI 与 RunAPI 均包含 `meta.summary.abnormalCount` 的字段说明和示例

## API 文档更新

更新 Swagger/OpenAPI 与 Apipost 中的 DisasterGroup 相关 API 文档。

## 风险与注意事项

1. **性能**: 列表接口会批量查询实例，当组数量和实例数量较大时可能有性能影响。考虑使用缓存或 Informer。
2. **实例不存在**: 组内配置的实例名可能不存在（被删除或配置错误），需要优雅处理。
3. **统计口径一致性**: `abnormalCount` 必须表达组健康异常，不得被纯 `FailingOver`、`FailingBack` 操作中状态抬高。
