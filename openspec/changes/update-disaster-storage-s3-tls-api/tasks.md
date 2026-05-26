# Tasks

## 1. Proposal
- [x] 1.1 评审 `addressingStyle` 枚举与默认值
- [x] 1.2 评审 CA Secret 契约与读写脱敏语义

## 2. Server
- [x] 2.1 扩展 storage DTO 与 handler
- [x] 2.2 让 `validateS3Connection` 接收 addressing style 与 CA
- [x] 2.3 补 create/update/patch/edit 语义测试

## 3. Alignment
- [x] 3.1 与 operator 对齐字段名和默认值
- [ ] 3.2 与 web 对齐编辑态交互

## 4. Verification
- [x] 4.1 `openspec validate update-disaster-storage-s3-tls-api --strict`
