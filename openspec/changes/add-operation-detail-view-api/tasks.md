## 1. Proposal
- [ ] 1.1 锁定 P1-P4 分期与现有 active change 的依赖关系
- [ ] 1.2 锁定 history item 与 detail/watch DTO 的共享字段口径

## 2. History / Detail
- [x] 2.1 为 instance history 增加 `operationName`、`operationUID`、`hasDetail`
- [x] 2.2 为 group history 增加同名字段
- [x] 2.3 新增共享 `OperationDetailDTO` 与 converter
- [x] 2.4 新增 instance operation detail route
- [x] 2.5 新增 group operation detail route
- [x] 2.6 补 owner 归属校验与 404 场景测试
- [x] 2.7 补 detail DTO 字段级回归测试，覆盖 `steps[]`、`autoCancel`、`groupStatus`、`roleStatus`

## 3. Watch / UI Alignment
- [x] 3.1 新增 instance single-operation watch route
- [ ] 3.2 复用现有 group single-operation watch DTO，不引入字段漂移
- [ ] 3.3 与 web 对齐 P3 语义：history row 点击后调 P2，运行中记录再追加 watch
- [ ] 3.4 补 watch DTO 一致性测试，确保 instance/group 单操作 watch 字段同名

## 4. Follow-up Boundary
- [ ] 4.1 把 P1 的 drill detail 快速收益写入设计说明
- [ ] 4.2 把 P4 对 `persist-event-history-v2` 与 `add-v2-event-emission-coverage` 的依赖写入设计说明

## 5. Verification
- [ ] 5.1 `openspec validate add-operation-detail-view-api --strict`
