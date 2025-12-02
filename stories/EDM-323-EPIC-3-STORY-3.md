# Story: Atomic Certificate Swap Operation

**Story ID:** EDM-323-EPIC-3-STORY-3  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to atomically swap certificates using POSIX atomic operations  
**So that** devices never lose valid certificates even during power loss

## Acceptance Criteria

Given a validated pending certificate  
When the atomic swap is performed  
Then the swap uses POSIX atomic rename operations

Given an atomic certificate swap  
When the swap completes  
Then the new certificate becomes active and the old certificate is removed

Given an atomic certificate swap  
When the swap is in progress  
Then the device always has at least one valid certificate available

Given an atomic certificate swap failure  
When the swap fails  
Then the old certificate is preserved and the pending certificate is cleaned up

Given a power loss during swap  
When the device restarts  
Then either the old or new certificate is active (never neither)

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/swap.go`
  - `AtomicSwap()` method
  - POSIX atomic rename operations
  - Backup and restore logic

### Atomic Operations
- Use `os.Rename()` for atomic file replacement (POSIX atomic)
- Rename pending certificate to active location
- Rename pending key to active location
- Both operations must succeed or both must fail

### Backup Strategy
- Backup current certificate before swap
- Restore backup if swap fails
- Clean up backup after successful swap

### Implementation
```go
// Atomic swap using POSIX rename (atomic operation)
if err := os.Rename(pendingCertPath, activeCertPath); err != nil {
    return err
}
if err := os.Rename(pendingKeyPath, activeKeyPath); err != nil {
    // Rollback first rename if possible
    return err
}
```

## Dependencies
- EDM-323-EPIC-3-STORY-2 (Certificate Validation)

## Testing Requirements
- Unit tests for atomic swap operations
- Integration tests for atomic swap
- Test power loss scenarios (simulated)
- Test concurrent swap attempts
- Test swap failure and rollback

## Definition of Done
- [ ] Atomic swap code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Power loss scenarios tested
- [ ] Documentation updated

