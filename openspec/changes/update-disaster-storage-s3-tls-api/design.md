# Design: Disaster Storage S3 TLS API

## 关键决策
- `addressingStyle` 作为正式 DTO 字段
- `caSecretRef` 或等价非敏感引用作为正式 DTO 字段
- 读接口只回显 `caConfigured` / `caSecretRef` 一类非敏感状态
- 校验接口必须与 operator runtime 使用同一参数集合

## 编辑语义
- 未携带 CA 相关字段：保持不变
- 显式替换 CA：更新 Secret 内容
- 显式删除 CA：清空引用并删除 Secret

## 兼容边界
- 默认值为 `PathStyle`
- 未显式配置时行为与旧版本一致
