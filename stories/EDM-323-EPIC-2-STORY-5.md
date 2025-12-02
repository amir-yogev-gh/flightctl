# Story: Agent Certificate Reception and Storage

**Story ID:** EDM-323-EPIC-2-STORY-5  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to receive and store new certificates from renewal requests  
**So that** I can use the new certificate for future operations

## Acceptance Criteria

Given a renewal CSR in progress  
When the CSR is approved and signed  
Then the agent polls for and retrieves the new certificate

Given a new certificate  
When the certificate is received  
Then the agent validates the certificate before storing

Given a validated certificate  
When the certificate is stored  
Then it is stored in the certificate storage system

Given a stored certificate  
When the certificate is stored  
Then the certificate state is updated to reflect the new certificate

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/provider/provisioner/csr.go`
  - Enhance `check()` method to handle renewal certificates
  - Certificate validation before storage

### Certificate Validation
- Verify certificate signature chain
- Verify certificate subject/SAN matches device identity
- Verify certificate is not expired
- Verify certificate is signed by expected CA

### Storage
- Use existing certificate storage mechanism
- Store certificate and key in appropriate location
- Update certificate metadata

## Dependencies
- EDM-323-EPIC-2-STORY-2 (CSR Generation)
- EDM-323-EPIC-2-STORY-4 (Certificate Issuance)
- Existing CSR provisioner polling logic

## Testing Requirements
- Unit tests for certificate reception
- Unit tests for certificate validation
- Integration tests for certificate storage
- Test certificate validation failures

## Definition of Done
- [ ] Certificate reception code implemented
- [ ] Certificate validation implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

