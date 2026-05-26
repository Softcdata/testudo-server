## ADDED Requirements

### Requirement: 集群 API 必须支持独立的 Velero 安装配置输入
系统必须 (MUST) 允许在集群创建和更新请求中通过独立的 `veleroInstall` 对象提交 Velero 安装镜像源和 write-only 凭据输入，而不得复用 `imageSources`。

#### Scenario: 创建集群时提交 Velero 安装镜像源和凭据
- **When** 客户端创建一个 `Cluster`，并提交 `veleroInstall.imageRegistry`、`username` 和 `password`
- **Then** Server 必须生成或更新一个管理平面 dockerconfigjson Secret
- **And** 必须将非敏感配置写回 `Cluster.spec.veleroInstall`
- **And** 不得将凭据明文写入 `Cluster` 的公开 DTO 字段

### Requirement: 集群查询接口必须脱敏回显 Velero 安装配置状态
系统必须 (MUST) 在集群详情和列表接口中返回 Velero 安装配置的非敏感状态，而不得泄露凭据内容。

#### Scenario: 查询集群时返回镜像源前缀和凭据状态
- **When** 客户端查询一个已配置 Velero 安装镜像源与凭据的 `Cluster`
- **Then** 返回结果必须包含 `veleroInstall.imageRegistry`
- **And** 返回结果必须能表明 `veleroInstall.credentialConfigured=true`
- **And** 不得返回用户名、密码或 dockerconfigjson 原文

### Requirement: 更新接口必须区分凭据轮换与保持不变
系统必须 (MUST) 区分“显式轮换/删除凭据”和“本次更新未修改凭据”这两种语义。

#### Scenario: 未携带凭据字段时保持现有 Secret 不变
- **Given** 一个 `Cluster` 已经配置了 Velero 安装凭据
- **When** 客户端更新该 `Cluster`，但本次请求未携带凭据字段
- **Then** Server 必须保留现有 Secret 内容不变
- **And** 不得将其视为清空凭据

#### Scenario: 显式删除凭据时移除对应 Secret 条目
- **Given** 一个 `Cluster` 已经配置了 Velero 安装凭据
- **When** 客户端显式提交删除 Velero registry 凭据的请求
- **Then** Server 必须删除对应的管理平面 Secret 或清空其内容
- **And** 必须清空 `Cluster.spec.veleroInstall.registryCredentialSecretRef`

#### Scenario: 仅修改镜像源前缀时保持凭据不变
- **Given** 一个 `Cluster` 已经配置了 Velero 安装镜像源和凭据
- **When** 客户端只修改 `veleroInstall.imageRegistry`
- **Then** Server 必须更新镜像源前缀
- **And** 必须保留现有 Secret 和凭据引用不变
