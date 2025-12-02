# Story: Certificate Renewal Events Table

**Story ID:** EDM-323-EPIC-7-STORY-1  
**Epic:** EDM-323-EPIC-7 (Database and API Enhancements)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Description

**As a** Flight Control service  
**I want** to track certificate renewal events in the database  
**So that** I can audit renewal operations and troubleshoot issues

## Acceptance Criteria

Given a certificate renewal event  
When the event occurs  
Then it is logged in the certificate_renewal_events table

Given renewal events  
When events are logged  
Then event types are tracked (renewal_start, renewal_success, renewal_failed, recovery_start, recovery_success, recovery_failed)

Given renewal events  
When events are logged  
Then event metadata is stored (reason, old_cert_expiration, new_cert_expiration, error_message)

Given renewal events  
When events are queried  
Then queries can efficiently filter by device, organization, event type, and timestamp using indexes

## Technical Details

### Components to Create/Modify
- Database migration script
  - Create `certificate_renewal_events` table
  - Create indexes for efficient queries
- `internal/store/certificate_renewal_events.go` (new)
  - Store interface for renewal events
  - CRUD operations for renewal events

### Table Schema
```sql
CREATE TABLE certificate_renewal_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    reason TEXT,
    old_cert_expiration TIMESTAMP,
    new_cert_expiration TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Indexes
- Index on `device_id` for device-specific queries
- Index on `org_id` for organization-specific queries
- Index on `created_at` for time-based queries
- Index on `event_type` for event type filtering

## Dependencies
- EDM-323-EPIC-1-STORY-4 (Database Schema)
- Existing store infrastructure

## Testing Requirements
- Unit tests for renewal event storage
- Unit tests for event queries
- Integration tests for event logging
- Performance tests for indexed queries

## Definition of Done
- [ ] Renewal events table created
- [ ] Store layer implemented
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

