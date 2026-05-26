# 任务清单：API 增强 - 过滤参数与输入验证

## 1. 存储端点前缀验证

- [x] 1.1 修改 `internal/apis/disaster_storage/v1/handler.go` 中的 `createStorage` 方法
  - 在 `BindJSON` 后增加 `endpoint` 格式校验
  - 使用 `strings.HasPrefix(req.Endpoint, "http://") || strings.HasPrefix(req.Endpoint, "https://")`
  - 校验失败返回 `400 Bad Request`

- [x] 1.2 同步修改 `updateStorage` 方法（如允许更新 endpoint）
  - 检查是否允许更新 endpoint 字段
  - 如果允许，增加相同的校验逻辑

## 2. 策略名称列表增加启用状态过滤

- [x] 2.1 修改 `internal/apis/disaster_policy/v1/handler.go` 中的 `policyNames` 方法
  - 解析 `enabled` 查询参数 (`ctx.Query("enabled")`)
  - 在内存过滤循环中增加 `State` 判断：
    ```go
    if enabledParam != "" {
        if enabledParam == "true" && item.Spec.State != dapisv1.PolicyStateEnabled {
            continue
        }
        if enabledParam == "false" && item.Spec.State != dapisv1.PolicyStateDisabled {
            continue
        }
    }
    ```

## 3. 备份历史增加状态过滤

- [x] 3.1 修改 `internal/apis/app_backup/v1/handler.go` 中的 `getBackupHistory` 方法
  - 解析 `status` 查询参数
  - 在返回前过滤 `item.Status.History`：
    ```go
    statusFilter := string(ctx.Query("status"))
    if statusFilter != "" {
        var filtered []dapisv1.BackupRecord
        for _, rec := range item.Status.History {
            if rec.Phase == statusFilter {
                filtered = append(filtered, rec)
            }
        }
        transport.WriteSuccess(ctx, consts.StatusOK, filtered, nil)
        return
    }
    ```

## 4. 验证

- [ ] 4.1 手动测试存储创建
  - 测试 `endpoint: "192.168.1.100:9000"` → 应返回 400
  - 测试 `endpoint: "http://192.168.1.100:9000"` → 应创建成功

- [ ] 4.2 手动测试策略名称列表
  - 测试 `GET /policies/names?enabled=true` → 仅返回启用策略

- [ ] 4.3 手动测试备份历史
  - 测试 `GET /appbackups/xxx/history?status=Completed` → 仅返回成功备份
