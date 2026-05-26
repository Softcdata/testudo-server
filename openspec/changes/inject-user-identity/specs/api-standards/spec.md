## ADDED Requirements
### Requirement: User Context Injection
API handlers MUST inject the authenticated user's identity into the resource metadata annotations during mutation operations.

#### Scenario: Create resource with user
- **WHEN** a user "admin" creates a resource via API
- **THEN** the created resource MUST have annotation `testudo.softcdata.com/user` set to "admin"

#### Scenario: Update resource with user
- **WHEN** a user "operator" updates a resource via API
- **THEN** the updated resource MUST have annotation `testudo.softcdata.com/user` set to "operator"
