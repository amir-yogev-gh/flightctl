package model

import (
	"time"

	"github.com/google/uuid"
)

// CertificateRenewalEvent tracks certificate renewal and recovery events for auditing and troubleshooting.
type CertificateRenewalEvent struct {
	ID                uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	DeviceID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	OrgID             uuid.UUID  `gorm:"type:uuid;not null;index"`
	EventType         string     `gorm:"type:text;not null;index"`
	Reason            *string    `gorm:"type:text"`
	OldCertExpiration *time.Time `gorm:"type:timestamp"`
	NewCertExpiration *time.Time `gorm:"type:timestamp"`
	ErrorMessage      *string    `gorm:"type:text"`
	CreatedAt         time.Time  `gorm:"type:timestamp;not null;default:now();index"`
}
