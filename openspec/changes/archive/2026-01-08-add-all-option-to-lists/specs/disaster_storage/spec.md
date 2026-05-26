## ADDED Requirements
### Requirement: List Storage Names
The system SHALL provide an API to retrieve a list of all storage names for selection purposes.

#### Scenario: Fetch storage names
- **WHEN** client requests `GET /disaster/v1/storages/names`
- **THEN** return a list of objects containing only the `name` field for all storages
- **AND** the list is not paginated
