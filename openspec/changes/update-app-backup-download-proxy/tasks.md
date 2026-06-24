## 1. 规范与文档
- [x] 1.1 更新 `specs/app-backup`，把下载能力改写为平台代理地址与流式输出
- [x] 1.2 更新 `openspec/specs/disaster-server-openapi.yaml`，同步下载接口的路径、返回值与错误语义
- [x] 1.3 更新 RunAPI 相关说明与本地证据，明确返回值不再暴露对象存储地址

## 2. 服务端实现
- [x] 2.1 在 `internal/apis/app_backup/v1` 中新增下载票据生成与校验流程
- [x] 2.2 新增下载流入口，完成 `resource`、`data`、`all` 三种下载类型的代理转发
- [x] 2.3 扩展 `internal/storage`，补齐对象流读取以及流式输出能力
- [x] 2.4 调整下载响应结构，确保前端继续使用 `window.open(download_url)` 即可完成下载

## 3. 测试与校验
- [x] 3.1 增加下载代理、票据失效、对象存储失败、历史记录不存在的单元测试
- [x] 3.2 增加流式下载的边界测试，覆盖客户端中断与空对象列表场景
- [x] 3.3 运行 `openspec validate update-app-backup-download-proxy --strict`
