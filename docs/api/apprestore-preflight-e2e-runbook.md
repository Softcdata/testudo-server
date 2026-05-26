# AppRestore Preflight 接口 E2E 测试手册

## 1. 目的
本文档用于在本机环境对以下接口执行真实端到端验证：

- `POST /apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate`

验证目标：

1. 在目标集群不存在 `requiredBSL=<storageRepository>-<sourceCluster>` 时，接口可触发 Operator 初始化。
2. 接口按等待窗口轮询，最终在 BSL 就绪后返回成功。
3. 接口响应 `meta` 字段符合约定。
4. 二次调用走“已初始化快速路径”。

## 2. 测试范围与原则

### 2.1 范围
本手册覆盖：

1. 路由可达性检查
2. 无 BSL 场景触发初始化并等待成功
3. 二次调用快速成功
4. 测试资源清理

### 2.2 安全原则
执行过程中必须遵守：

1. 不删除现有业务 BSL。
2. 使用唯一 `sourceCluster` 构造全新 `requiredBSL` 名称，避免碰撞。
3. 仅删除本次测试创建的临时资源。

## 3. 前置条件
在执行前确认：

1. 管理集群可访问，`kubectl` 可用。
2. `disaster-server` 与 `disaster-operator` 已部署并运行。
3. 至少存在一个 `Ready` 状态的 `Cluster`。
4. 至少存在一个 `Available` 状态的 `StorageRepository`。

## 4. 环境变量初始化
在终端执行：

```bash
cd /home/chenxi/YS/disaster-server

export MGMT_NS=disaster-system
export VELERO_NS=velero
export TS=$(date +%Y%m%d%H%M%S)
export E2E_ID=e2e-preflight-$TS
```

## 5. 环境与服务可达性确认

### 5.1 当前上下文确认
```bash
kubectl config current-context
kubectl get ns | head
```

预期：

1. 输出正确测试集群上下文。
2. 命名空间列表可正常返回。

### 5.2 平台组件状态
```bash
kubectl -n "$MGMT_NS" get pods | rg -i 'disaster-server|operator|velero'
kubectl -n "$MGMT_NS" get svc | rg -i 'disaster-server'
```

预期：

1. `disaster-server`、`disaster-operator` 相关 Pod 为 `Running`。
2. 存在 `disaster-server` Service。

## 6. 接口连通性与路由存在性检查

### 6.1 建立本地端口转发（新终端前台运行）
```bash
kubectl -n "$MGMT_NS" port-forward svc/disaster-server 18080:80
```

### 6.2 设置请求地址
在另一个终端执行：

```bash
export API_BASE=http://127.0.0.1:18080
# 若环境启用鉴权，请取消下一行注释并设置 token
# export AUTH_HEADER="Authorization: Bearer <your_token>"
```

### 6.3 路由探测（只验证是否存在）
```bash
curl -sS -o /tmp/preflight_probe.json -w '%{http_code}\n' \
  -X POST "$API_BASE/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate" \
  -H 'Content-Type: application/json' \
  ${AUTH_HEADER:+-H "$AUTH_HEADER"} \
  -d '{"backupSource":"__probe__","targetCluster":"__probe__","waitSeconds":1}'

cat /tmp/preflight_probe.json
```

预期：

1. HTTP 状态码不是 `404`（可为 `400` 或 `404` 业务资源错误）。
2. 返回统一 Envelope JSON。

## 7. 选择测试对象并构造唯一 requiredBSL

### 7.1 选择目标集群
```bash
export TARGET_CLUSTER=$(kubectl get clusters.testudo.softcdata.com \
  -o jsonpath='{range .items[?(@.status.status=="Ready")]}{.metadata.name}{"\n"}{end}' | head -n1)
echo "$TARGET_CLUSTER"
```

### 7.2 选择存储仓库
```bash
export STORAGE_REPO=$(kubectl -n "$MGMT_NS" get storagerepositories.testudo.softcdata.com \
  -o jsonpath='{range .items[?(@.status.status=="Available")]}{.metadata.name}{"\n"}{end}' | head -n1)
echo "$STORAGE_REPO"
```

