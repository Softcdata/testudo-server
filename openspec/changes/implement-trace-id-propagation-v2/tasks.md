# Task: 实现 V2 链路追踪 (Server Side)

- [ ] **Review Proposal** <!-- id: 0 -->
    - [x] Create proposal `implement-trace-id-propagation-v2/proposal.md` <!-- id: 1 -->
    - [ ] Approve proposal <!-- id: 2 -->

- [ ] **Implementation** <!-- id: 3 -->
    - [ ] Locate `ExecuteAction` handler for Disaster Instance (`internal/apis/disaster_instance/v1/handler_action.go`). <!-- id: 4 -->
    - [ ] Locate Group Action handlers for Disaster Group (`internal/apis/disaster_group/v1/handler.go`). <!-- id: 50 -->
    - [ ] Implement Trace ID extraction from header and injection into `DisasterOperation` annotations for BOTH Instance and Group operations. <!-- id: 5 -->

- [ ] **Validation** <!-- id: 6 -->
    - [ ] Unit test: Verify `DisasterOperation` created via API has the correct annotation. <!-- id: 7 -->
