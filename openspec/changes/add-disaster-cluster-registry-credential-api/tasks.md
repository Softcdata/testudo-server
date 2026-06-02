# Tasks

## 1. Proposal
- [x] 1.1 评审 `veleroInstall` 与 `imageSources` 的职责拆分
- [x] 1.2 评审 `veleroInstall.imageRegistry` / `registryCredentialSecretRef` / `credentialConfigured` 命名
- [x] 1.3 评审“未携带凭据”“显式删除凭据”“仅修改镜像源前缀”三种编辑语义

## 2. Server
- [x] 2.1 扩展 cluster create/update/patch 请求 DTO，支持 `veleroInstall` 输入对象
- [x] 2.2 根据 write-only 凭据生成或更新 cluster 级 dockerconfigjson Secret
- [x] 2.3 将非敏感 Secret 引用写回 `Cluster.spec.veleroInstall`
- [x] 2.4 为 detail/list API 增加 `veleroInstall.credentialConfigured` 脱敏回显
- [x] 2.5 为 detail/list/create/patch/watch 响应补充 `veleroInstall.username` 回显，保持 `password` 不回显
- [x] 2.6 补“创建 / 轮换 / 删除 / 显式传空清空 / 未修改保持不变 / 仅修改 imageRegistry / username 回显” handler tests

## 3. Alignment
- [x] 3.1 与 operator 对齐 `veleroInstall` 字段命名和生命周期契约
- [x] 3.2 与 web 对齐编辑态“未修改”和“显式清空”两种语义

## 4. Verification
- [x] 4.1 `openspec validate add-disaster-cluster-registry-credential-api --strict`
- [x] 4.2 对齐共享 `e2e-test-procedure.md`，固定 `registry:2 + htpasswd` 作为轻量认证 registry 模型
- [x] 4.3 真实环境联调：验证 create 请求创建 `cluster-velero-regcred-<cluster>` 且 detail/list 仅回显脱敏字段
- [x] 4.4 真实环境联调：验证 patch 轮换与 `removeCredential` 删除语义