### 7.3 构造唯一源集群与 requiredBSL
```bash
export SOURCE_CLUSTER="e2e-src-$TS"
export REQUIRED_BSL="${STORAGE_REPO}-${SOURCE_CLUSTER}"
echo "$SOURCE_CLUSTER"
echo "$REQUIRED_BSL"
```

预期：

1. `TARGET_CLUSTER` 非空。
2. `STORAGE_REPO` 非空。
3. `REQUIRED_BSL` 为本次唯一名称。

## 8. 创建测试 AppBackup 夹具
创建仅用于 preflight 的临时 `AppBackup`：

```bash
cat <<EOF | kubectl apply -f -
apiVersion: testudo.softcdata.com/v1
kind: AppBackup
metadata:
  name: ${E2E_ID}
  namespace: ${MGMT_NS}
spec:
  cluster: ${SOURCE_CLUSTER}
  schedule: "@daily"
  paused: true
  template:
    storageLocation: ${STORAGE_REPO}
    includedNamespaces:
    - default
EOF
```

校验字段：

```bash
kubectl -n "$MGMT_NS" get appbackups.testudo.softcdata.com "$E2E_ID" \
  -o jsonpath='{.spec.cluster}{"|"}{.spec.template.storageLocation}{"\n"}'
```

预期：

1. 资源创建成功。
2. 输出 `${SOURCE_CLUSTER}|${STORAGE_REPO}`。

## 9. 调用前确认目标集群无 requiredBSL

### 9.1 生成目标集群 kubeconfig
```bash
kubectl get clusters.testudo.softcdata.com "$TARGET_CLUSTER" \
  -o jsonpath='{.spec.kubeConfig}' > /tmp/${E2E_ID}.kubeconfig.b64

if [ -s /tmp/${E2E_ID}.kubeconfig.b64 ]; then
  base64 -d /tmp/${E2E_ID}.kubeconfig.b64 > /tmp/${E2E_ID}.target.kubeconfig
else
  ENDPOINT=$(kubectl get clusters.testudo.softcdata.com "$TARGET_CLUSTER" -o jsonpath='{.spec.endpoint}')
  TOKEN=$(kubectl get clusters.testudo.softcdata.com "$TARGET_CLUSTER" -o jsonpath='{.spec.token}')
  if [[ "$TOKEN" != eyJ* ]]; then
    TOKEN_DECODED=$(echo "$TOKEN" | base64 -d 2>/dev/null || true)
    [ -n "$TOKEN_DECODED" ] && TOKEN="$TOKEN_DECODED"
  fi
  cat > /tmp/${E2E_ID}.target.kubeconfig <<EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: ${ENDPOINT}
    insecure-skip-tls-verify: true
  name: target
contexts:
- context:
    cluster: target
    user: target
  name: target
current-context: target
users:
- name: target
  user:
    token: ${TOKEN}
EOF
fi
```

### 9.2 验证不存在
```bash
kubectl --kubeconfig /tmp/${E2E_ID}.target.kubeconfig -n "$VELERO_NS" \
  get backupstoragelocations.velero.io "$REQUIRED_BSL"
```

预期：

1. 返回 `NotFound`。

## 10. 核心 E2E：首次调用（无 BSL -> 初始化 -> 等待）
执行请求并记录耗时：

```bash
export T0=$(date +%s)

curl -sS -X POST "$API_BASE/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate" \
  -H 'Content-Type: application/json' \
  ${AUTH_HEADER:+-H "$AUTH_HEADER"} \
  -d "{\"backupSource\":\"${E2E_ID}\",\"targetCluster\":\"${TARGET_CLUSTER}\",\"waitSeconds\":60}" \
  | tee /tmp/${E2E_ID}.preflight.first.json

export T1=$(date +%s)
echo "elapsed_first=$((T1-T0))s"
```

检查返回体：

```bash
cat /tmp/${E2E_ID}.preflight.first.json
# 若安装 jq:
# jq '.' /tmp/${E2E_ID}.preflight.first.json
```

预期：

