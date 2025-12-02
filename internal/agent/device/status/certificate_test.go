package status

import (
	"context"
	"testing"
	"time"

	"github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertificateExporter(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	// Test with nil certManager - should not panic
	exporter := NewCertificateExporter(nil, logger)

	assert.NotNil(t, exporter)
	assert.Equal(t, logger, exporter.log)
}

func TestCertificateExporter_Status_NoCertManager(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	exporter := &CertificateExporter{
		certManager: nil,
		log:         logger,
	}

	deviceStatus := &v1beta1.DeviceStatus{
		SystemInfo: v1beta1.DeviceSystemInfo{},
	}

	err := exporter.Status(context.Background(), deviceStatus)
	assert.NoError(t, err)
}

func TestCertificateExporter_Status_AddsToCustomInfo(t *testing.T) {
	logger := log.NewPrefixLogger("test")

	// Test with nil certManager - should not add custom info
	exporter := NewCertificateExporter(nil, logger)

	deviceStatus := &v1beta1.DeviceStatus{
		SystemInfo: v1beta1.DeviceSystemInfo{},
	}

	err := exporter.Status(context.Background(), deviceStatus)
	require.NoError(t, err)

	// Since lifecycle manager is nil, no custom info should be added
	assert.Nil(t, deviceStatus.SystemInfo.CustomInfo)
}

func TestCertificateStatus_Fields(t *testing.T) {
	now := time.Now()
	expiration := now.Add(30 * 24 * time.Hour)
	days := 30
	state := "normal"
	lastRenewed := now.Add(-7 * 24 * time.Hour)
	renewalCount := 5

	status := &CertificateStatus{
		Expiration:          &expiration,
		DaysUntilExpiration: &days,
		State:               state,
		LastRenewed:         &lastRenewed,
		RenewalCount:        &renewalCount,
	}

	assert.NotNil(t, status.Expiration)
	assert.NotNil(t, status.DaysUntilExpiration)
	assert.Equal(t, state, status.State)
	assert.NotNil(t, status.LastRenewed)
	assert.NotNil(t, status.RenewalCount)
	assert.Equal(t, 30, *status.DaysUntilExpiration)
	assert.Equal(t, 5, *status.RenewalCount)
}

func TestCertificateStatus_OptionalFields(t *testing.T) {
	status := &CertificateStatus{
		State: "normal",
	}

	// All fields except State are optional
	assert.Equal(t, "normal", status.State)
	assert.Nil(t, status.Expiration)
	assert.Nil(t, status.DaysUntilExpiration)
	assert.Nil(t, status.LastRenewed)
	assert.Nil(t, status.RenewalCount)
}
