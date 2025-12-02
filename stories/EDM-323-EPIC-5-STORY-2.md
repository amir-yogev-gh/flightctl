# Story: Device Status Certificate State Indicators

**Story ID:** EDM-323-EPIC-5-STORY-2  
**Epic:** EDM-323-EPIC-5 (Configuration and Observability)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Description

**As a** fleet administrator  
**I want** to see certificate state in device status  
**So that** I can quickly identify devices with certificate issues

## Acceptance Criteria

Given a device status  
When certificate information is included  
Then certificate expiration date is reported

Given a device status  
When certificate information is included  
Then certificate state is reported (normal, expiring_soon, renewing, expired, recovering, renewal_failed)

Given a device status  
When certificate information is included  
Then days until expiration is reported

Given a device status  
When certificate information is included  
Then last renewal timestamp is reported (if applicable)

Given a device status  
When certificate information is included  
Then renewal count is reported (if applicable)

## Technical Details

### Components to Create/Modify
- `api/v1beta1/types.go` (or generated)
  - Add `CertificateStatus` to `DeviceStatus`
- `internal/agent/device/status/` (extend)
  - Certificate status collection
  - Certificate state determination

### Status Structure
```go
type CertificateStatus struct {
    Expiration        *time.Time `json:"expiration,omitempty"`
    DaysUntilExpiration int      `json:"daysUntilExpiration,omitempty"`
    State             string     `json:"state"` // normal, expiring_soon, renewing, expired, recovering, renewal_failed
    LastRenewed       *time.Time `json:"lastRenewed,omitempty"`
    RenewalCount      int        `json:"renewalCount,omitempty"`
}
```

### State Values
- `normal`: Certificate valid, not expiring soon
- `expiring_soon`: Certificate expiring within threshold
- `renewing`: Certificate renewal in progress
- `expired`: Certificate has expired
- `recovering`: Expired certificate recovery in progress
- `renewal_failed`: Certificate renewal failed

## Dependencies
- EDM-323-EPIC-1 (Foundation)
- EDM-323-EPIC-2 (Proactive Renewal)
- EDM-323-EPIC-4 (Expired Recovery)
- Existing device status infrastructure

## Testing Requirements
- Unit tests for certificate status collection
- Unit tests for state determination
- Integration tests for status reporting
- Test status updates during renewal/recovery

## Definition of Done
- [ ] Certificate status in device status implemented
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] API documentation updated
- [ ] Documentation updated

