# Story: Expired Certificate Detection

**Story ID:** EDM-323-EPIC-4-STORY-1  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to detect expired management certificates  
**So that** I can trigger recovery when certificates have expired

## Acceptance Criteria

Given a device with a management certificate  
When the agent starts up  
Then the agent checks if the certificate is expired

Given a device with a management certificate  
When the agent runs periodically  
Then the agent checks if the certificate is expired

Given an expired certificate  
When expiration is detected  
Then the agent differentiates between "expiring soon" and "already expired"

Given an expired certificate  
When expiration is detected  
Then the certificate state is set to "expired"

Given an expired certificate  
When expiration is detected  
Then recovery is triggered automatically

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/lifecycle.go`
  - `DetectExpiredCertificate()` method
  - Expiration detection logic
  - State management for expired certificates

### Detection Logic
- Compare certificate expiration date with current time
- Differentiate between:
  - "expiring soon" (< threshold days remaining)
  - "already expired" (past expiration date)

### Integration Points
- Integrate with expiration monitor
- Trigger recovery flow when expired certificate detected
- Update certificate state

## Dependencies
- EDM-323-EPIC-1-STORY-1 (Certificate Expiration Monitoring)
- EDM-323-EPIC-1-STORY-2 (Certificate Lifecycle Manager)

## Testing Requirements
- Unit tests for expiration detection
- Unit tests for state differentiation
- Integration tests for expiration detection
- Test with certificates at various expiration states

## Definition of Done
- [ ] Expiration detection code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

