# 项目背景

## 目标
`disaster-server` 是灾难恢复系统的后端服务，旨在提供 RESTful API 以管理和监控 Kubernetes 环境中的灾难恢复资源。它与 `disaster-operator` 紧密协作，负责处理用户请求、管理 CRD（如 `DisasterBackup`, `DisasterCluster` 等）以及提供系统状态的可视化数据支持。

## 技术栈
- **编程语言**: Go (Golang)
- **Web 框架**: Hertz (CloudWeGo)
- **Kubernetes 集成**: client-go, controller-runtime
- **CLI 工具**: Cobra
- **配置管理**: Viper
- **日志**: Zap (hertzzap)
- **部署**: Docker, Helm

## 项目规范

### 代码风格
- 遵循标准 Go 代码规范 (Effective Go)。
- 使用 `gofmt` 和 `goimports` 进行代码格式化。
- 变量和函数命名采用驼峰式 (CamelCase)。

### 架构模式
- **分层架构**:
  - `cmd/`: 应用程序入口和 CLI 命令定义。
  - `internal/apis/`: API 路由和处理逻辑 (Handlers)。
  - `internal/kube/`: Kubernetes 客户端封装和资源操作。
  - `internal/middleware/`: HTTP 中间件 (日志, 恢复, JWT 等)。
- **RESTful API**: 使用 Hertz 框架构建 REST 风格的接口。
- **Operator 模式集成**: 通过 Kubernetes 客户端直接操作 CRD 资源。

### 测试策略
- 使用 Go 标准库 `testing` 进行单元测试。
- 关键逻辑（如 Kubernetes 资源操作）应包含集成测试。

### Git 工作流
- 主分支为 `main`。
- 所有变更通过 Pull Request (PR) 合并。
- 提交信息应清晰描述变更内容。

### OpenSpec 语言规范
所有 OpenSpec 文档（包括 proposal, specs, tasks 等）必须使用**中文**编写。

## 领域背景
- **灾难恢复 (Disaster Recovery)**: 系统核心关注数据的备份、恢复和跨集群容灾。
- **Kubernetes CRDs**: 需要深入理解以下自定义资源：
  - `DisasterBackup`: 定义备份策略和状态。
  - `DisasterCluster`: 管理参与容灾的 Kubernetes 集群。
  - `DisasterConfig`: 系统级配置。
  - `DisasterJob`: 执行具体的备份或恢复任务。
  - `DisasterStorage`: 定义备份存储后端。

## 重要约束
- **Kubernetes 依赖**: 系统必须运行在 Kubernetes 集群中或能够访问 Kubernetes API Server。
- **Operator 耦合**: 服务逻辑依赖于 `disaster-operator` 定义的 CRD 结构，需保持版本兼容。

## 外部依赖
- **Kubernetes API Server**: 核心交互组件。
- **disaster-operator**: 提供 CRD 定义和底层控制器逻辑。

