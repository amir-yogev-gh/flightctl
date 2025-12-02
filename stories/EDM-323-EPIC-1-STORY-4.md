# Story: Database Schema for Certificate Tracking

**Story ID:** EDM-323-EPIC-1-STORY-4  
**Epic:** EDM-323-EPIC-1 (Certificate Lifecycle Foundation)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** High

## Description

**As a** Flight Control service  
**I want** to track certificate expiration and renewal history in the database  
**So that** I can monitor certificate health and audit renewal operations

## Acceptance Criteria

Given a database migration  
When the migration runs  
Then certificate tracking fields are added to the devices table

Given a device with a certificate  
When the certificate is issued or renewed  
Then the certificate expiration date is stored in the database

Given a certificate renewal event  
When renewal occurs  
Then the event is logged in the certificate_renewal_events table

Given certificate tracking data  
When querying devices  
Then queries can efficiently filter by certificate expiration date using indexes

## Technical Details

### Components to Create/Modify
- Database migration script
  - Add `certificate_expiration` TIMESTAMP to devices table
  - Add `certificate_last_renewed` TIMESTAMP to devices table
  - Add `certificate_renewal_count` INTEGER to devices table
  - Add `certificate_fingerprint` TEXT to devices table
  - Create indexes for efficient queries

### Migration Script
```sql
ALTER TABLE devices 
ADD COLUMN certificate_expiration TIMESTAMP,
ADD COLUMN certificate_last_renewed TIMESTAMP,
ADD COLUMN certificate_renewal_count INTEGER DEFAULT 0,
ADD COLUMN certificate_fingerprint TEXT;

CREATE INDEX idx_devices_cert_expiration 
ON devices(certificate_expiration) 
WHERE certificate_expiration IS NOT NULL;
```

### Store Layer Updates
- `internal/store/device.go` - Add certificate tracking fields to Device model
- Update device creation/update methods to handle certificate fields

## Dependencies
- None (can be done in parallel)

## Testing Requirements
- Unit tests for migration script
- Integration tests for database schema
- Tests for store layer updates
- Performance tests for indexed queries

## Definition of Done
- [ ] Migration script created and tested
- [ ] Store layer updated
- [ ] Unit tests written and passing
- [ ] Migration tested on test database
- [ ] Code reviewed and approved
- [ ] Migration documented

