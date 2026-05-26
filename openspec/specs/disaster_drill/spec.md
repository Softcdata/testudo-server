# disaster_drill Specification

## Purpose
TBD - created by archiving change add-drill-cleanup-api. Update Purpose after archive.
## Requirements
### Requirement: 演练清理执行 API
系统必须 (MUST) 提供确认执行容灾演练清理的 API。当用户在控制台手动验证演练环境没有问题并确认结束验证后，点击特定按钮来调用此 API 以启动资源释放。

#### Scenario: 指定成功完成的演练并触发清理
- **GIVEN** 存在一个处于 `Completed` 状态的 DisasterDrill
- **WHEN** 用户已完成验证并决定释放资源
- **AND** 客户端发送 `POST /apis/v1/drills/{name}/cleanup` 请求
- **THEN** 系统应当接受该请求
- **AND** 系统通过调用 kube client，将底层 `DisasterDrill` 资源的 `spec.cleanup` 修改为 `true`
- **AND** 返回状态码 `200 OK`，或返回包含更新后状态 (CleaningUp) 的响应 DTO

#### Scenario: 拒绝不在 Completed 状态的演练清理
- **GIVEN** 存在一个仍然在 `Executing` 状态或早已在 `CleanedUp` 的 DisasterDrill
- **WHEN** 客户端发送 `POST /apis/v1/drills/{name}/cleanup` 请求
- **THEN** 系统必须 (MUST) 拒绝修改
- **AND** 返回状态码 `400 Bad Request`
- **AND** 响应消息提示：只有 Completed 状态并且未开始清理（!cleanup）的演练可以进行清理操作。

### Requirement: 演练 DTO 格式
API 返回的演练数据必须 (MUST) 使用 DTO 格式。

#### Scenario: DisasterDrillDTO 扩充支持新状态和配置
- 必须包含字段 `cleanup: boolean`（标志用户是否触发过清理流程）。
- `state` 的可选值域更新为 `Pending, Ready, Executing, Completed, CleaningUp, CleanedUp, Failed`。

