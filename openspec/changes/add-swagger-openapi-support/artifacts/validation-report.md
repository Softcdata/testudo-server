# Swagger/OpenAPI 阶段校验报告

## 已执行校验
- `go run ./tools/openapi validate --spec openspec/specs/disaster-server-openapi.yaml`
- `go test ./...`
- `go test ./internal/router ./internal/openapi ./configs ./tools/openapi`
- `git diff --check`
- `openspec validate add-swagger-openapi-support --strict`

## 校验结果
- OpenAPI 版本：`3.0.3`
- OpenAPI operation 数量：156
- OpenAPI path 数量：115
- WebSocket operation 数量：24
- server 与 OpenAPI 差异：0
- RunAPI 缺失的 server 接口：0
- OpenAPI 额外接口：0
- RunAPI 额外接口：1，历史裸路径 `POST /disasterjobs.testudo.softcdata.com/v1/jobs`

## 未完成项
- 逐接口 request schema 仍为骨架。
- 逐接口 response schema 仍为骨架。
- 逐接口错误触发条件仍需按 handler 与 operator 调用链补齐。
- Swagger UI 真实浏览器截图尚未执行。
