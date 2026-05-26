## ADDED Requirements

### Requirement: RunAPI 详细说明必须采用五段结构
系统必须 (MUST) 为 RunAPI 中每个接口维护结构化详细说明，详细说明必须包含接口用途、控制资源、入参字段、返回字段、可能错误五类信息。

#### Scenario: 新增接口详细说明
- **Given** RunAPI 中新增一个接口
- **When** 编写接口详细说明
- **Then** 详细说明必须包含 `## 1. 接口用来干什么`
- **And** 详细说明必须包含 `## 2. 控制哪些资源`
- **And** 详细说明必须包含 `## 3. 入参详细说明`
- **And** 详细说明必须包含 `## 4. 返回详细说明`
- **And** 详细说明必须包含 `## 5. 可能返回的错误`

#### Scenario: 更新已有接口详细说明
- **Given** RunAPI 中已有接口存在历史详细说明
- **When** 补充标准化详细说明
- **Then** 新增五段说明必须放在最前面
- **And** 历史详细说明必须原样保留在 `## 原有说明` 后面
- **And** 历史详细说明不得删减、改写、重排

### Requirement: 入参与返回字段必须达到字段级说明粒度
系统必须 (MUST) 为每个接口的入参与返回字段提供字段级中文说明，说明必须覆盖字段含义、取值、目的、必填性、来源与约束。

#### Scenario: 编写入参说明
- **Given** 接口存在请求体、路径参数、查询参数、Header 参数
- **When** 编写 `## 3. 入参详细说明`
- **Then** 每个字段必须说明字段路径
- **And** 每个字段必须说明类型
- **And** 每个字段必须说明是否必传
- **And** 每个字段必须说明可传入值
- **And** 每个字段必须说明中文含义
- **And** 每个字段必须说明传入目的
- **And** 每个字段必须说明约束与默认值

#### Scenario: 编写返回说明
- **Given** 接口存在响应字段
- **When** 编写 `## 4. 返回详细说明`
- **Then** 每个字段必须说明字段路径
- **And** 每个字段必须说明类型
- **And** 每个字段必须说明可能取值
- **And** 每个字段必须说明中文含义
- **And** 每个字段必须说明字段来源
- **And** 每个字段必须说明字段用途与为空条件

### Requirement: 字段说明必须基于 server 与 operator 调用链取证
系统必须 (MUST) 在编写 RunAPI 详细说明前完成 server 与 operator 调用链取证，字段含义不得仅凭字段名推断。

#### Scenario: 差异对账后逐接口取证
- **Given** server 接口清单、RunAPI 接口清单与差异清单均已完成
- **When** 开始编写某一个接口的详细说明
- **Then** 必须先查询该接口的 server 调用链
- **And** 必须查询该接口的 operator 调用链
- **And** 不得在差异对账完成前批量查询接口级 operator 调用链并修改 RunAPI

#### Scenario: 确认入参字段语义
- **Given** server 接口请求字段会写入 operator CRD Spec
- **When** 编写该字段的中文含义、可传入值、传入目的、约束与默认值
- **Then** 必须读取 server request struct
- **And** 必须读取 server 校验、默认值与转换逻辑
- **And** 必须读取 operator CRD Spec 类型
- **And** 必须读取 operator controller 对该字段的使用逻辑

#### Scenario: 确认返回字段语义
- **Given** server 接口返回字段来自 CRD Status
- **When** 编写该字段的可能取值、中文含义、字段来源与用途
- **Then** 必须读取 server DTO 与转换函数
- **And** 必须读取 operator CRD Status 类型
- **And** 必须读取 operator controller 状态回写逻辑
- **And** 必须说明该字段由 operator reconcile 后回写

#### Scenario: 字段证据不足
- **Given** 字段含义无法通过 server 与 operator 调用链确认
- **When** 编写 RunAPI 详细说明
- **Then** 对应字段必须标记为 `待确认`
- **And** 对应接口 Todo 必须保留复核项
- **And** 不得以字段名猜测业务含义

