## ADDED Requirements
### Requirement: Fuzzy Search Clusters
The system SHALL support fuzzy searching of clusters by name or tag using a single keyword.

#### Scenario: Search clusters by keyword
- **WHEN** client requests `GET /disaster/v1/clusters?keyword=test`
- **THEN** return clusters where `name` contains "test" OR `cluster-tag` label contains "test"
- **AND** the filtering happens before pagination
