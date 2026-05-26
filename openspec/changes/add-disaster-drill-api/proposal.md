---
name: add-disaster-drill-api
description: 新增容灾演练 (DisasterDrill) API，支持实例演练和容灾组演练，包含创建、查询、确认执行和删除功能
status: draft
---

# 新增容灾演练 API

## Why (背景)

当前 `disaster-server` 缺少容灾演练 (Drill) 的 API 支持。容灾演练是验证灾备系统恢复能力的关键功能：

1. **验证数据完整性**: 备份的数据是否真的能恢复？
2. **验证业务可用性**: 恢复后的应用是否能正常启动？
3. **验证流程有效性**: RTO 是否满足预期？

`disaster-operator` 已实现 `DisasterDrill` CRD 及其 Controller，现需在 Server 层暴露 RESTful API 供前端调用。

## What Changes (变更内容)

### 1. 新增 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/apis/v1/drills` | 列出所有演练 |
| `GET` | `/apis/v1/drills/{name}` | 获取演练详情 |
| `POST` | `/apis/v1/drills` | 创建演练 |
| `GET` | `/apis/v1/drills/actions/protected-namespaces` | 查询实例/容灾组保护的命名空间 |
| `POST` | `/apis/v1/drills/{name}/confirm` | 确认执行演练 |
| `DELETE` | `/apis/v1/drills/{name}` | 删除演练 |

### 2. 数据模型

#### 2.1 DisasterDrillDTO (响应)

```go
type DisasterDrillDTO struct {
    // 基础信息
    Name         string    `json:"name"`
    Namespace    string    `json:"namespace"`
    InstanceName string    `json:"instanceName,omitempty"`  // 实例演练时有值
    GroupName    string    `json:"groupName,omitempty"`     // 容灾组演练时有值
    CreatedAt    time.Time `json:"createdAt"`

    // 配置
    TargetCluster    string            `json:"targetCluster,omitempty"`
    NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`
    SkipValidation   bool              `json:"skipValidation"`
    NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`
    SkipValidation   bool              `json:"skipValidation"`
    Confirmed        bool              `json:"confirmed"`
    
    // 失败策略回显 (自动继承 Group 配置)
    // Abort: 任何子实例演练失败，立即中止后续 Level
    // Continue: 记录失败但继续执行后续 Level
    FailurePolicy    string            `json:"failurePolicy,omitempty"`

    // 状态
    State            string    `json:"state"`         // Pending, Ready, Executing, Completed, Failed
    RestoreMode      string    `json:"restoreMode"`   // FullRestore
    OperationName    string    `json:"operationName,omitempty"`
    CurrentStep      string    `json:"currentStep,omitempty"`
    Message          string    `json:"message,omitempty"`
    
    // 时间戳
    StartTime      *time.Time `json:"startTime,omitempty"`
    ReadyTime      *time.Time `json:"readyTime,omitempty"`
    ExecutionTime  *time.Time `json:"executionTime,omitempty"`
    CompletionTime *time.Time `json:"completionTime,omitempty"`

    // 校验结果
    ValidationResults *DrillValidationDTO `json:"validationResults,omitempty"`
    
    // 容灾组演练进度 (仅容灾组演练时有值)
    GroupProgress *DrillGroupProgressDTO `json:"groupProgress,omitempty"`
}

type DrillValidationDTO struct {
    InstanceValid        bool       `json:"instanceValid"`
    ClusterReachable     bool       `json:"clusterReachable"`
    BackupAvailable      bool       `json:"backupAvailable"`
    LastDataSyncTime     *time.Time `json:"lastDataSyncTime,omitempty"`
    LastResourceSyncTime *time.Time `json:"lastResourceSyncTime,omitempty"`
}

// DrillGroupProgressDTO 容灾组演练进度
type DrillGroupProgressDTO struct {
    TotalLevels     int                        `json:"totalLevels"`
    CompletedLevels int                        `json:"completedLevels"`
    CurrentLevel    int                        `json:"currentLevel"`
    InstanceResults []DrillInstanceResultDTO   `json:"instanceResults,omitempty"`
}

// DrillInstanceResultDTO 单个实例的演练结果
type DrillInstanceResultDTO struct {
    InstanceName string `json:"instanceName"`
    State        string `json:"state"`
    Message      string `json:"message,omitempty"`
}
```

#### 2.2 CreateDrillRequest (请求)

```go
type CreateDrillRequest struct {
    // 二选一：关联的容灾实例名称 OR 容灾组名称
    InstanceName string `json:"instanceName,omitempty"`  // 实例演练
    GroupName    string `json:"groupName,omitempty"`     // 容灾组演练 (与 instanceName 互斥)
    
    // 可选：演练目标集群 (不指定则使用 Instance 的 secondaryCluster)
    TargetCluster string `json:"targetCluster,omitempty"`
    
    // 可选：命名空间映射 (不指定则使用原始命名空间)
    NamespaceMapping map[string]string `json:"namespaceMapping,omitempty"`
    
    // 可选：跳过前置校验
    SkipValidation bool `json:"skipValidation,omitempty"`

    // 可选：自定义演练名称 (不指定则自动生成)
    Name string `json:"name,omitempty"`
    
    // 可选：指定命名空间
    Namespace string `json:"namespace,omitempty"`
}
```

