# Story: Certificate Validation Before Activation

**Story ID:** EDM-323-EPIC-3-STORY-2  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to validate new certificates before activating them  
**So that** I don't activate invalid or corrupted certificates

## Acceptance Criteria

Given a pending certificate  
When validation is performed  
Then the certificate signature chain is verified against the CA bundle

Given a pending certificate  
When validation is performed  
Then the certificate subject/SAN is verified to match device identity

Given a pending certificate  
When validation is performed  
Then the certificate expiration date is checked (must not be expired)

Given a pending certificate  
When validation is performed  
Then the certificate key pair is verified to match

Given a pending certificate validation failure  
When validation fails  
Then the pending certificate is rejected and the old certificate is preserved

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/swap.go` (new)
  - `ValidatePendingCertificate()` method
  - Certificate signature verification
  - Certificate identity verification
  - Certificate expiration check
  - Key pair verification

### Validation Steps
1. Load pending certificate and key
2. Parse X.509 certificate
3. Verify certificate signature chain against CA bundle
4. Verify certificate subject/SAN matches device identity
5. Verify certificate is not expired
6. Verify certificate and key match (can be used together)

### Error Handling
- Return detailed error messages for each validation step
- Clean up pending files on validation failure
- Preserve old certificate on validation failure

## Dependencies
- EDM-323-EPIC-3-STORY-1 (Pending Certificate Storage)

## Testing Requirements
- Unit tests for each validation step
- Unit tests for validation failures
- Integration tests for certificate validation
- Test with invalid certificates (wrong identity, expired, wrong CA)

## Definition of Done
- [ ] Certificate validation code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

