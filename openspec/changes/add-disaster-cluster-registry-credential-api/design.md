# Design: Disaster Cluster Velero Install API

## 背景
server 需要成为“添加集群时配置 Velero 安装镜像源和拉取凭据”的唯一写入口，同时避免把密码与 dockerconfigjson 这类敏感值直接暴露给 `Cluster` DTO 与查询接口。

当前 `imageSources` 已经服务于实例镜像改写场景，因此新的 API 契约必须独立建模为 `veleroInstall`，而不是向 `imageSources[]` 继续塞凭据字段。

## 关键决策

### D1. `veleroInstall` 使用独立输入模型
- create/update/patch 可接收：
  - `veleroInstall.imageRegistry`
  - `veleroInstall.username`（write-only）
  - `veleroInstall.password`（write-only）
  - `veleroInstall.removeCredential`（显式删除标记）
- detail/list 只回显：
  - `veleroInstall.imageRegistry`
  - `veleroInstall.username`（从管理平面 Secret 解析得到，解析失败时省略）
  - `veleroInstall.credentialConfigured`

### D2. 一个 Cluster 对应一个管理平面 dockerconfigjson Secret
- Secret 类型：`kubernetes.io/dockerconfigjson`
- 内容：保存 Velero 安装镜像仓库的拉取凭据
- 命名：稳定生成，例如 `cluster-velero-regcred-<cluster-name>`
- 原因：与 operator 侧“一次注入一个 target pull secret”模型对齐。

### D3. 未携带字段不代表清空
- 为避免编辑时误删，常规 update 未携带 `veleroInstall.imageRegistry` 时保持已有镜像源配置不变。
- 常规 update 未携带 `veleroInstall.username/password` 时保持已有 Secret 内容不变。
- PATCH 携带 `veleroInstall.password=""` 时保持已有 Secret 内容不变，用于兼容前端编辑页空密码框提交。
- PATCH 显式提交 `veleroInstall.imageRegistry=""` 时清空整段 `Cluster.spec.veleroInstall`，并删除 server 管理的 registry Secret。
- PATCH 显式删除凭据使用 `veleroInstall.removeCredential=true`。

### D4. operator 关心的是 Secret 引用，不是明文
- server 在内部管理 Secret 生命周期
- `Cluster` CR 只持久化：
  - `veleroInstall.imageRegistry`
  - `veleroInstall.registryCredentialSecretRef`
- operator 不从 API DTO 读取明文账号密码

### D5. 首期 E2E 固定采用 `registry:2 + htpasswd`
- 认证私有仓库统一使用官方 `registry:2` 与 `htpasswd` basic auth。
- E2E 固定从 HTTP API 发起 create/patch 请求，再以 Kubernetes Secret 与查询 API 作为验收面。
- 原因：
  - 本 change 真正需要验证的是 DTO -> Secret -> Cluster 非敏感引用的链路。
  - Harbor 过重，不适合作为当前提案的联调依赖。

## API 草案
- `veleroInstall.imageRegistry`：read/write
- `veleroInstall.username`：write/read，写入非空值且 `password` 非空时生成 Secret，读取时仅回显用户名
- `veleroInstall.password`：write-only，PATCH 传空字符串时表示不修改既有凭据
- `veleroInstall.removeCredential`：write-only
- `veleroInstall.credentialConfigured`：read-only

## Secret 生命周期
- create：若请求携带 `veleroInstall.username/password`，则生成 cluster 级 dockerconfigjson Secret
- update：若请求携带非空 `password` 和非空 `username`，则轮换 Secret 内容；若只改 `imageRegistry` 为非空值，或提交空 `password`，则保持 Secret 不变
- clear install：PATCH 显式提交 `veleroInstall.imageRegistry=""` 时删除 server 管理的 Secret 并清空整段 `Cluster.spec.veleroInstall`
- remove credential：若 `removeCredential=true`，则删除 Secret 并清空引用
- read：若 `Cluster.spec.veleroInstall.registryCredentialSecretRef` 指向 server 管理的 Secret，则 server 从该 Secret 的 `.dockerconfigjson` 中解析 username 并填入 DTO；解析失败时省略 `veleroInstall.username`，不影响 `credentialConfigured`

## E2E 验收设计
- S1 创建集群：`POST /clusters` 写入 `veleroInstall.imageRegistry + username/password`，验证管理平面 Secret 创建、`Cluster.spec.veleroInstall.registryCredentialSecretRef` 写回、查询接口回显 `username` 且不回显 `password`。
- S2 轮换凭据：`PATCH /clusters/:name` 只提交新凭据，验证 Secret 内容更新且 Secret 名称不变。
- S3 删除凭据：`PATCH /clusters/:name` 提交 `removeCredential=true`，验证 Secret 删除、引用清空、查询接口 `credentialConfigured=false`。
- S4 空密码兼容：`PATCH /clusters/:name` 提交 `veleroInstall.password=""` 时保持 Secret 和引用不变；提交 `veleroInstall.imageRegistry=""` 时清空整段 Velero 安装配置。
- 共享执行手册位于 `disaster-operator` change `add-cluster-registry-credential-flow/e2e-test-procedure.md`。

## 与 operator 的契约
- server 负责：`veleroInstall` DTO/请求体验证、Secret 生成/轮换/删除、非敏感引用写回 `Cluster`
- operator 负责：读取 `veleroInstall` 配置、同步 target secret、生成 Helm overlay、安装结果引用、远端清理
