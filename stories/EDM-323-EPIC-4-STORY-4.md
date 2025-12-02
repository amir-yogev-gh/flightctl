# Story: Service-Side Recovery Request Validation

**Story ID:** EDM-323-EPIC-4-STORY-4  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control service  
**I want** to validate recovery requests from devices with expired certificates  
**So that** I can securely issue new certificates to legitimate devices

## Acceptance Criteria

Given a recovery CSR request  
When the request is received  
Then the service validates it is a recovery request (expired certificate)

Given a recovery CSR request  
When validating the request  
Then the service verifies the device exists and was previously enrolled

Given a recovery CSR request with bootstrap certificate  
When validating the request  
Then the service accepts bootstrap certificate authentication

Given a recovery CSR request with TPM attestation  
When validating the request  
Then the service validates TPM attestation against device records

Given a recovery CSR request  
When validating the request  
Then the service validates device fingerprint matches device record

Given a valid recovery CSR request  
When validation passes  
Then the service auto-approves and issues a new certificate

Given an invalid recovery CSR request  
When validation fails  
Then the service rejects the request with an appropriate error

## Technical Details

### Components to Create/Modify
- `internal/service/certificatesigningrequest.go`
  - `validateExpiredCertificateRenewal()` method
  - TPM attestation validation
  - Device fingerprint validation
  - Bootstrap certificate acceptance

### Validation Steps
1. Verify device exists in database
2. Verify device was previously enrolled
3. If bootstrap certificate: validate bootstrap cert is valid
4. If TPM attestation: validate TPM quote and PCR values
5. Validate device fingerprint matches stored value
6. Check device is not revoked/blacklisted

### TPM Attestation Validation
- Verify TPM quote signature
- Compare PCR values with expected values
- Verify device fingerprint matches device record
- Check attestation freshness (prevent replay)

## Dependencies
- EDM-323-EPIC-4-STORY-3 (TPM Attestation Generation)
- Existing TPM validation infrastructure

## Testing Requirements
- Unit tests for recovery validation logic
- Unit tests for TPM attestation validation
- Unit tests for device fingerprint validation
- Integration tests for recovery request flow
- Test invalid recovery requests are rejected

## Definition of Done
- [ ] Recovery validation code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

