# Design: Instance Condition Summary API

## 设计目标
- 让 list/detail 都能稳定展示 condition 摘要
- server 只聚合 operator 已有 condition，不另造状态

## 输出形态
- detail：完整 condition 列表 + 高优先级摘要
- list：高优先级摘要
