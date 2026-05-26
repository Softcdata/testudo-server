# 变更提案: 添加 Ingress 类名称映射支持

## Why (为什么)

在跨集群灾难恢复场景中,源集群和目标集群可能使用不同的 Ingress Controller,例如:
- 源集群使用 **Nginx Ingress Controller** (`ingressClassName: nginx`)
- 目标集群使用 **Traefik** (`ingressClassName: traefik`)

当前系统已支持:
- ✅ **StorageClass 映射** (`SCMapping`): 通过 `scMapping` 参数映射 PVC 的 StorageClass
- ✅ **清理 PVC VolumeName** (`CleanVolumes`): 通过 `cleanVolumes` 参数移除 PVC 的 volumeName 绑定

但**缺少 Ingress 类名称映射**,导致:
1. 恢复后的 Ingress 资源使用源集群的 `ingressClassName`,在目标集群中无效
2. 用户需要手动修改所有 Ingress 资源,增加运维负担
3. 无法实现全自动的跨集群恢复

## What Changes (变更内容)

### 1. 添加 IngressClass 映射功能

#### 1.1 新增 ResourceModifier 规则生成函数
**文件**: `internal/resourcemodifier/rule.go`

```go
// IngressClassMapping generates rules to map Ingress class names during restore
// Example: {"nginx": "traefik"} will replace ingressClassName from "nginx" to "traefik"
func IngressClassMapping(mapping map[string]string) []dapisv1.ResourceModifierRule {
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

#### 1.2 更新 API 请求结构
**文件**: `internal/apis/app_restore/v1/types.go`

在 `CreateAppRestoreRequest` 和 `UpdateAppRestoreRequest` 中添加:
```go
type CreateAppRestoreRequest struct {
    // ... 现有字段 ...
    SCMapping           map[string]string `json:"scMapping,omitempty"`
    IngressClassMapping map[string]string `json:"ingressClassMapping,omitempty"` // 新增
}
```

#### 1.3 应用 IngressClass 映射规则
**文件**: `internal/apis/app_restore/v1/handler.go`

在 `createAppRestore` 方法中:
```go
if req.SCMapping != nil && len(req.SCMapping) > 0 {
    body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.SCMapping(req.SCMapping)...)
}
// 新增: 应用 IngressClass 映射
if req.IngressClassMapping != nil && len(req.IngressClassMapping) > 0 {
    body.Spec.ResourceModifierRules = append(body.Spec.ResourceModifierRules, resourcemodifier.IngressClassMapping(req.IngressClassMapping)...)
}
```

### 2. 更新文档

#### 2.1 更新 README
**文件**: `internal/resourcemodifier/readme.md`

添加 IngressClass 映射示例:
```markdown
## 示例：替换 Ingress Class

在跨集群恢复时,源集群使用 Nginx Ingress,目标集群使用 Traefik:

\`\`\`go
func IngressClassMapping(mapping map[string]string) []dapisv1.ResourceModifierRule {
    // mapping: {"nginx": "traefik"}
    // 将所有 ingressClassName: nginx 替换为 traefik
}
\`\`\`

API 请求示例:
\`\`\`json
{
  "name": "restore-prod",
  "backupSource": "backup-prod",
  "cluster": "target-cluster",
  "backupName": "backup-20250101",
  "ingressClassMapping": {
    "nginx": "traefik",
    "alb": "haproxy"
  }
}
\`\`\`
```

## Impact (影响范围)

### 受影响的项目
- **disaster-server**: API 层增加 `ingressClassMapping` 参数支持

### 受影响的文件
- `internal/resourcemodifier/rule.go` - 新增 `IngressClassMapping` 函数
- `internal/resourcemodifier/readme.md` - 添加文档
- `internal/apis/app_restore/v1/types.go` - 添加 API 字段
- `internal/apis/app_restore/v1/handler.go` - 应用映射规则

### 破坏性变更
- **无破坏性变更**: `ingressClassMapping` 是可选参数,不影响现有 API 调用

## 技术细节

### Ingress 资源结构

Kubernetes Ingress 资源使用 `spec.ingressClassName` 字段指定 Ingress Controller:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
spec:
  ingressClassName: nginx  # 需要映射的字段
  rules:
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: example-service
                port:
                  number: 80
```

### JSON Patch 操作流程

1. **Test 操作**: 验证当前 `ingressClassName` 是否为 `nginx`
2. **Replace 操作**: 如果匹配,替换为 `traefik`

```json
[
  {
    "op": "test",
    "path": "/spec/ingressClassName",
    "value": "nginx"
  },
  {
    "op": "replace",
    "path": "/spec/ingressClassName",
    "value": "traefik"
  }
]
```

### API 请求示例

```bash
curl -X POST http://disaster-server/apis/apprestores.testudo.softcdata.com/v1/apprestores \
  -H "Content-Type: application/json" \
  -d '{
    "name": "restore-prod-to-dr",
    "backupSource": "backup-prod",
    "cluster": "dr-cluster",
    "backupName": "prod-backup-20250101",
    "cleanVolumes": true,
    "scMapping": {
      "gp2": "managed-nfs-storage"
    },
    "ingressClassMapping": {
      "nginx": "traefik",
      "alb": "haproxy"
    }
  }'
```

## 风险与缓解

### 风险1: Ingress Controller 功能差异
- **场景**: 不同 Ingress Controller 支持的 Annotation 可能不同
- **缓解**: 
  - 仅映射 `ingressClassName`,不修改 Annotations
  - 用户需要在恢复后手动调整特定 Controller 的 Annotations

### 风险2: 多个 IngressClass 映射冲突
- **场景**: 用户配置了重复的映射规则
- **缓解**: 
  - Map 结构天然去重
  - 文档中说明映射规则的执行顺序

## 测试计划

1. **单元测试**: 验证 `IngressClassMapping` 函数生成正确的规则
2. **集成测试**: 创建包含 Ingress 的备份,恢复时应用映射
3. **E2E 测试**: 在真实环境中验证跨 Ingress Controller 的恢复

## 示例场景

### 场景1: Nginx → Traefik 迁移

**源集群 Ingress**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: nginx
  rules:
    - host: app.example.com
```

**API 请求**:
```json
{
  "ingressClassMapping": {
    "nginx": "traefik"
  }
}
```

**恢复后 Ingress**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  ingressClassName: traefik  # ✅ 已映射
  rules:
    - host: app.example.com
```

### 场景2: 多 IngressClass 环境

**API 请求**:
```json
{
  "ingressClassMapping": {
    "nginx": "traefik",
    "alb": "haproxy",
    "istio": "kong"
  }
}
```

## 与现有功能的对比

| 功能 | 参数名 | 作用 | 资源类型 |
|------|--------|------|---------|
| **清理 PVC 绑定** | `cleanVolumes` | 移除 `spec.volumeName` | PersistentVolumeClaim |
| **StorageClass 映射** | `scMapping` | 替换 `spec.storageClassName` | PersistentVolumeClaim |
| **IngressClass 映射** | `ingressClassMapping` | 替换 `spec.ingressClassName` | Ingress (新增) |

## 后续扩展

未来可以考虑添加更多资源映射:
- **ServiceType 映射**: LoadBalancer → ClusterIP
- **镜像仓库映射**: `registry-a.com` → `registry-b.com`
- **NodeSelector 映射**: 跨集群节点标签不同
