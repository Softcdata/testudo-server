## ADDED Requirements

### Requirement: Scale-to-Zero during Restore
The system MUST allow users to specify a list of workloads that should be scaled to zero immediately upon restoration.

#### Scenario: Warm Standby Restore
- **GIVEN** a user creates an `AppRestore` request
- **WHEN** they provide `scaleToZeroList: ["backend-service", "worker"]`
- **THEN** the system MUST generate Resource Modifier rules
- **AND** the restored Deployment `backend-service` MUST have 0 replicas
- **AND** the restored StatefulSet/Deployment `worker` MUST have 0 replicas
