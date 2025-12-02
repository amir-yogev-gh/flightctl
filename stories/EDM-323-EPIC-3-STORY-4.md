# Story: Rollback Mechanism for Failed Swaps

**Story ID:** EDM-323-EPIC-3-STORY-4  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to rollback to the old certificate if swap fails  
**So that** the device always has a valid certificate

## Acceptance Criteria

Given a certificate swap failure  
When the swap fails  
Then the old certificate is restored from backup

Given a certificate swap failure  
When rollback occurs  
Then the pending certificate files are cleaned up

Given a certificate swap failure  
When rollback occurs  
Then appropriate error is logged

Given a certificate swap failure  
When rollback occurs  
Then the certificate state is updated to reflect the failure

Given a successful rollback  
When the device continues operating  
Then it uses the old (still valid) certificate

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/swap.go`
  - `RollbackSwap()` method
  - Backup restoration logic
  - Cleanup logic for failed swaps

### Rollback Flow
1. Detect swap failure
2. Restore old certificate from backup
3. Clean up pending certificate files
4. Update certificate state
5. Log error and retry information

### Backup Management
- Create backup before swap
- Restore backup on failure
- Clean up backup after successful swap or rollback

## Dependencies
- EDM-323-EPIC-3-STORY-3 (Atomic Swap)

## Testing Requirements
- Unit tests for rollback logic
- Integration tests for rollback scenarios
- Test rollback after various failure points
- Test that device continues operating after rollback

## Definition of Done
- [ ] Rollback mechanism implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

