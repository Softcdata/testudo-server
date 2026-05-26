## ADDED Requirements
### Requirement: List Policy Names
The system SHALL provide an API to retrieve a list of all backup policy names for selection purposes.

#### Scenario: Fetch policy names
- **WHEN** client requests `GET /disaster/v1/policies/names`
- **THEN** return a list of objects containing only the `name` field for all policies
- **AND** the list is not paginated
