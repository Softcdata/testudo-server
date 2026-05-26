## ADDED Requirements

### Requirement: Durable Event Storage
The System MUST persist all structural Task Events into a relational database (Dameng DB) to support long-term auditing and history retrieval beyond the Kubernetes Event TTL (1 hour).

#### Scenario: Historical Query
- **WHEN** a user queries events from 30 days ago
- **THEN** the system retrieves the record from the database
- **AND** returns the complete timeline of that historical task

### Requirement: Event Synchronization
The Server MUST continuously synchronize volatile Kubernetes Events into the database.

#### Scenario: Sync Timeline
- **WHEN** a new `InProgress` event is emitted by the Operator
- **THEN** the Syncer captures it
- **AND** appends a entry to the `timeline` JSON field of the corresponding Task record in DB
- **AND** updates the task's `status` and `updated_at` fields