### Requirement: 控制资源说明必须覆盖直接资源与下层影响资源
系统必须 (MUST) 在每个接口详细说明中明确该接口控制的资源范围，包含 server 直接操作资源与 operator 下层影响资源。

#### Scenario: 接口直接操作 CRD
- **Given** server handler 创建、查询、更新、删除 Kubernetes CRD
- **When** 编写 `## 2. 控制哪些资源`
- **Then** 必须列出 server 直接操作的 CRD 类型
- **And** 必须列出资源命名空间与作用范围

#### Scenario: operator 下层创建资源
- **Given** operator controller 基于该 CRD 创建、查询、更新、删除下层资源
- **When** 编写 `## 2. 控制哪些资源`
- **Then** 必须列出下层影响资源
- **And** 涉及 Velero 资源时必须列出对应 Velero 类型
- **And** 涉及 Kubernetes 原生资源时必须列出对应资源类型
- **And** 涉及源集群与目标集群时必须说明跨集群作用范围

### Requirement: 错误说明必须区分 server 当场错误与 operator 后续失败
系统必须 (MUST) 在 RunAPI 详细说明中区分接口调用当场返回错误，以及接口成功后由 operator 回写的异步失败。

#### Scenario: 编写 server 当场错误
- **Given** server handler 可能直接返回错误
- **When** 编写 `## 5. 可能返回的错误`
- **Then** 必须列出 HTTP 状态码
- **And** 必须列出业务码
- **And** 必须列出错误原因
- **And** 必须列出触发条件

#### Scenario: 编写 operator 后续失败
- **Given** 接口成功后由 operator 继续执行下层任务
- **When** 编写 `## 5. 可能返回的错误`
- **Then** 必须说明下层执行失败不表现为本接口 HTTP 错误
- **And** 必须列出 operator 回写失败的 Status 字段路径
- **And** 必须说明调用方需要通过详情接口、列表接口以及事件流接口查看最终状态

### Requirement: RunAPI 更新前必须完成全量对账
系统必须 (MUST) 先完成 server 接口清单、RunAPI 接口清单与差异清单，再按接口粒度更新 RunAPI。

#### Scenario: 生成 server 接口清单
- **When** 启动 RunAPI 文档完善实施
- **Then** 必须读取 `internal/router/router.go`
- **And** 必须读取每个模块的 `router.go`
- **And** 必须输出包含模块、方法、路径、handler、直接操作资源、WebSocket 标记的 server 接口清单

#### Scenario: 生成 RunAPI 接口清单
- **When** 对 RunAPI 文档进行更新前
- **Then** 必须拉取 RunAPI 全量目录与接口列表
- **And** 必须记录接口 ID、目录 ID、目录名称、方法、路径、已有详细说明状态

#### Scenario: 生成差异清单
- **Given** server 接口清单与 RunAPI 接口清单均已完成
- **When** 执行接口对账
- **Then** 必须使用 `METHOD + 标准化路径` 作为主键
- **And** 必须输出 RunAPI 缺失接口清单
- **And** 必须输出 RunAPI 已存在接口清单
- **And** 必须输出模块疑似错位接口清单
- **And** 必须输出需要人工复核接口清单

#### Scenario: 生成逐接口勾选清单
- **Given** 差异清单已经完成
- **When** 生成实施 Todo
- **Then** 必须输出按模块组织的逐接口勾选清单
- **And** 每个清单项必须包含方法、路径、RunAPI 状态、目标动作、处理状态
- **And** 清单必须包含 RunAPI 缺失接口、已存在接口、模块疑似错位接口、需要人工复核接口

#### Scenario: 单接口完成后立即勾选
- **Given** 某个接口已完成调用链取证、详细说明写入、RunAPI 回读验证
- **When** 标记该接口处理完成
- **Then** 必须立即勾选该接口清单项
- **And** 不得在一批接口全部完成后再集中勾选
- **And** 存在 `待确认` 字段的接口不得勾选完成
