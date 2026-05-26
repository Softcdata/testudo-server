## 1. API Implementation
- [x] 1.1 Update `internal/apis/app_restore/v1/types.go` to add `StandbyList []string` field.
- [x] 1.2 Implement `StandbyReplacement(names []string)` in `internal/resourcemodifier/rule.go`.
    - Replace Image with `busybox:latest`
    - Replace Command with `sleep infinity`
    - Remove Probes
- [x] 1.3 Update `internal/apis/app_restore/v1/handler.go` to process `StandbyList`.

## 2. Verification
- [x] 2.1 Verify via E2E test (update basic_test.go to use StandbyList for StatefulSet).
