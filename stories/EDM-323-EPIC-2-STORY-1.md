# Story: Agent-Side Certificate Renewal Trigger

**Story ID:** EDM-323-EPIC-2-STORY-1  
**Epic:** EDM-323-EPIC-2 (Proactive Certificate Renewal)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to automatically trigger certificate renewal when approaching expiration  
**So that** certificates are renewed before they expire

## Acceptance Criteria

Given a certificate with expiration within threshold  
When the expiration monitor checks the certificate  
Then renewal is triggered automatically

Given a certificate renewal trigger  
When renewal is triggered  
Then the lifecycle manager initiates the renewal process

Given a renewal process  
When renewal is in progress  
Then the certificate state is set to "renewing"

Given a renewal trigger  
When the certificate is not within threshold  
Then renewal is not triggered

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/lifecycle.go`
  - `TriggerRenewal()` method
  - Renewal threshold comparison logic
  - State management for renewing certificates

### Integration Points
- Integrate with expiration monitor (EPIC-1)
- Integrate with CSR provisioner (next story)
- Update certificate state tracking

### Flow
1. Expiration monitor checks certificate
2. Compares days until expiration with threshold
3. If within threshold, triggers renewal
4. Sets certificate state to "renewing"
5. Initiates CSR generation

## Dependencies
- EDM-323-EPIC-1-STORY-1 (Certificate Expiration Monitoring)
- EDM-323-EPIC-1-STORY-2 (Certificate Lifecycle Manager)
- EDM-323-EPIC-1-STORY-3 (Configuration Schema)

## Testing Requirements
- Unit tests for renewal trigger logic
- Unit tests for threshold comparison
- Unit tests for state transitions
- Integration test with expiration monitor

## Definition of Done
- [ ] Renewal trigger code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

