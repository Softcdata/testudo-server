# Tasks

## 1. Proposal
- [x] 1.1 评审 route 形态、`type` 枚举、读取 DTO 字段集、错误码、action 响应结构

## 2. Server
- [x] 2.1 注册 `refresh-namespaces` action route
- [x] 2.2 为 action request 增加 `type` 校验
- [x] 2.3 使用 `RetryOnConflict` 写入 `Cluster` typed refresh signal，且只变更 `testudo.softcdata.com/refresh-cluster-stats`
- [x] 2.4 在 Cluster 详情、列表、watch DTO 与现有 `GET /clusters/names` 摘要 DTO 中暴露 workload namespace 统计字段
- [x] 2.5 返回 `202 Accepted` 与回显字段
- [x] 2.6 补 action handler / DTO / conflict retry tests，并覆盖“不写 user/audit annotation”的约束

## 3. Alignment
- [x] 3.1 与 operator 对齐 `type` 枚举、signal key、signal 生命周期语义、读取字段口径
- [ ] 3.2 与 web 对齐 loading / success / fail 提示

## 4. Verification
- [x] 4.1 `openspec validate add-cluster-namespace-refresh-action --strict`
