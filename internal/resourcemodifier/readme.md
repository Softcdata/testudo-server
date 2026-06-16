# Resource Modifier Rules

`ResourceModifierRule` 用于在 Velero 恢复过程中动态修改 Kubernetes 资源。这允许我们在资源被应用到集群之前，对其 YAML/JSON 定义进行补丁操作。

## 作用

在灾难恢复或跨集群迁移场景中，源集群的某些资源配置可能不适用于目标集群。例如：
- **StorageClass 变更**：源集群使用 `gp2`，目标集群使用 `managed-nfs-storage`。
- **PVC 绑定清理**：源集群的 PVC 绑定了特定的 PV（通过 `volumeName`），在目标集群恢复时需要清除该绑定，以便动态创建新的 PV。
- **镜像仓库替换**：将私有镜像仓库地址从 `registry-a` 替换为 `registry-b`。
- **Service 类型变更**：将 `LoadBalancer` 修改为 `ClusterIP`。

## 结构

一个规则由两部分组成：

1.  **Conditions (匹配条件)**：决定哪些资源会被修改。
    *   `GroupResource`: 资源类型（如 `persistentvolumeclaims`, `deployments.apps`）。
    *   `ResourceNameRegex`: 资源名称的正则表达式（可选）。
    *   `Namespaces`: 匹配的命名空间列表（可选）。
    *   `LabelSelector`: 标签选择器（可选）。

2.  **Patches (补丁操作)**：对匹配到的资源执行 JSON Patch 操作。
    *   `Operation`: 操作类型 (`add`, `remove`, `replace`, `move`, `copy`, `test`)。
    *   `Path`: JSON Path 路径（如 `/spec/storageClassName`）。
    *   `Value`: 新值（对于 `add` 和 `replace` 操作）。

## 示例：清理 PVC VolumeName

在跨集群恢复时，PVC 中记录的 `volumeName` 指向源集群的 PV，这在目标集群中通常是无效的。我们需要清空它，让 StorageClass 动态制备新的 PV。这里必须使用幂等 `add` 空值，不能使用 `remove`，否则当备份中的 PVC 不包含该字段时 JSON Patch 会失败。

```go
func CleanVolume() dapisv1.ResourceModifierRule {
    return dapisv1.ResourceModifierRule{
        Conditions: dapisv1.Conditions{
            GroupResource: "persistentvolumeclaims", // 匹配所有 PVC
        },
        Patches: []dapisv1.JSONPatch{
            {
                Operation: "add",               // 幂等清空
                Path:      "/spec/volumeName",  // 清空 volumeName 字段
                Value:     "",
            },
        },
    }
}
```

## 示例：替换 StorageClass

```yaml
conditions:
  groupResource: persistentvolumeclaims
patches:
  - operation: replace
    path: "/spec/storageClassName"
    value: "new-storage-class"
```

## 示例：替换 Ingress Class

在跨集群恢复时,源集群使用 Nginx Ingress Controller,目标集群使用 Traefik:

```go
func IngressClassMapping(mapping map[string]string) []dapisv1.ResourceModifierRule {
    // mapping: {"nginx": "traefik"}
    // 将所有 ingressClassName: nginx 替换为 traefik
    var rules []dapisv1.ResourceModifierRule
    for oldClass, newClass := range mapping {
        rules = append(rules, dapisv1.ResourceModifierRule{
            Conditions: dapisv1.Conditions{
                GroupResource: "ingresses.networking.k8s.io",
            },
            Patches: []dapisv1.JSONPatch{
                {
                    Operation: "test",
                    Path:      "/spec/ingressClassName",
                    Value:     oldClass,
                },
                {
                    Operation: "replace",
                    Path:      "/spec/ingressClassName",
                    Value:     newClass,
                },
            },
        })
    }
    return rules
}
```

### API 请求示例

```json
{
  "name": "restore-prod",
  "backupSource": "backup-prod",
  "cluster": "target-cluster",
  "backupName": "backup-20250101",
  "cleanVolumes": true,
  "scMapping": {
    "gp2": "managed-nfs-storage"
  },
  "ingressClassMapping": {
    "nginx": "traefik",
    "alb": "haproxy"
  }
}
```

### 使用场景

- **跨云迁移**: AWS ALB → GCP Ingress
- **Ingress Controller 升级**: Nginx → Traefik
- **多集群环境**: 不同集群使用不同的 Ingress Controller
