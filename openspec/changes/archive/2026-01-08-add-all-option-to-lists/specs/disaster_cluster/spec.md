## ADDED Requirements
### Requirement: List Cluster Names
The system SHALL provide an API to retrieve a list of all cluster names for selection purposes.

#### Scenario: Fetch cluster names
- **WHEN** client requests `GET /disaster/v1/clusters/names`
- **THEN** return a list of objects containing only the `name` field for all clusters
- **AND** the list is not paginated
