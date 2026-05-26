## 1. Domain & Config
- [ ] 1.1 定义配置结构 `internal/config`: 增加 Database 配置 (Driver, DSN)
- [ ] 1.2 引入依赖: GORM 和 达梦驱动
- [ ] 1.3 定义实体 `internal/model/task_event.go`: GORM Model (含 JSON 字段处理)

## 2. Infrastructure (DAO)
- [ ] 2.1 初始化 DB 连接池 `internal/infra/db/client.go`
- [ ] 2.2 实现 TaskEventRepository `internal/infra/repo/task_event_repo.go`
  - [ ] Upsert 能力 (基于 ID)

## 3. Event Syncer (Core Logic)
- [ ] 3.1 创建 Syncer 组件 `internal/syncer/event_syncer.go`
  - [ ] 启动 SharedInformer 监听 `disaster-system` 下的 Events
- [ ] 3.2 实现聚合逻辑 `OnAdd/OnUpdate`
  - [ ] 过滤 Label `testudo.softcdata.com/task-event=true`
  - [ ] 解析 Event Message
  - [ ] **Transaction**:
    - [ ] 查出 DB 中现有记录
    - [ ] Append 新节点到 Timeline
    - [ ] 更新 Status/EndTime
    - [ ] Save 回 DB

## 4. API Adaptation
- [ ] 4.1 修改 `internal/apis/event/v1/list.go`
  - [ ] 移除 K8s Client List 调用
  - [ ] 注入 Repository
  - [ ] 改为调用 `repo.List(filterOptions)`

## 5. Verification
- [ ] 5.1 验证 DB 连接 (Mock/Docker)
- [ ] 5.2 验证 Event 写入后 Timeline 的连续性
