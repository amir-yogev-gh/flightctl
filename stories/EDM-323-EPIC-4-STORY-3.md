# Story: TPM Attestation Generation for Recovery

**Story ID:** EDM-323-EPIC-4-STORY-3  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to generate TPM attestation for expired certificate recovery  
**So that** I can prove device identity when both certificates are expired

## Acceptance Criteria

Given an expired certificate recovery request  
When TPM attestation is needed  
Then the agent generates a TPM quote

Given a TPM attestation request  
When attestation is generated  
Then the agent includes PCR values in the attestation

Given a TPM attestation request  
When attestation is generated  
Then the agent includes device fingerprint in the attestation

Given a TPM attestation  
When the attestation is created  
Then it can be included in renewal CSR requests

## Technical Details

### Components to Create/Modify
- `internal/agent/identity/tpm.go`
  - `GenerateRenewalAttestation()` method
  - TPM quote generation
  - PCR value reading
  - Device fingerprint generation

### Attestation Structure
```go
type RenewalAttestation struct {
    TPMQuote          []byte
    PCRValues         map[int][]byte
    DeviceFingerprint string
}
```

### TPM Operations
- Generate TPM quote using TPM client
- Read PCR values from TPM
- Get device fingerprint from identity provider
- Package attestation for CSR submission

## Dependencies
- EDM-323-EPIC-4-STORY-1 (Expired Certificate Detection)
- Existing TPM infrastructure

## Testing Requirements
- Unit tests for TPM attestation generation
- Unit tests for PCR value reading
- Unit tests for device fingerprint generation
- Integration tests with TPM hardware (or simulator)
- Test attestation packaging

## Definition of Done
- [ ] TPM attestation generation implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] TPM hardware/simulator tests passing
- [ ] Documentation updated

