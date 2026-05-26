# Capability: API 文档标准

## Description
定义 API 文档的生成和维护标准，确保开发人员和前端团队能够方便地测试和集成 API。

### Requirement: Postman 集合导出
所有 API 接口开发并测试完成后，必须提供对应的 Postman Collection JSON 文件，以便导入到 ApiPost 或 Postman 中进行调试。

#### Scenario: 导出接口定义
- **WHEN** 完成一组 API 接口的开发和单元测试 (例如 `AppBackup` 相关接口)
- **THEN** 必须创建一个 Postman Collection JSON 文件
- **AND** 文件应存放在 `tools/export_postman/` 目录下（或项目指定的文档目录）
- **AND** 集合中应包含该资源的所有 CRUD 操作示例
- **AND** 请求示例应包含典型的 Request Body 和预期的 Response

### Requirement: 中文命名规范
为了方便团队协作，Postman 集合中的目录和接口名称必须使用中文。

#### Scenario: 接口命名
- **WHEN** 定义 Postman 集合中的 Item 名称
- **THEN** 目录名称应使用中文描述资源类别（如 "应用备份管理"）
- **AND** 接口名称必须遵循以下映射关系：
    - List (列表) -> "列表查询"
    - Get (详情) -> "获取详情"
    - Create (创建) -> "创建"
    - Update (更新) -> "更新"
    - Delete (删除) -> "删除"
    - Watch List (全部监听) -> "事件流(全部) [WebSocket]"
    - Watch Item (单个监听) -> "事件流(单个) [WebSocket]"

### Requirement: WebSocket 接口标识
对于 WebSocket 类型的接口，必须在文档中进行明确标识。

#### Scenario: 标识 WebSocket 接口
- **WHEN** 接口为 WebSocket 类型 (如 Watch 接口)
- **THEN** 必须在 `request.description` 中说明该接口为 WebSocket 连接，并描述其用途

### Requirement: 环境变量使用
为了适应不同的部署环境，Postman 集合中的 URL 必须使用环境变量。

#### Scenario: Base URL 配置
- **WHEN** 定义请求的 URL
- **THEN** 必须使用 `{{baseurl}}` 作为 API 地址的前缀
- **AND** 不得硬编码 `http://localhost:8080` 或其他具体 IP 地址

```json
{
	"info": {
		"name": "Disaster Server API",
		"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	},
	"item": [
		{
			"name": "应用备份管理",
			"item": [
				{
					"name": "列表查询",
					"request": {
						"method": "GET",
						"url": {
							"raw": "{{baseurl}}/apis/appbackups.testudo.softcdata.com/v1/backups",
							"host": ["{{baseurl}}"],
							"path": ["apis", "appbackups.testudo.softcdata.com", "v1", "backups"]
						}
					}
				}
                // ... 其他接口
			]
		}
	]
}
```
