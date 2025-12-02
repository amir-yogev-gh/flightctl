# Story: Bootstrap Certificate Fallback Handler

**Story ID:** EDM-323-EPIC-4-STORY-2  
**Epic:** EDM-323-EPIC-4 (Expired Certificate Recovery)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to fall back to bootstrap certificate when management certificate is expired  
**So that** I can authenticate to the service for certificate recovery

## Acceptance Criteria

Given an expired management certificate  
When recovery is needed  
Then the agent attempts to load the bootstrap certificate

Given a bootstrap certificate  
When the certificate is loaded  
Then the agent validates the bootstrap certificate is not expired

Given a valid bootstrap certificate  
When authentication is needed  
Then the agent uses the bootstrap certificate for mTLS authentication

Given a bootstrap certificate fallback  
When the bootstrap certificate is used  
Then the agent can connect to the service for renewal requests

Given an expired or missing bootstrap certificate  
When fallback is attempted  
Then the agent falls back to TPM attestation (next story)

## Technical Details

### Components to Create/Modify
- `internal/agent/device/bootstrap.go`
  - `GetBootstrapCertificate()` method
  - Bootstrap certificate validation
  - Certificate switching logic

### Bootstrap Certificate
- Load from `/var/lib/flightctl/certs/bootstrap/cert.pem`
- Validate certificate is not expired
- Use for mTLS client authentication
- Switch between management and bootstrap certificates as needed

### Certificate Switching
- Detect which certificate to use based on management cert state
- Switch TLS client configuration based on certificate state
- Handle certificate loading errors gracefully

## Dependencies
- EDM-323-EPIC-4-STORY-1 (Expired Certificate Detection)
- Existing bootstrap certificate infrastructure

## Testing Requirements
- Unit tests for bootstrap certificate loading
- Unit tests for bootstrap certificate validation
- Unit tests for certificate switching
- Integration tests for bootstrap fallback
- Test with expired bootstrap certificate

## Definition of Done
- [ ] Bootstrap certificate fallback implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

