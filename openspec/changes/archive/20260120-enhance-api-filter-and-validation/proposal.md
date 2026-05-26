# 提案：API 增强 - 过滤参数与输入验证

## 背景
根据用户反馈和实际使用需求，当前 API 存在以下待改进项：

1. **存储端点验证缺失**: 创建存储时，`endpoint` 字段未验证格式，导致用户可能输入无效端点（如缺少 `http://` 或 `https://` 前缀），造成后续 S3 连接失败。
2. **策略列表无启用状态过滤**: `/policies/names` 接口返回所有策略，前端下拉框需要额外过滤，用户体验不佳。
3. **备份历史无状态过滤**: `/appbackups/:name/history` 返回全部历史记录，无法快速获取成功备份列表用于恢复操作。

## 变更内容

### 1. 存储端点前缀验证
**接口**: `POST /storages`

**逻辑**:
- 在 `createStorage` 中增加 `endpoint` 格式验证
- 要求 `endpoint` 必须以 `http://` 或 `https://` 开头
- 否则返回 `400 Bad Request`，提示 "endpoint must start with http:// or https://"

**代码位置**: `internal/apis/disaster_storage/v1/handler.go` - `createStorage()`

### 2. 策略名称列表增加启用状态过滤
**接口**: `GET /policies/names`

**参数**: 
- `enabled` (可选): `true` 仅返回 `State=Enabled` 的策略；`false` 仅返回 `State=Disabled` 的策略；不传则返回全部

**逻辑**:
- 解析 `enabled` 查询参数
- 在内存过滤阶段增加 `State` 字段判断

**代码位置**: `internal/apis/disaster_policy/v1/handler.go` - `policyNames()`

### 3. 备份历史增加状态过滤
**接口**: `GET /appbackups/:name/history`

**参数**:
- `status` (可选): 仅返回指定状态的备份记录 (如 `Completed`, `Failed`, `InProgress`)

**逻辑**:
- 解析 `status` 查询参数
- 在返回 `history` 数组前进行内存过滤

**代码位置**: `internal/apis/app_backup/v1/handler.go` - `getBackupHistory()`

## 影响范围
- **受影响文件**:
  - `internal/apis/disaster_storage/v1/handler.go`
  - `internal/apis/disaster_policy/v1/handler.go`
  - `internal/apis/app_backup/v1/handler.go`
- **受影响规范**:
  - `api-standards` (更新接口文档)

## 变更 ID
`enhance-api-filter-and-validation`
