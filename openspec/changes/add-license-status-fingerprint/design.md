# Design: License 状态接口返回当前集群指纹

## Context

`disaster-operator/pkg/license.KubernetesStore.Fingerprint(ctx)` 已实现 `k8s-v1` 指纹算法。该算法读取 `kube-system` Namespace UID、License Namespace UID、API Server CA hash 与 `disaster-platform-install-id`，并在安装 ID 不存在时创建该 Secret。

当前 server 的 License 状态接口存在两条状态来源：优先读取 `disaster-platform-license-status` ConfigMap 展示缓存，缓存不存在时基于 License Secret 执行实时评价。现有展示状态不能满足签发 License 前获取当前指纹的需求。

## Goals

- 前端只调用 `GET /apis/v1/platform-license/status` 即可拿到状态与当前指纹。
- 指纹字段用于签发新的 License，不表示已安装 License 中声明的指纹。
- 保持集群创建 License 门禁的可信校验逻辑不变，创建接口继续重新读取 Secret 并重新计算指纹。
- 不新增单独的 fingerprint API。

## Non-Goals

- 不在 server 仓库加入 License 私钥、签发逻辑、签发接口。
- 不改变 License 内容格式与签名算法。
- 不让状态 ConfigMap 成为创建集群门禁的可信输入。

## API Shape

`GET /apis/v1/platform-license/status` 的 `data` 新增字段：

```json
{
  "fingerprint": "sha256:0123456789abcdef...",
  "fingerprintVersion": "k8s-v1",
  "fingerprintRequest": {
    "product": "disaster-platform",
    "fingerprintVersion": "k8s-v1",
    "fingerprint": "sha256:0123456789abcdef...",
    "namespace": "disaster-system",
    "generatedAt": "2026-05-16T10:20:30Z"
  }
}
```

字段含义：

- `fingerprint`：当前管理面集群指纹，用于签发 License。
- `fingerprintVersion`：当前指纹算法版本，固定为 `k8s-v1`。
- `fingerprintRequest`：可直接交给签发侧使用的请求对象，字段与 `disasterctl license fingerprint` 输出保持一致。

## Status Semantics

server 在每次处理 `GET /apis/v1/platform-license/status` 时，完成 Cluster 数量统计并得到展示状态以后，在写出响应以前调用 `KubernetesStore.Fingerprint(ctx)` 生成当前指纹。

当当前指纹生成成功时，server 必须填充 `fingerprint`、`fingerprintVersion` 与 `fingerprintRequest`。该行为不受 License 状态影响，`Free`、`Expired`、`FingerprintMismatch`、`Malformed` 等状态都必须返回当前指纹。

当当前指纹生成失败，且本次状态不属于 License 内容错误分类时，server 必须返回 HTTP 200，`data.state` 固定为 `Unknown`，`data.reason` 固定为 `LicenseEnvironmentInvalid`，`data.message` 写入确定的失败原因，`data.fingerprint` 为空字符串，`data.fingerprintVersion` 固定为 `k8s-v1`，`data.fingerprintRequest` 不返回。

当 License 内容已经被分类为 `Malformed`、`InvalidSignature`、`UnknownKey`、`UnsupportedVersion`、`ProductMismatch`、`NotYetValid`、`Expired` 时，server 必须保留该 License 内容状态和 reason，`data.fingerprint` 为空字符串，`data.fingerprintVersion` 固定为 `k8s-v1`，`data.fingerprintRequest` 不返回。该规则保持现有“License 内容错误优先于部署指纹错误”的语义。

## Implementation Notes

- 在 `StatusDTO` 新增 `Fingerprint`、`FingerprintVersion`、`FingerprintRequest` 字段。
- 新增 `FingerprintRequestDTO`，字段保持 `disasterctl license fingerprint` 输出结构。
- 在 `Service.Status` 返回前统一追加当前指纹，避免只在实时评价路径返回指纹。
- `Service.Install` 继续返回 `StatusDTO`，因此安装响应也会携带当前指纹，前端可统一按同一响应模型处理。
- `KubernetesStore.Fingerprint(ctx)` 会确保 `disaster-platform-install-id` Secret 存在；Chart 的 `disaster-server` 权限需要允许读取该 Secret，并允许在 Secret 不存在时创建。
- operator 状态 ConfigMap 可不新增指纹字段；server 返回的是请求期间生成的当前指纹，不依赖展示缓存。

## Frontend Flow

1. 页面进入 License 管理页后调用 `GET /apis/v1/platform-license/status`。
2. 页面展示 `data.state`、`data.reason`、额度字段与 `data.fingerprint`。
3. 当 `data.fingerprintRequest` 存在时，页面提供复制指纹以及复制完整请求 JSON。
4. 签发侧基于 `data.fingerprintRequest` 生成 License。
5. 页面调用 `POST /apis/v1/platform-license/install` 上传 License。
6. 安装接口返回新的 `StatusDTO` 后，页面刷新展示状态与当前指纹。

## Risks

- 状态接口会在 `disaster-platform-install-id` 不存在时创建该 Secret；这是生成稳定指纹的必要副作用，需要在文档与 RBAC 中明确。
- 如果 server 部署环境无法读取 API Server CA、Namespace UID、安装 ID Secret，接口会返回 `LicenseEnvironmentInvalid`，此时前端不能提交签发请求。
