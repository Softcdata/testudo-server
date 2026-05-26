# Tasks: 实现删除选项

- [ ] 修改 `DeleteCluster` 接收 `uninstall_velero` query 参数。
- [ ] 在执行 Delete 前，如果参数为 true，先 Patch Cluster CR 添加 `testudo.softcdata.com/uninstall-velero: "true"`。
- [ ] 编写测试验证 Annotation 是否正确添加。
