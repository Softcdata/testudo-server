# 规范变更：删除保护与优雅删除

## ADDED Requirements

### StorageRepository 删除保护

系统必须防止正在被使用的存储仓库被意外删除。

#### Scenario: 删除被引用的存储
Given 一个 StorageRepository "s3-storage"
And 存在一个 DisasterConfig 引用了 "s3-storage"
When 用户尝试删除 "s3-storage"
Then 资源不应被立即物理删除
And 资源状态应变为 `Deleting`
And 资源 Status.Reason 应为 `DeletionBlocked`
And 资源 Status.Message 应包含引用该存储的资源信息

#### Scenario: 删除无引用的存储
Given 一个 StorageRepository "unused-storage"
And 不存在任何资源引用该存储
When 用户尝试删除 "unused-storage"
Then 资源应被成功删除

### DisasterPolicy 删除保护

系统必须防止正在被使用的策略被意外删除。

#### Scenario: 删除被引用的策略
Given 一个 DisasterPolicy "daily-backup"
And 存在一个 DisasterConfig 引用了 "daily-backup"
When 用户尝试删除 "daily-backup"
Then 资源不应被立即物理删除
And 资源状态应变为 `Deleting`
And 资源 Status.Reason 应为 `DeletionBlocked`
And 资源 Status.Message 应包含引用该策略的资源信息

## MODIFIED Requirements

### Server 删除接口行为

Server 端的删除接口不再进行应用层的依赖检查，而是依赖 Operator 的 Finalizer 机制。

#### Scenario: 调用删除接口
Given 一个被保护的资源 (Storage 或 Policy)
When 调用 DELETE API
Then API 返回 200 OK (表示删除请求已接受)
And 资源在 K8s 中进入 `Terminating` 状态
And 实际删除被 Operator 阻塞直到依赖清除
