# Story: Certificate Expiration Monitoring Infrastructure

**Story ID:** EDM-323-EPIC-1-STORY-1  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 5  
**Priority:** High

## Description

**As a** Flight Control agent  
**I want** to monitor certificate expiration dates continuously  
**So that** I can proactively trigger renewal before certificates expire

## Acceptance Criteria

Given a device with a management certificate  
When the agent starts up  
Then the agent parses the certificate expiration date from the certificate file

Given a device with a management certificate  
When the agent runs periodically  
Then the agent checks the certificate expiration date at configurable intervals (default: daily)

Given a certificate with a known expiration date  
When the agent calculates days until expiration  
Then the calculation is accurate and accounts for time zones correctly

Given a certificate expiration check  
When the check completes  
Then the expiration information is stored in memory and available for renewal decisions

## Technical Details

### Components to Create/Modify
- `internal/agent/device/certmanager/expiration.go` (new)
  - `ParseCertificateExpiration()` function
  - `CalculateDaysUntilExpiration()` function
  - `IsExpired()` function
  - `IsExpiringSoon()` function

### Integration Points
- Integrate with existing `CertManager` in `internal/agent/device/certmanager/manager.go`
- Use existing certificate storage to load certificates
- Parse X.509 certificates using `crypto/x509`

### Configuration
- `certificate.renewal.check_interval` (default: 24h)

## Dependencies
- None (foundational story)

## Testing Requirements
- Unit tests for expiration date parsing
- Unit tests for days until expiration calculation
- Unit tests for edge cases (expired, expiring today, far future)
- Integration test with real certificate files

## Definition of Done
- [ ] Expiration monitoring code implemented
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

