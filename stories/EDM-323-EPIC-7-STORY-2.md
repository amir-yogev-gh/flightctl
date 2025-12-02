# Story: Store Layer Extensions for Certificate Tracking

**Story ID:** EDM-323-EPIC-7-STORY-2  
**Epic:** EDM-323-EPIC-7 (Database and API Enhancements)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Description

**As a** Flight Control service  
**I want** store layer methods to update certificate tracking fields  
**So that** certificate metadata is properly maintained in the database

## Acceptance Criteria

Given a device certificate update  
When certificate is issued or renewed  
Then certificate tracking fields are updated in the database

Given certificate tracking fields  
When device is queried  
Then certificate metadata is included in device records

Given certificate tracking operations  
When operations are performed  
Then database transactions ensure data consistency

Given certificate tracking fields  
When fields are updated  
Then updates are efficient and use appropriate indexes

## Technical Details

### Components to Create/Modify
- `internal/store/device.go`
  - `UpdateCertificateExpiration()` method
  - `UpdateCertificateRenewalInfo()` method
  - Update device creation/update methods to handle certificate fields

### Methods to Add
```go
func (s *DeviceStore) UpdateCertificateExpiration(ctx context.Context, orgID uuid.UUID, deviceName string, expiration time.Time) error

func (s *DeviceStore) UpdateCertificateRenewalInfo(ctx context.Context, orgID uuid.UUID, deviceName string, lastRenewed time.Time, renewalCount int) error
```

### Integration Points
- Integrate with certificate issuance code
- Integrate with renewal completion code
- Update device model to include certificate fields

## Dependencies
- EDM-323-EPIC-1-STORY-4 (Database Schema)
- EDM-323-EPIC-7-STORY-1 (Renewal Events Table)
- Existing store infrastructure

## Testing Requirements
- Unit tests for certificate tracking updates
- Unit tests for database transactions
- Integration tests for certificate tracking
- Performance tests for updates

## Definition of Done
- [ ] Store layer extensions implemented
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

