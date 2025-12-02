package store

import (
	"context"

	"github.com/flightctl/flightctl/internal/store/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CertificateRenewalEventStore interface {
	InitialMigration(ctx context.Context) error
	Create(ctx context.Context, orgID uuid.UUID, event *model.CertificateRenewalEvent) error
	List(ctx context.Context, orgID uuid.UUID, deviceID *uuid.UUID, eventType *string, limit int) ([]*model.CertificateRenewalEvent, error)
	Get(ctx context.Context, orgID uuid.UUID, eventID uuid.UUID) (*model.CertificateRenewalEvent, error)
}

type certificateRenewalEventStore struct {
	dbHandler *gorm.DB
	log       logrus.FieldLogger
}

// Make sure we conform to CertificateRenewalEventStore interface
var _ CertificateRenewalEventStore = (*certificateRenewalEventStore)(nil)

func NewCertificateRenewalEvent(db *gorm.DB, log logrus.FieldLogger) CertificateRenewalEventStore {
	return &certificateRenewalEventStore{
		dbHandler: db,
		log:       log,
	}
}

func (s *certificateRenewalEventStore) getDB(ctx context.Context) *gorm.DB {
	return s.dbHandler.WithContext(ctx)
}

func (s *certificateRenewalEventStore) InitialMigration(ctx context.Context) error {
	db := s.getDB(ctx)

	if err := db.AutoMigrate(&model.CertificateRenewalEvent{}); err != nil {
		return err
	}

	// Create indexes following the pattern used by other stores
	if err := s.createDeviceIDIndex(db); err != nil {
		return err
	}
	if err := s.createOrgIDIndex(db); err != nil {
		return err
	}
	if err := s.createCreatedAtIndex(db); err != nil {
		return err
	}
	if err := s.createEventTypeIndex(db); err != nil {
		return err
	}

	return nil
}

func (s *certificateRenewalEventStore) createDeviceIDIndex(db *gorm.DB) error {
	if !db.Migrator().HasIndex(&model.CertificateRenewalEvent{}, "idx_cert_renewal_events_device_id") {
		if db.Dialector.Name() == "postgres" {
			return db.Exec("CREATE INDEX idx_cert_renewal_events_device_id ON certificate_renewal_events(device_id)").Error
		} else {
			return db.Migrator().CreateIndex(&model.CertificateRenewalEvent{}, "DeviceID")
		}
	}
	return nil
}

func (s *certificateRenewalEventStore) createOrgIDIndex(db *gorm.DB) error {
	if !db.Migrator().HasIndex(&model.CertificateRenewalEvent{}, "idx_cert_renewal_events_org_id") {
		if db.Dialector.Name() == "postgres" {
			return db.Exec("CREATE INDEX idx_cert_renewal_events_org_id ON certificate_renewal_events(org_id)").Error
		} else {
			return db.Migrator().CreateIndex(&model.CertificateRenewalEvent{}, "OrgID")
		}
	}
	return nil
}

func (s *certificateRenewalEventStore) createCreatedAtIndex(db *gorm.DB) error {
	if !db.Migrator().HasIndex(&model.CertificateRenewalEvent{}, "idx_cert_renewal_events_created_at") {
		if db.Dialector.Name() == "postgres" {
			return db.Exec("CREATE INDEX idx_cert_renewal_events_created_at ON certificate_renewal_events(created_at)").Error
		} else {
			return db.Migrator().CreateIndex(&model.CertificateRenewalEvent{}, "CreatedAt")
		}
	}
	return nil
}

func (s *certificateRenewalEventStore) createEventTypeIndex(db *gorm.DB) error {
	if !db.Migrator().HasIndex(&model.CertificateRenewalEvent{}, "idx_cert_renewal_events_event_type") {
		if db.Dialector.Name() == "postgres" {
			return db.Exec("CREATE INDEX idx_cert_renewal_events_event_type ON certificate_renewal_events(event_type)").Error
		} else {
			return db.Migrator().CreateIndex(&model.CertificateRenewalEvent{}, "EventType")
		}
	}
	return nil
}

func (s *certificateRenewalEventStore) Create(ctx context.Context, orgID uuid.UUID, event *model.CertificateRenewalEvent) error {
	event.OrgID = orgID
	return s.getDB(ctx).Create(event).Error
}

func (s *certificateRenewalEventStore) List(ctx context.Context, orgID uuid.UUID, deviceID *uuid.UUID, eventType *string, limit int) ([]*model.CertificateRenewalEvent, error) {
	query := s.getDB(ctx).Where("org_id = ?", orgID)

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
	if err := s.getDB(ctx).Where("org_id = ? AND id = ?", orgID, eventID).First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}
