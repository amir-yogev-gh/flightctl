# Developer Story: Certificate Renewal Events Table

**Story ID:** EDM-323-EPIC-7-STORY-1  
**Epic:** EDM-323-EPIC-7 (Database and API Enhancements)  
**Feature:** EDM-323 (Agent Certificate Rotation)  
**Story Points:** 3  
**Priority:** Medium

## Overview

Create database table and store layer for tracking certificate renewal events to enable auditing and troubleshooting of renewal operations.

## Implementation Tasks

### Task 1: Create Database Migration

**File:** `internal/store/migrations/XXXX_add_certificate_renewal_events.go` (new)

**Objective:** Create database migration for renewal events table.

**Implementation:**
```go
package migrations

import (
    "context"
    "github.com/flightctl/flightctl/internal/store/model"
    "gorm.io/gorm"
)

func AddCertificateRenewalEvents(ctx context.Context, db *gorm.DB) error {
    if err := db.AutoMigrate(&model.CertificateRenewalEvent{}); err != nil {
        return err
    }
    
    // Create indexes
    if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_device_id ON certificate_renewal_events(device_id)").Error; err != nil {
        return err
    }
    if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_org_id ON certificate_renewal_events(org_id)").Error; err != nil {
        return err
    }
    if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_created_at ON certificate_renewal_events(created_at)").Error; err != nil {
        return err
    }
    if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_cert_renewal_events_event_type ON certificate_renewal_events(event_type)").Error; err != nil {
        return err
    }
    
    return nil
}
```

**Testing:**
- Test migration creates table
- Test indexes are created
- Test migration is idempotent

---

### Task 2: Create CertificateRenewalEvent Model

**File:** `internal/store/model/certificate_renewal_event.go` (new)

**Objective:** Create GORM model for renewal events.

**Implementation:**
```go
package model

import (
    "time"
    "github.com/google/uuid"
)

type CertificateRenewalEvent struct {
    ID                uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    DeviceID          uuid.UUID `gorm:"type:uuid;not null;index"`
    OrgID             uuid.UUID `gorm:"type:uuid;not null;index"`
    EventType         string    `gorm:"type:text;not null;index"`
    Reason            *string   `gorm:"type:text"`
    OldCertExpiration *time.Time `gorm:"type:timestamp"`
    NewCertExpiration *time.Time `gorm:"type:timestamp"`
    ErrorMessage      *string   `gorm:"type:text"`
    CreatedAt         time.Time `gorm:"type:timestamp;not null;default:now();index"`
}
```

**Testing:**
- Test model can be created
- Test model fields are correct

---

### Task 3: Create Store Interface

**File:** `internal/store/certificate_renewal_events.go` (new)

**Objective:** Create store interface for renewal events.

**Implementation:**
```go
package store

import (
    "context"
    "time"
    "github.com/flightctl/flightctl/internal/store/model"
    "github.com/google/uuid"
)

type CertificateRenewalEventStore interface {
    Create(ctx context.Context, orgID uuid.UUID, event *model.CertificateRenewalEvent) error
    List(ctx context.Context, orgID uuid.UUID, deviceID *uuid.UUID, eventType *string, limit int) ([]*model.CertificateRenewalEvent, error)
    Get(ctx context.Context, orgID uuid.UUID, eventID uuid.UUID) (*model.CertificateRenewalEvent, error)
}

type certificateRenewalEventStore struct {
    db *gorm.DB
}

func (s *certificateRenewalEventStore) Create(ctx context.Context, orgID uuid.UUID, event *model.CertificateRenewalEvent) error {
    event.OrgID = orgID
    return s.db.WithContext(ctx).Create(event).Error
}

func (s *certificateRenewalEventStore) List(ctx context.Context, orgID uuid.UUID, deviceID *uuid.UUID, eventType *string, limit int) ([]*model.CertificateRenewalEvent, error) {
    query := s.db.WithContext(ctx).Where("org_id = ?", orgID)
    
    if deviceID != nil {
        query = query.Where("device_id = ?", *deviceID)
    }
    if eventType != nil {
        query = query.Where("event_type = ?", *eventType)
    }
    
    query = query.Order("created_at DESC").Limit(limit)
    
    var events []*model.CertificateRenewalEvent
    if err := query.Find(&events).Error; err != nil {
        return nil, err
    }
    
    return events, nil
}

func (s *certificateRenewalEventStore) Get(ctx context.Context, orgID uuid.UUID, eventID uuid.UUID) (*model.CertificateRenewalEvent, error) {
    var event model.CertificateRenewalEvent
    if err := s.db.WithContext(ctx).Where("org_id = ? AND id = ?", orgID, eventID).First(&event).Error; err != nil {
        return nil, err
    }
    return &event, nil
}
```

**Testing:**
- Test Create stores event
- Test List filters correctly
- Test Get retrieves event

---

### Task 4: Integrate Event Logging

**File:** `internal/service/certificatesigningrequest.go` (modify)

**Objective:** Log renewal events during certificate operations.

**Implementation:**
```go
// In CreateCertificateSigningRequest, log renewal start:
if h.isRenewalRequest(result) {
    event := &model.CertificateRenewalEvent{
        DeviceID:  deviceID,
        EventType: "renewal_start",
        Reason:    &reason,
    }
    _ = h.store.CertificateRenewalEvent().Create(ctx, orgId, event)
}

// In signApprovedCertificateSigningRequest, log renewal success:
event := &model.CertificateRenewalEvent{
    DeviceID:          deviceID,
    EventType:         "renewal_success",
    Reason:            &reason,
    NewCertExpiration: &newExpiration,
}
_ = h.store.CertificateRenewalEvent().Create(ctx, orgId, event)
```

**Testing:**
- Test events are logged
- Test event data is correct

---

## Definition of Done

- [ ] Renewal events table created
- [ ] Store layer implemented
- [ ] Event logging integrated
- [ ] Unit tests written and passing
- [ ] Code reviewed and approved
- [ ] Integration tests passing
- [ ] Documentation updated

---

## Related Files

- `internal/store/model/certificate_renewal_event.go` - Event model
- `internal/store/certificate_renewal_events.go` - Store interface
- `internal/service/certificatesigningrequest.go` - Event logging

---

## Dependencies

- **EDM-323-EPIC-1-STORY-4**: Database Schema (must be completed)
- **Existing Store Infrastructure**: Uses existing store patterns

---

## Notes

- **Event Types**: renewal_start, renewal_success, renewal_failed, recovery_start, recovery_success, recovery_failed
- **Indexes**: Indexes on device_id, org_id, created_at, event_type for efficient queries
- **Audit Trail**: Events provide audit trail for certificate operations
- **Troubleshooting**: Events enable troubleshooting of renewal issues

---

**Document End**

