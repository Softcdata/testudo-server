# Change: 为集群 API 增加 Velero 安装镜像与拉取凭据契约

## Why
当前 `disaster_cluster` API 只允许维护 `imageSources` 的 alias -> registry 映射，不支持“添加集群时为 Velero 安装配置客户镜像源和拉取凭据”。

而在当前代码中，`imageSources` 已经服务于实例镜像改写，不适合作为 Velero 安装镜像和凭据入口。若 server 不先定义独立 API 契约，operator 侧即使补齐了 Helm values overlay 和 target secret 同步，也没有稳定的上游输入来源。

## What Changes

### 1. cluster create/update/patch API 增加独立的 `veleroInstall` 输入
- 允许在 cluster 写请求中提供 `veleroInstall.imageRegistry`。
- 允许在 `veleroInstall` 中直接提供 write-only 的 `username`、`password`。
- 允许通过 `veleroInstall.removeCredential` 表达显式删凭据。
- 这些字段只用于 server 生成或更新管理平面 Secret 与 Cluster 非敏感配置，不直接回写明文凭据。

### 2. cluster detail/list API 增加 Velero 安装配置的脱敏回显
- 返回 `veleroInstall.imageRegistry` 等非敏感配置。
- 返回 `veleroInstall.credentialConfigured` 一类脱敏状态。
- 当管理平面 Secret 中存在可解析用户名时，返回 `veleroInstall.username` 供编辑态展示。
- 不回显 `veleroInstall.password` 与 dockerconfigjson 内容。

### 3. server 负责维护管理平面 dockerconfigjson Secret
- 每个 `Cluster` 对应一个管理平面 `kubernetes.io/dockerconfigjson` Secret。
- Secret 名称由 server 稳定生成，不由客户端传入。
- server 把非敏感的 `registryCredentialSecretRef` 写回 `Cluster.spec.veleroInstall`。

### 4. 定义修改、保持与删除语义
- 修改镜像源前缀：覆盖 `imageRegistry`
- 修改凭据：轮换 Secret 内容
- 清空镜像源配置：PATCH 显式提交 `veleroInstall.imageRegistry=""`，删除 server 管理的 Secret 并清空整段 `Cluster.spec.veleroInstall`
- 删除凭据：提交 `removeCredential=true`；PATCH 显式提交 `veleroInstall.username=""` 时执行同一删除语义，删除 server 管理的 Secret 并清空引用
- 请求未携带凭据：保持现状，不视为清空

### 5. E2E 验收以真实 API 黑盒链路为准
- 联调入口固定为 cluster create/patch API，不直接伪造 handler 内部输入。
- 认证私有仓库固定使用 `registry:2 + htpasswd`，不引入 Harbor。
- E2E 需要验证的 server 侧事实：
  - create 请求会创建 `cluster-velero-regcred-<cluster-name>`
  - detail/list 回显 `imageRegistry`、`credentialConfigured` 与可解析的 `username`
  - detail/list 不回显 `password` 与 dockerconfigjson 内容
  - patch 请求能区分“轮换凭据”“显式清空镜像源配置”“显式删除凭据”“未携带凭据保持不变”

## Non-Goals
- 不改变 `imageSources` 的 alias -> registry 语义。
- 不在 server 侧处理目标集群 Secret 同步。
- 不允许客户端直接指定管理平面 Secret 名称或目标集群 Secret 名称。
- 不在首期支持逐镜像单独 repository 配置。
- 不在首期引入 Harbor、Nexus 一类重量级制品仓库作为联调依赖。

## Impact
- Affected specs:
  - `disaster_cluster`
- Affected code:
  - `internal/apis/disaster_cluster/v1/types.go`
  - `internal/apis/disaster_cluster/v1/handler.go`
  - `internal/apis/disaster_cluster/v1/*_test.go`
- Cross-repo impact:
  - `disaster-operator`：消费 `veleroInstall.imageRegistry` 与 `registryCredentialSecretRef`
  - `cluster-disaster-web`：新增 Velero 镜像源与凭据录入 UI

## Risks
- 若 `veleroInstall` 与 `imageSources` 的边界不清晰，后续实现仍会混淆两个能力域。
- 写请求的“未携带凭据”与“显式清空凭据”若不区分，会造成误删。
- 如果 Secret 命名规则不稳定，会增加 operator 对齐和清理复杂度。
