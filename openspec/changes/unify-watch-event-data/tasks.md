## 1. Implementation
- [x] 1.1 Modify `TaskEvent` struct in `internal/apis/event/v1/list.go` to add `Reason` field.
- [x] 1.2 Update `aggregateEvents` in `internal/apis/event/v1/list.go` to populate `Reason` from the latest event.
- [x] 1.3 Update `ConvertToTaskEventDTO` in `internal/apis/event/v1/types.go` to parse `TaskName` from JSON payload.
- [x] 1.4 Update `ConvertToTaskEventDTO` in `internal/apis/event/v1/types.go` to populate `Reason` field.
