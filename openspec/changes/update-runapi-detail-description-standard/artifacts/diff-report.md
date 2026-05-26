# Server 与 RunAPI 差异对账

- Server 接口总数：156，唯一对账键：150
- RunAPI 接口总数：144，唯一对账键：127
- 已存在对账键：125
- RunAPI 缺失对账键：25
- RunAPI 额外/疑似不匹配对账键：2

## RunAPI 缺失接口

| 方法路径 | 模块 | Handler | WebSocket | 初步资源 |
|---|---|---|---|---|
| `DELETE /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | 容灾备份 | `c.deleteBackup` | 否 | DisasterBackup |
| `DELETE /apis/v1/system-settings/:config_key` | 系统设置 | `h.deleteSetting` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `DELETE /apis/v1/users/:username` | 用户管理 | `h.deleteUser` | 否 | disaster-server 用户 Secret |
| `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups` | 容灾备份 | `c.backups` | 否 | DisasterBackup |
| `GET /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | 容灾备份 | `c.backup` | 否 | DisasterBackup |
| `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups` | 容灾备份 | `c.watchBackups` | 是 | DisasterBackup |
| `GET /apis/disasterbackups.testudo.softcdata.com/v1/watch/backups/:name` | 容灾备份 | `c.watchBackup` | 是 | DisasterBackup |
| `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/groups` | 容灾实例 | `h.getInstanceGroups` | 否 | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET /apis/disasterinstances.testudo.softcdata.com/v1/instances/:name/validate-target` | 容灾实例 | `h.validateTarget` | 否 | DisasterInstance、DisasterOperation、DataSync、ResourceSync |
| `GET /apis/v1/:resource/:name/history` | 事件与历史 | `h.listResourceEvents` | 否 | Kubernetes Event |
| `GET /apis/v1/system-settings` | 系统设置 | `h.listSettings` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `GET /apis/v1/system-settings/assets/:config_key` | 系统设置 | `h.getAsset` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `GET /apis/v1/system-settings/public` | 系统设置 | `h.listPublicSettings` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `GET /apis/v1/users` | 用户管理 | `h.listUsers` | 否 | disaster-server 用户 Secret |
| `GET /apis/v1/watch/:resource/:name/events` | 事件与历史 | `h.watchResourceEvents` | 是 | Kubernetes Event |
| `PATCH /apis/disastergroups.testudo.softcdata.com/v1/groups/:name` | 容灾组 | `h.updateGroup` | 否 | DisasterGroup、DisasterOperation、DisasterInstance |
| `PATCH /apis/v1/users/:username/password` | 用户管理 | `h.patchUserPassword` | 否 | disaster-server 用户 Secret |
| `PATCH /apis/v1/users/:username/status` | 用户管理 | `h.patchUserStatus` | 否 | disaster-server 用户 Secret |
| `POST /apis/disasterbackups.testudo.softcdata.com/v1/backups` | 容灾备份 | `c.createBackup` | 否 | DisasterBackup |
| `POST /apis/v1/deletion/check` | 删除检查 | `h.check` | 否 | 被删除目标资源依赖检查 |
| `POST /apis/v1/system-settings` | 系统设置 | `h.createSetting` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `POST /apis/v1/system-settings/assets/:config_key` | 系统设置 | `h.uploadAsset` | 否 | SystemSettings ConfigMap/Secret 及资产 |
| `POST /apis/v1/users` | 用户管理 | `h.createUser` | 否 | disaster-server 用户 Secret |
| `PUT /apis/disasterbackups.testudo.softcdata.com/v1/backups/:name` | 容灾备份 | `c.updateBackup` | 否 | DisasterBackup |
| `PUT /apis/v1/system-settings/:config_key` | 系统设置 | `h.updateSetting` | 否 | SystemSettings ConfigMap/Secret 及资产 |

## RunAPI 额外或疑似路径未标准化接口

| 方法路径 | RunAPI 名称 | 目录 | Target ID | 原路径 |
|---|---|---|---|---|
| `GET /apis/v1/appbackups/:name/history` | 获取指定资源历史记录 | 容灾云平台 / V1 / 历史事件 | `1be82b9e38801001` | `/apis/v1/appbackups/app-backup-1768290527617ms/history` |
| `POST /disasterjobs.testudo.softcdata.com/v1/jobs` | 创建容灾任务 | 容灾云平台 / V1 / 任务 | `3ee6850678c074` | `/disasterjobs.testudo.softcdata.com/v1/jobs` |
