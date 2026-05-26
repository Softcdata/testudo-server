# AppBackup API 接口文档

## 1. 资源概述
`AppBackup` 资源用于定义应用级的定时备份计划。它基于 Velero 的备份能力，支持按命名空间、资源类型、标签选择器等维度进行备份，并支持定时调度。

## 2. 接口定义

### 2.1 创建备份计划 (Create AppBackup)

创建一个新的应用备份计划。

- **URL**: `/appbackups.testudo.softcdata.com/v1/appbackups`
- **Method**: `POST`
- **Content-Type**: `application/json`

#### 请求参数 (Request Body)

请求体必须符合 `AppBackup` CRD 的定义。

| 字段 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `metadata.name` | string | 是 | 备份计划名称，需符合 K8s 命名规范。 |
| `metadata.namespace` | string | 否 | 命名空间，默认为系统配置的命名空间（通常为 `disaster-system`）。 |
| `spec.cluster` | string | 是 | 目标集群名称。 |
| `spec.schedule` | string | 否 | 调度策略：<br>1. **定时备份**：填写 Cron 表达式（如 `0 0 * * *`）。<br>2. **单次备份**：留空（空字符串），创建后立即执行一次备份。 |
| `spec.paused` | boolean | 否 | 是否暂停调度。 |
| `spec.useOwnerReferencesInBackup` | boolean | 否 | 是否在备份中使用 OwnerReferences。 |
| `spec.skipImmediately` | boolean | 否 | 是否跳过立即执行。 |
| `spec.template` | object | 是 | 备份模板，包含具体的 Velero 备份配置。 |

**`spec.template` 核心参数 (Velero BackupSpec):**

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `includedNamespaces` | []string | 要备份的命名空间列表。支持 `*`。 |
| `excludedNamespaces` | []string | 要排除的命名空间列表。 |
| `includedResources` | []string | 要备份的资源类型（如 `deployments`）。 |
| `labelSelector` | object | 标签选择器 `{ "matchLabels": { ... } }`。 |
| `storageLocation` | string | 备份存储位置名称。 |
| `ttl` | string | 备份保留时间（如 `720h`）。 |
| `snapshotVolumes` | boolean | 是否对 PV 进行快照。 |
| `defaultVolumesToFsBackup` | boolean | 是否使用文件系统备份 (Restic/Kopia)。 |
| `hooks` | object | 备份前后的钩子配置。 |

#### 请求示例 (Request Example)

```json
{
  "metadata": {
    "name": "daily-mysql-backup",
    "namespace": "disaster-system"
  },
  "spec": {
    "cluster": "prod-cluster",
    "schedule": "0 3 * * *",
    "template": {
      "includedNamespaces": [
        "mysql-prod"
      ],
      "ttl": "720h",
      "storageLocation": "minio-default",
      "snapshotVolumes": true,
      "defaultVolumesToFsBackup": false,
      "hooks": {
        "resources": [
          {
            "name": "mysql-fsfreeze",
            "includedNamespaces": ["mysql-prod"],
            "includedResources": ["pods"],
            "labelSelector": {
              "matchLabels": {
                "app": "mysql"
              }
            },
            "pre": [
              {
                "exec": {
                  "container": "mysql",
                  "command": ["/bin/sh", "-c", "mysql -e 'FLUSH TABLES WITH READ LOCK;'"],
                  "onError": "Fail",
                  "timeout": "30s"
                }
              }
            ],
            "post": [
              {
                "exec": {
                  "container": "mysql",
                  "command": ["/bin/sh", "-c", "mysql -e 'UNLOCK TABLES;'"],
                  "onError": "Continue",
                  "timeout": "30s"
                }
              }
            ]
          }
        ]
      }
    }
  }
}
```

#### 响应 (Response)

- **Status**: `201 Created`
- **Body**: 返回创建成功的 `AppBackup` 对象完整信息。

```json
{
    "metadata": {
        "name": "daily-mysql-backup",
        "namespace": "disaster-system",
        "uid": "...",
        "resourceVersion": "...",
        "creationTimestamp": "2023-10-27T10:00:00Z",
        "generation": 1
    },
    "spec": {
        ...
    },
    "status": {
        "status": "Pending",
        "scheduleStatus": {
            "phase": "Enabled",
            "lastBackup": "2023-10-27T10:00:00Z"
        },
        "backupStatus": {
            "phase": "Completed",
            "completionTimestamp": "2023-10-27T10:05:00Z"
        }
    }
}
```


