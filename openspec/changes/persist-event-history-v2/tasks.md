## 1. Spec 与建模
- [ ] 1.1 新建 server 侧 `global-events` capability 增量规范
- [ ] 1.2 明确 durable history、execution identity、owner snapshot、timeline node 的契约
- [ ] 1.3 明确与 active change `persist-events-dm` 的收敛策略

## 2. Server 基础设施
- [ ] 2.1 为 history store 增加抽象接口与配置入口
- [ ] 2.2 设计可替换的存储驱动接入点，不在首期写死具体数据库
- [ ] 2.3 明确 retention、迁移与启动失败语义

## 3. Event Projector
- [ ] 3.1 新增结构化任务事件 projector / syncer 组件
- [ ] 3.2 复用当前复合聚合键计算 execution identity
- [ ] 3.3 增加基于 `sourceEventUID` 的 timeline node 幂等保护
- [ ] 3.4 明确 projector 与实时 watch 的隔离边界

## 4. API 适配
- [ ] 4.1 将 `/apis/v1/events` 切换为读取 durable history
- [ ] 4.2 将 `/:resource/:name/history` 切换为读取 durable history
- [ ] 4.3 增加 `executionId` 等兼容字段，避免继续混淆资源身份与执行身份
- [ ] 4.4 保持 `/watch/events` 与 `/watch/:resource/:name/events` 继续消费实时 Kubernetes Events

## 5. Cross-Repo 协调
- [ ] 5.1 与 `disaster-operator` 的 companion proposal 对齐 durable history 依赖的事件字段与终态完整性
- [ ] 5.2 与 `cluster-disaster-web` 对齐 `executionId` 的展示与跳转语义

## 6. 验证
- [ ] 6.1 补充 server 单测：聚合、幂等、executionId 区分
- [ ] 6.2 补充集成测试：history store 故障不影响实时 watch
- [ ] 6.3 补充联调步骤：验证跨 TTL 历史查询与前端审计页兼容
