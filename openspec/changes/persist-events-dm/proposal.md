# Change: Persist Global Events to Dameng DB

## Why
Kubernetes Events 默认仅保留 1 小时，且存储在 Etcd 中，不适合长期审计和历史回溯。为了支持查询历史灾备任务详情（如“上个月的恢复记录”），需要在 Server 端引入持久化存储。根据用户环境要求，指定使用 **达梦数据库 (Dameng DB)**。

## What Changes
- **Architecture**:
  - `disaster-server` 引入 GORM 及达梦驱动 (`dm`)。
  - 新增 `TaskEvent` 模型定义及 Migration。
  - 新增 `EventSyncer` 组件：启动 K8s Informer 监听 Event，实时写入/更新数据库。
- **Feature**:
  - `listEvents` API 改为查询数据库，支持全量历史检索。
  - 实现 `TaskEvent` 的状态机聚合逻辑（将散乱的 K8s Events 聚合为 DB 中的一条 Task 记录 + Timeline JSON）。
  - **Timeline**: 数据库中存储为 JSON 字段，记录时间轴。

## Impact
- **disaster-server**:
  - 新增依赖: GORM, Dameng Driver。
  - 新增启动参数: 数据库连接串。
  - API 行为变更: 历史数据不再依赖 K8s TTL。

## Database Schema
```sql
CREATE TABLE task_events (
    id VARCHAR(64) PRIMARY KEY, -- TaskID (UID)
    task_name VARCHAR(255),
    task_type VARCHAR(64),
    cluster VARCHAR(64),
    status VARCHAR(32),
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    duration VARCHAR(32),
    trace_id VARCHAR(64),
    user_name VARCHAR(64),
    timeline TEXT, -- JSON Array: [{"time":..., "status":..., "message":...}]
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```
