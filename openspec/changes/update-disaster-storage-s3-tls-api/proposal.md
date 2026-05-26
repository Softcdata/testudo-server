# Change: 为存储 API 增加 S3 addressing style 与 CA Secret 契约

## Why
当前 `disaster_storage` API 只能表达 endpoint、bucket、accessKey、secretKey、region，不能表达：
- addressing style
- 自定义 CA 引用

这导致 server 的连接校验无法与 operator/Velero runtime 保持同构，条目 5 的兼容性问题无法从 API 层面闭环。

## What Changes

### 1. Storage API 增加显式 addressing style 字段
- 支持：
  - `PathStyle`
  - `VirtualHostedStyle`
- 默认值与当前行为兼容：`PathStyle`

### 2. Storage API 增加 CA Secret 契约
- 写接口支持上传或引用自定义 CA
- 读接口只返回非敏感引用状态，不回显证书原文

### 3. validateS3Connection 必须使用与 operator 一致的参数集合
- 统一校验：
  - endpoint
  - region
  - addressing style
  - TLS trust source

### 4. 保持 patch/update 现有 endpoint 语义
- 本 proposal 不主动改写现有 patch/update 的 endpoint 可变规则
- 但在设计文档中要求把 addressing style 与 CA 的编辑语义写清楚

## Non-Goals
- 不在 server 侧承诺所有 S3 兼容实现都支持。
- 不把证书明文直接回显给客户端。

## Impact
- Affected specs:
  - `disaster_storage`
- Affected code:
  - `internal/apis/disaster_storage/v1/types.go`
  - `internal/apis/disaster_storage/v1/handler.go`
  - `internal/storage/*`
- Cross-repo impact:
  - `disaster-operator`：消费一致的字段集
  - `cluster-disaster-web`：新增 addressing style 与 CA 交互

## Risks
- 若 update 语义不区分“未修改 CA”与“显式替换 CA”，编辑体验会混乱。
