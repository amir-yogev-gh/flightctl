# Story: Service-Side Renewal Request Validation

**Story ID:** EDM-323-EPIC-2-STORY-3  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control service  
**I want** to validate and auto-approve renewal requests from valid devices  
**So that** certificates can be renewed automatically without manual intervention

## Acceptance Criteria

Given a renewal CSR request  
When the request is received  
Then the service validates it is a renewal request (not initial enrollment)

Given a renewal CSR request  
When validating the request  
Then the service verifies the device exists and was previously enrolled

Given a renewal CSR request  
When validating the request  
Then the service verifies the current certificate is valid and matches the device

Given a valid renewal CSR request  
When the request is approved  
Then the service auto-approves the CSR without manual intervention

Given an invalid renewal CSR request  
When validation fails  
Then the service rejects the request with an appropriate error message

## Technical Details

### Components to Create/Modify
- `internal/service/certificatesigningrequest.go`
  - `validateRenewalRequest()` method
  - `validateProactiveRenewal()` method
  - Auto-approval logic for renewal CSRs

### Validation Steps
1. Check if CSR has renewal label/metadata
2. Extract device name from CSR CN
3. Verify device exists in database
4. Verify device was previously enrolled
5. Verify current certificate is valid
6. Verify certificate matches device identity

### Auto-Approval Logic
- If validation passes, automatically approve CSR
- Set approval condition on CSR status
- Trigger certificate signing

## Dependencies
- EDM-323-EPIC-1-STORY-4 (Database Schema)
- Existing CSR handler code

## Testing Requirements
- Unit tests for renewal validation logic
- Unit tests for auto-approval logic
- Integration tests for renewal request flow
- Test invalid renewal requests are rejected

## Definition of Done
- [ ] Renewal validation code implemented
- [ ] Auto-approval logic implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

