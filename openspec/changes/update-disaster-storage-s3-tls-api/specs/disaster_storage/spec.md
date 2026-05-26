## ADDED Requirements

### Requirement: 存储 API 必须支持显式 S3 addressing style
系统必须 (MUST) 允许在存储创建和更新接口中声明 S3 addressing style。

#### Scenario: 创建存储时声明 VirtualHostedStyle
- **When** 客户端创建一个 `StorageRepository` 并显式提交 `addressingStyle=VirtualHostedStyle`
- **Then** Server 必须保存该值并传递给后续校验链路

### Requirement: 存储 API 必须支持自定义 CA 的非敏感契约
系统必须 (MUST) 允许在存储写接口中配置自定义 CA，并在读接口中只返回非敏感状态。

#### Scenario: 查询存储时不回显 CA 原文
- **Given** 一个 `StorageRepository` 已配置自定义 CA
- **When** 客户端查询该存储详情
- **Then** 返回结果可以表明已配置 CA 或 CA 引用
- **And** 不得返回证书原文

### Requirement: S3 校验接口必须使用完整运行参数
系统必须 (MUST) 使用 endpoint、region、addressing style 和 TLS trust source 的完整参数集合执行 S3 连接校验。

#### Scenario: 校验接口使用 addressing style 与 CA
- **Given** 一个存储请求同时声明了 `addressingStyle` 和自定义 CA
- **When** Server 执行连接校验
- **Then** 校验链路必须同时使用这两个参数
- **And** 不得退回到旧的默认硬编码路径
