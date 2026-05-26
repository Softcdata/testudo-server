# OpenAPI 当前清单初始扫描

- 执行时机：开始实施 `add-swagger-openapi-support` 后，生成 OpenAPI 骨架前。
- 扫描结果：项目中不存在 `openspec/specs/disaster-server-openapi.yaml`。
- 原因：此前 RunAPI 治理提案明确不生成新的 OpenAPI 契约文件，Swagger/OpenAPI 支持作为独立提案实施。
- 处理方式：本变更已新建 `openspec/specs/disaster-server-openapi.yaml`，并按 server 全量接口清单写入 156 个 operation 骨架。