1. `code=0`
2. `data=true`
3. `meta.required_bsl == $REQUIRED_BSL`
4. `meta.source_cluster == $SOURCE_CLUSTER`
5. `meta.target_cluster == $TARGET_CLUSTER`
6. `meta.storage_repository == $STORAGE_REPO`
7. `meta.state` 为 `Available`（或最终可用态）

## 11. 调用后状态验证

### 11.1 目标集群 BSL 状态
```bash
kubectl --kubeconfig /tmp/${E2E_ID}.target.kubeconfig -n "$VELERO_NS" \
  get backupstoragelocations.velero.io "$REQUIRED_BSL" \
  -o jsonpath='{.metadata.name}{"|"}{.status.phase}{"|"}{.metadata.creationTimestamp}{"\n"}'
```

预期：

1. BSL 存在。
2. `phase=Available`。

### 11.2 注解消费状态
```bash
kubectl get clusters.testudo.softcdata.com "$TARGET_CLUSTER" \
  -o jsonpath='{.metadata.annotations.disaster\.wuxs\.vip/ensure-storage}{"|"}{.metadata.annotations.disaster\.wuxs\.vip/ensure-storage-source-cluster}{"\n"}'
```

预期：

1. 两个注解为空（已被 Operator 消费移除）。

## 12. 二次调用（快速路径）
```bash
export T2=$(date +%s)

curl -sS -X POST "$API_BASE/apis/apprestores.testudo.softcdata.com/v1/apprestores/preflight/validate" \
  -H 'Content-Type: application/json' \
  ${AUTH_HEADER:+-H "$AUTH_HEADER"} \
  -d "{\"backupSource\":\"${E2E_ID}\",\"targetCluster\":\"${TARGET_CLUSTER}\",\"waitSeconds\":60}" \
  | tee /tmp/${E2E_ID}.preflight.second.json

export T3=$(date +%s)
echo "elapsed_second=$((T3-T2))s"
```

预期：

1. `code=0` 且 `data=true`。
2. `elapsed_second` 明显小于首次调用。

## 13. 清理步骤

### 13.1 删除测试 AppBackup
```bash
kubectl -n "$MGMT_NS" delete appbackups.testudo.softcdata.com "$E2E_ID" --ignore-not-found
```

### 13.2 删除测试创建的 BSL
```bash
kubectl --kubeconfig /tmp/${E2E_ID}.target.kubeconfig -n "$VELERO_NS" \
  delete backupstoragelocations.velero.io "$REQUIRED_BSL" --ignore-not-found
```

### 13.3 删除临时文件
```bash
rm -f /tmp/${E2E_ID}.kubeconfig.b64 /tmp/${E2E_ID}.target.kubeconfig \
      /tmp/${E2E_ID}.preflight.first.json /tmp/${E2E_ID}.preflight.second.json \
      /tmp/preflight_probe.json
```

### 13.4 复核无残留
```bash
kubectl -n "$MGMT_NS" get appbackups.testudo.softcdata.com | rg "$E2E_ID" || true
kubectl get clusters.testudo.softcdata.com "$TARGET_CLUSTER" \
  -o jsonpath='{.metadata.annotations.disaster\.wuxs\.vip/ensure-storage}{"|"}{.metadata.annotations.disaster\.wuxs\.vip/ensure-storage-source-cluster}{"\n"}'
```

预期：

1. 无测试 AppBackup。
2. 无 `ensure-storage*` 注解残留。

## 14. 通过判定标准
本次 E2E 通过需同时满足：

1. 调用前 `requiredBSL` 不存在。
2. 首次 preflight 返回 `data=true`，且 `meta` 字段完整正确。
3. 调用后 `requiredBSL` 在目标集群存在且 `phase=Available`。
4. 二次调用快速成功。
5. 清理后无测试残留资源。

## 15. 常见失败与处理建议

1. 路由 404：确认 server 是否部署新版本，或路由前缀是否正确。
2. 持续超时：检查 operator 日志、目标集群 Velero 控制器状态、对象存储连通性。
3. `failed to get client for cluster`：检查 `Cluster` CR 中 kubeconfig/token+endpoint 配置。
4. 注解不消费：检查 operator 是否包含 `ensure-storage-source-cluster` 逻辑版本。
