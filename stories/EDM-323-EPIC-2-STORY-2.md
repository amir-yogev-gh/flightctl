# Story: CSR Generation for Certificate Renewal

**Story ID:** EDM-323-EPIC-2-STORY-2  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to generate certificate signing requests for renewal  
**So that** I can request new certificates from the service

## Acceptance Criteria

Given a certificate renewal trigger  
When renewal is initiated  
Then the agent generates a CSR with the same device identity

Given a renewal CSR  
When the CSR is created  
Then it includes renewal context in metadata/labels

Given a renewal CSR  
When the CSR is submitted  
Then it uses the current valid certificate for authentication

Given a renewal CSR  
When the CSR is generated  
Then it uses the same TPM-backed identity as the original certificate

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/provider/provisioner/csr.go`
  - Extend `Provision()` to accept renewal context
  - Add renewal reason to CSR metadata
  - Ensure same device identity is used

### CSR Metadata
```json
{
  "metadata": {
    "labels": {
      "flightctl.io/renewal-reason": "proactive",
      "flightctl.io/renewal-threshold-days": "30"
    }
  }
}
```

### Integration Points
- Use existing CSR provisioner infrastructure
- Integrate with TPM identity provider
- Use existing certificate for authentication

## Dependencies
- EDM-323-EPIC-2-STORY-1 (Renewal Trigger)
- Existing CSR provisioner code

## Testing Requirements
- Unit tests for CSR generation with renewal context
- Unit tests for device identity preservation
- Integration test with CSR submission
- Test that renewal CSR uses valid certificate for auth

## Definition of Done
- [ ] CSR generation for renewal implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