**校验规则**:
- `instanceName` 和 `groupName` 必须指定其一
- 不能同时指定 `instanceName` 和 `groupName`

### 3. Handler 实现

#### 3.1 文件结构

```
internal/apis/disaster_drill/
├── v1/
│   ├── handler.go      # Handler 实现
│   ├── router.go       # 路由注册
│   └── types.go        # DTO 定义
```

#### 3.2 核心方法

```go
type DrillHandler struct {
    DisasterClient disasterclientset.Interface
    Log            logr.Logger
}

// listDrills 列出所有演练
func (h *DrillHandler) listDrills(c context.Context, ctx *app.RequestContext)

// getDrill 获取演练详情
func (h *DrillHandler) getDrill(c context.Context, ctx *app.RequestContext)

// createDrill 创建演练
func (h *DrillHandler) createDrill(c context.Context, ctx *app.RequestContext)

// confirmDrill 确认执行演练
func (h *DrillHandler) confirmDrill(c context.Context, ctx *app.RequestContext)

// deleteDrill 删除演练
func (h *DrillHandler) deleteDrill(c context.Context, ctx *app.RequestContext)
```

### 4. 路由注册

```go
// RegisterRoutes 注册演练 API 路由
func RegisterRoutes(group *route.RouterGroup, handler *DrillHandler) {
    g := group.Group("/drills")
    {
        g.GET("", handler.listDrills)
        g.GET("/:name", handler.getDrill)
        g.POST("", handler.createDrill)
        g.POST("/:name/confirm", handler.confirmDrill)
        g.DELETE("/:name", handler.deleteDrill)
    }
}
```

### 5. 查询参数支持

#### 5.1 列表接口 `/apis/v1/drills`

| 参数 | 类型 | 说明 |
|------|------|------|
| `instanceName` | string | 按关联的 Instance 过滤 |
| `groupName` | string | 按关联的 Group 过滤 (容灾组演练) |
| `state` | string | 按状态过滤: `Pending`, `Ready`, `Executing`, `Completed`, `Failed` |
| `namespace` | string | 按命名空间过滤 |
| `limit` | int | 分页大小，默认 10 |
| `page` | int | 页码，默认 1 |

### 6. 演练生命周期

```
创建 (POST /drills)
    ↓
校验阶段 (自动)
    ↓ (state: Pending → Ready)
等待确认 (state: Ready)
    ↓
确认执行 (POST /drills/{name}/confirm)
    ↓ (state: Ready → Executing)
执行阶段
    ↓ (state: Executing → Completed/Failed)
删除 (DELETE /drills/{name})
```

## Impact (影响)

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/apis/disaster_drill/v1/types.go` | DTO 定义 |
| `internal/apis/disaster_drill/v1/handler.go` | Handler 实现 |
| `internal/apis/disaster_drill/v1/router.go` | 路由注册 |

### 修改文件

| 文件 | 说明 |
|------|------|
| `internal/apis/router.go` | 注册 drill 路由组 |

## 非目标 (Non-Goals)

本提案**不包含**：
1. ❌ WebSocket 实时推送演练状态
2. ❌ 演练调度 (定时触发)
3. ❌ 演练报告生成

## 与 Operator 的集成

Server 层仅负责 CRUD 操作，实际的演练逻辑由 `disaster-operator` 中的 Controller 执行：

```
┌─────────────────┐     ┌─────────────────────┐     ┌──────────────────┐
│  disaster-web   │ --> │  disaster-server    │ --> │ disaster-operator│
│  (前端 UI)      │     │  (RESTful API)      │     │ (CRD Controller) │
└─────────────────┘     └─────────────────────┘     └──────────────────┘
     创建演练              操作 DisasterDrill CR       Reconcile 执行
     确认执行              Patch confirmed=true        恢复 + 扩容
     查看状态              读取 CR Status             状态更新
```

## 前端集成建议

### 演练列表页
- 显示所有演练及其状态
- 支持按 Instance / Group 过滤
- 显示演练类型标签：`实例演练` / `容灾组演练`
- 状态标签：`待确认 (Ready)`, `执行中 (Executing)`, `已完成 (Completed)`, `失败 (Failed)`

### 演练详情页
- 显示配置信息 (目标集群、命名空间映射)
- 显示校验结果 (Instance 有效性、集群可达性、备份可用性)
- 显示执行步骤进度
- **容灾组演练**: 显示各 Level 进度和各实例状态

### 创建演练弹窗
1. 选择演练类型：`实例演练` / `容灾组演练`
2. 根据类型选择容灾实例或容灾组
3. 可选：指定目标集群
4. 可选：配置命名空间映射
5. 创建后自动跳转到详情页等待确认

**安全校验**:
前端应校验 `TargetCluster` 和 `NamespaceMapping`，如果用户尝试在配置为 Secondary 的集群上覆盖写入且未设置 NamespaceMapping，应弹出红色高危警告。

### 确认执行按钮
- 仅在 `Ready` 状态时可用
- 点击后调用 `POST /drills/{name}/confirm`

### 容灾组演练进度展示
- 显示 Level 进度条 (如: Level 2/3)
- 列表展示各实例的演练状态
- 支持查看单个实例的详细日志
