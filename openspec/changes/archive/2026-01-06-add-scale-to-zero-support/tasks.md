## 1. API Implementation
- [x] 1.1 Update `internal/apis/app_restore/v1/types.go` to add `ScaleToZeroList []string` field to request structs.
- [x] 1.2 Implement `ScaleToZero(names []string)` function in `internal/resourcemodifier/rule.go`.
- [x] 1.3 Update `internal/apis/app_restore/v1/handler.go` to process `ScaleToZeroList` and append rules.

## 2. Verification
- [x] 2.1 Verify unit tests or compilation (server runs).
- [x] 2.2 Functional verification via E2E test (Ref: disaster-e2e-test).
