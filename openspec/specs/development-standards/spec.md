# Capability: 服务端开发标准

## Description
本规范定义了 `disaster-server` 项目中服务端开发的一般标准工作流和最佳实践。

## Requirements

### Requirement: OpenSpec 变更归档与提交
当 OpenSpec 变更提案（Change Proposal）完成并归档时，必须 (MUST) 使用标准化的 Commit Message 提交代码。

#### Scenario: 归档变更提案
- **Given** 一个变更提案的所有任务已完成
- **And** 执行了 `openspec archive <change-name>`
- **When** 提交代码到版本控制系统
- **Then** Commit Message 的标题应为 `feat(<scope>): <summary>`
- **And** Commit Message 的正文应包含已完成任务的详细列表（通常来源于 `tasks.md`）
- **And** 确保 Commit Message 清晰反映了该变更带来的价值和影响

### Requirement: 资源更新操作 (Resource Update Operations)
所有资源的更新操作必须 (MUST) 遵循 "Get-Modify-Update" 模式，以防止数据丢失或意外覆盖。

#### Scenario: 更新资源
- **Given** 客户端请求更新某个资源（如 `AppBackup`）
- **When** 处理更新请求时
- **Then** 必须先从 Kubernetes 获取（Get）该资源的最新版本
- **And** 仅将请求体中提供的 Spec 字段应用到获取到的对象上
- **And** 保持 Status 和其他未变更的字段不变
- **And** 最后执行更新（Update）操作
