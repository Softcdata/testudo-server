## Context

Velero 的包含项以及排除项最终由 `resource.group` 进行匹配。备份资源清单把资源类型记录为 GVK 字符串，当前 Server 没有在接口边界完成格式转换。客户端把清单值用于恢复请求后，Velero 找不到对应 group-resource，资源不会进入恢复处理。

## Goals / Non-Goals

- Goals：在资源清单响应边界输出 Velero group-resource；在 AppRestore 写入边界兼容 GVK；统一处理包含项以及排除项；对无法解析的值给出确定性错误。
- Non-Goals：不修改 Velero 过滤算法；不修改 Operator 的 RestoreSpec 转发；不改变 Deployment selector；不为资源 Kind 手工推断复数名称。

## Decisions

- Decision：复用源备份集群客户端提供的 `RESTMapper`。清单转换在读取源集群备份资源清单后执行；AppRestore 转换在创建流程读取 AppBackup 并确定源集群后执行；更新流程读取现有 AppRestore 的源备份引用后执行。
- Decision：只把可识别的 `group/version/Kind` 以及 `version/Kind` 输入送入 RESTMapper。已经符合 group-resource 形态的字符串原样保留，通配符原样保留。
- Decision：转换失败携带字段名和原始资源值。创建以及更新接口返回 400；资源清单接口返回内部错误并记录源集群、备份名和资源值。更新 AppRestore 时，若请求更改 `backupSource`，使用新 AppBackup 的 `spec.cluster`；否则优先使用现有 `spec.sourceCluster`，该字段为空则读取现有关联 AppBackup 的 `spec.cluster`。仍无法确定源集群时拒绝 GVK 输入，不使用恢复目标集群进行映射。
- Decision：创建和更新共用一个无副作用转换函数，函数返回新切片，不修改请求对象的底层数组。

## Alternatives Considered

- 手工维护 Kind 到复数资源名称的映射：无法覆盖 CRD 和版本差异，放弃。
- 只在 Web 端转换：绕过其他 API 客户端，不能保护 Server 直连调用，放弃。
- 只在 Operator 端转换：资源清单接口仍然输出错误格式，排除项也会继续失效，放弃。

## Risks / Trade-offs

- 源集群 Discovery 不可用时，携带 GVK 的请求会明确失败；这会暴露连接问题，避免创建表面成功但实际不生效的恢复。
- 历史请求使用 group-resource 时不需要 Discovery，保持既有调用兼容。
- 资源清单接口首次读取新备份时增加一次 RESTMapper 查询；缓存沿用现有清单缓存，避免同一备份重复转换。

## Migration Plan

1. 先部署 Server 修复并验证资源清单接口返回 group-resource。
2. 使用历史备份创建包含项以及排除项均为 GVK 的 AppRestore，确认 CR 中保存 group-resource。
3. 执行 `existingResourcePolicy=update` 恢复，确认 Deployment 业务标签以及 Velero 标签更新。
4. 保留 group-resource 请求回归用例，确认升级前客户端无需修改。

## Open Questions

- 无。资源筛选字段的目标格式、映射来源以及失败行为已确定。
