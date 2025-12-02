# Story: Integration Test Suite for Certificate Rotation

**Story ID:** EDM-323-EPIC-6-STORY-2  
**Epic:** EDM-323-EPIC-6 (Testing and Validation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 8  
**Priority:** High

## Description

**As a** developer  
**I want** integration tests for certificate rotation flows  
**So that** I can validate end-to-end certificate operations

## Acceptance Criteria

Given a full renewal flow  
When integration test runs  
Then the test validates agent → service → agent renewal cycle

Given an expired certificate recovery  
When integration test runs  
Then the test validates complete recovery flow

Given a bootstrap certificate fallback  
When integration test runs  
Then the test validates fallback authentication works

Given atomic swap operations  
When integration test runs  
Then the test validates atomic swap under various conditions

Given retry logic  
When integration test runs  
Then the test validates exponential backoff and retry behavior

Given integration tests  
When tests run  
Then all tests pass consistently

## Technical Details

### Test Scenarios
1. **Proactive Renewal Flow**
   - Agent detects expiring certificate
   - Agent generates renewal CSR
   - Service validates and approves
   - Agent receives and swaps certificate

2. **Expired Certificate Recovery**
   - Agent detects expired certificate
   - Agent falls back to bootstrap
   - Agent generates recovery CSR with attestation
   - Service validates and issues certificate
   - Agent receives and swaps certificate

3. **Atomic Swap Scenarios**
   - Normal swap
   - Swap with validation failure
   - Swap with rollback

4. **Retry Scenarios**
   - Network interruption during renewal
   - Service unavailable during renewal
   - Retry with exponential backoff

### Test Infrastructure
- Use test harness for agent/service setup
- Mock or use real TPM simulator
- Use test certificates
- Validate database state

## Dependencies
- All implementation stories (EPIC-1 through EPIC-5)
- Test infrastructure

## Testing Requirements
- Integration tests for all major flows
- Test with real components (not just mocks)
- Test error scenarios
- Test retry logic

## Definition of Done
- [ ] Integration tests written for all flows
- [ ] All integration tests passing
- [ ] Code reviewed and approved
- [ ] Test documentation updated

