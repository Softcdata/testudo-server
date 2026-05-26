## 1. 契约与类型
- [x] 1.1 同步 server vendored operator modifier 类型到 pair-only
- [x] 1.2 为 server 变更补充 OpenSpec proposal 与 spec 增量

## 2. 校验链路
- [x] 2.1 将实例静态校验从 `transform` 切换到 `pair`
- [x] 2.2 将 live validation 的 reversible 路径提取切换到 `pair.path`
- [x] 2.3 统一旧写法拒绝错误口径为 pair canonical form

## 3. 测试与示例
- [x] 3.1 将 handler tests 中的 reversible 请求样例全部切换到 pair-only
- [x] 3.2 将 `modifierRulesText` 示例切换到 pair-only
- [x] 3.3 补充旧 `transform` 被拒绝的回归测试

## 4. 验证
- [x] 4.1 `go test ./internal/apis/disaster_instance/v1 -count=1`
- [x] 4.2 `openspec validate refactor-server-modifier-pair-only-alignment --strict`
