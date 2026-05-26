# Change: Inject User Identity into CRDs

## Why
Operator 依赖资源上的 `testudo.softcdata.com/user` 注解来记录审计事件。目前 Server 仅在策略资源中实现了该注解的注入，Cluster、Storage、AppBackup 等资源尚未支持，导致操作审计信息丢失。

## What Changes
- **规范层面**: 强制要求所有涉及 CRD 变更的 API 必须注入用户身份注解。
- **实现层面**: 
    - 修改 `internal/apis/` 下的所有 Handler (Cluster, Storage, Backup, etc.)。
    - 从 Context 中提取 JWT User Claim 并写入 Annotation。

## Impact
- **Affected Specs**: `api-standards`
- **Affected Code**: `internal/apis/*/handler.go`
