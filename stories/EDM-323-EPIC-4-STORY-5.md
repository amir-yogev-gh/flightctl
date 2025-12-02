# Story: Complete Recovery Flow Implementation

**Story ID:** EDM-323-EPIC-4-STORY-5  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to complete the full recovery flow from expired certificate to new certificate  
**So that** devices can automatically recover without manual intervention

## Acceptance Criteria

Given an expired management certificate  
When recovery is triggered  
Then the agent follows the complete recovery flow

Given a recovery flow  
When bootstrap certificate is available  
Then the agent uses bootstrap certificate for authentication

Given a recovery flow  
When bootstrap certificate is not available  
Then the agent uses TPM attestation for authentication

Given a recovery flow  
When the new certificate is received  
Then the agent installs it using atomic swap

Given a successful recovery  
When recovery completes  
Then the agent resumes normal operations with the new certificate

Given a recovery failure  
When recovery fails  
Then the agent logs the error and retries according to retry policy

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/lifecycle.go`
  - `RecoverExpiredCertificate()` method
  - Complete recovery flow orchestration
  - Integration with all recovery components

### Recovery Flow
1. Detect expired certificate
2. Attempt bootstrap certificate fallback
3. If bootstrap unavailable, use TPM attestation
4. Generate renewal CSR with attestation
5. Submit CSR to service
6. Poll for certificate approval
7. Receive new certificate
8. Validate and atomically swap certificate
9. Resume normal operations

### Error Handling
- Handle bootstrap certificate loading failures
- Handle TPM attestation generation failures
- Handle CSR submission failures
- Handle certificate validation failures
- Implement retry logic with exponential backoff

## Dependencies
- EDM-323-EPIC-4-STORY-1 (Expired Certificate Detection)
- EDM-323-EPIC-4-STORY-2 (Bootstrap Certificate Fallback)
- EDM-323-EPIC-4-STORY-3 (TPM Attestation Generation)
- EDM-323-EPIC-4-STORY-4 (Recovery Validation)
- EDM-323-EPIC-3 (Atomic Swap)

## Testing Requirements
- Unit tests for recovery flow orchestration
- Integration tests for complete recovery flow
- E2E tests for offline → expired → recovery scenario
- Test recovery with bootstrap certificate
- Test recovery with TPM attestation
- Test recovery failure scenarios

## Definition of Done
- [ ] Complete recovery flow implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] Documentation updated

