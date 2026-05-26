# 任务列表：增强容灾组 API

## 1. 规范与文档
- [x] 1.1 明确 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups` 的 `meta.summary.instanceCount` 统计口径。
- [x] 1.2 明确 `GET /apis/disastergroups.testudo.softcdata.com/v1/groups` 的 `meta.summary.abnormalCount` 统计口径。
- [x] 1.3 更新 Swagger/OpenAPI，补充 `abnormalCount` 字段说明与响应示例。
- [x] 1.4 更新 RunAPI/Apipost，补充 `abnormalCount` 字段说明与响应示例。

## 2. 实现
- [x] 2.1 修改 `summarizeDisasterGroupList`，返回 `instanceCount` 与 `abnormalCount`。
- [x] 2.2 保持统计范围为过滤后、分页前的容灾组集合。
- [x] 2.3 增加异常容灾组判定函数，覆盖组级 `reason`、`Error=True` 条件、`Degraded` 展示态、成员 `Failed`、成员 `ConfigError`、成员 `NotFound`、成员非空 `reason`。
- [x] 2.4 确保单独处于 `FailingOver`、`FailingBack` 的容灾组不计入 `abnormalCount`。

## 3. 测试
- [x] 3.1 增加单元测试：正常容灾组 `abnormalCount=0`。
- [x] 3.2 增加单元测试：组级 `reason` 非空时计入异常容灾组。
- [x] 3.3 增加单元测试：成员 `Failed`、`ConfigError`、`NotFound` 时计入异常容灾组。
- [x] 3.4 增加单元测试：单独 `FailingOver`、`FailingBack` 不计入异常容灾组。
- [x] 3.5 增加单元测试：`abnormalCount` 基于过滤后、分页前集合计算。
