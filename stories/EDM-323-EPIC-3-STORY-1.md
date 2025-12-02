# Story: Pending Certificate Storage Mechanism

**Story ID:** EDM-323-EPIC-3-STORY-1  
**Epic:** EDM-323-EPIC-3 (Atomic Certificate Swap)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to store new certificates in a pending location before activation  
**So that** I can validate them before making them active

## Acceptance Criteria

Given a new certificate received from renewal  
When the certificate is ready to be stored  
Then it is written to a pending location (cert.pem.pending)

Given a pending certificate  
When the certificate is written  
Then both certificate and key are written to pending locations

Given a pending certificate  
When writing to pending location  
Then the old certificate remains in the active location

Given a pending certificate write  
When the write fails  
Then the error is handled gracefully and the old certificate is preserved

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/provider/storage/fs.go`
  - `WritePending()` method
  - Pending certificate path handling
  - Error handling for pending writes

### File Paths
- Active: `/var/lib/flightctl/certs/management/cert.pem`
- Pending: `/var/lib/flightctl/certs/management/cert.pem.pending`
- Active key: `/var/lib/flightctl/certs/management/key.pem`
- Pending key: `/var/lib/flightctl/certs/management/key.pem.pending`

### Error Handling
- Ensure atomic writes (write to temp, then rename)
- Clean up pending files on error
- Preserve old certificate on failure

## Dependencies
- EDM-323-EPIC-2-STORY-5 (Certificate Reception)
- Existing certificate storage code

## Testing Requirements
- Unit tests for pending certificate writes
- Unit tests for error handling
- Integration tests for pending storage
- Test file system error scenarios

## Definition of Done
- [ ] Pending certificate storage implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

