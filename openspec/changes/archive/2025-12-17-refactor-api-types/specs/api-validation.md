# API Types Refactoring Specification Delta

## ADDED Requirements

#### Scenario: Validate AppRestore Request
Given a CreateAppRestoreRequest with missing required fields
When the API handler processes the request
Then it should return a 400 Bad Request error with validation details
And the request should not be converted to a CRD object

#### Scenario: Convert AppRestore DTO to CRD
Given a valid CreateAppRestoreRequest with ResourceModifiers
When the request is converted to CRD
Then the resulting AppRestoreSpec should contain the correct ResourceModifierRules
And the structure should match the Operator's expectation
