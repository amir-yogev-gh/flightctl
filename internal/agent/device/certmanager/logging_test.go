package certmanager

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockLogger is a mock implementation of provider.Logger for testing.
type mockLogger struct {
	loggedMessages []string
	logLevels      []string
}

func (m *mockLogger) Info(args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprint(args...))
	m.logLevels = append(m.logLevels, "info")
}

func (m *mockLogger) Warn(args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprint(args...))
	m.logLevels = append(m.logLevels, "warn")
}

func (m *mockLogger) Error(args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprint(args...))
	m.logLevels = append(m.logLevels, "error")
}

func (m *mockLogger) Debug(args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprint(args...))
	m.logLevels = append(m.logLevels, "debug")
}

func (m *mockLogger) Infof(format string, args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprintf(format, args...))
	m.logLevels = append(m.logLevels, "info")
}

func (m *mockLogger) Warnf(format string, args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprintf(format, args...))
	m.logLevels = append(m.logLevels, "warn")
}

func (m *mockLogger) Errorf(format string, args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprintf(format, args...))
	m.logLevels = append(m.logLevels, "error")
}

func (m *mockLogger) Debugf(format string, args ...interface{}) {
	m.loggedMessages = append(m.loggedMessages, fmt.Sprintf(format, args...))
	m.logLevels = append(m.logLevels, "debug")
}

func TestLogCertificateOperation(t *testing.T) {
	mockLog := &mockLogger{}

	t.Run("Logs operation with all fields", func(t *testing.T) {
		mockLog.loggedMessages = []string{}
		mockLog.logLevels = []string{}

		ctx := CertificateLogContext{
			Operation:           "renewal",
			CertificateType:     "management",
			CertificateName:     "test-cert",
			DeviceName:          "test-device",
			Reason:              "proactive",
			ThresholdDays:       30,
			DaysUntilExpiration: 15,
			Duration:            5 * time.Second,
			Success:             true,
		}

		LogCertificateOperation(mockLog, ctx)

		assert.Len(t, mockLog.loggedMessages, 1)
		assert.Contains(t, mockLog.loggedMessages[0], "operation=renewal")
		assert.Contains(t, mockLog.loggedMessages[0], "certificate_type=management")
		assert.Contains(t, mockLog.loggedMessages[0], "certificate_name=test-cert")
		assert.Contains(t, mockLog.loggedMessages[0], "device_name=test-device")
		assert.Contains(t, mockLog.loggedMessages[0], "reason=proactive")
		assert.Contains(t, mockLog.loggedMessages[0], "threshold_days=30")
		assert.Contains(t, mockLog.loggedMessages[0], "days_until_expiration=15")
		assert.Contains(t, mockLog.loggedMessages[0], "duration_seconds=")
		assert.Contains(t, mockLog.loggedMessages[0], "success=true")
		assert.Equal(t, "info", mockLog.logLevels[0])
	})

	t.Run("Logs error with error field", func(t *testing.T) {
		mockLog.loggedMessages = []string{}
		mockLog.logLevels = []string{}

		ctx := CertificateLogContext{
			Operation:       "renewal",
			CertificateType: "management",
			CertificateName: "test-cert",
			Error:           errors.New("test error"),
			Success:         false,
		}

		LogCertificateOperation(mockLog, ctx)

		assert.Len(t, mockLog.loggedMessages, 1)
		assert.Contains(t, mockLog.loggedMessages[0], "operation=renewal")
		assert.Contains(t, mockLog.loggedMessages[0], "error=test error")
		assert.Contains(t, mockLog.loggedMessages[0], "success=false")
		assert.Equal(t, "error", mockLog.logLevels[0])
	})

	t.Run("Logs operation without optional fields", func(t *testing.T) {
		mockLog.loggedMessages = []string{}
		mockLog.logLevels = []string{}

		ctx := CertificateLogContext{
			Operation:       "validation",
			CertificateType: "management",
			CertificateName: "test-cert",
		}

		LogCertificateOperation(mockLog, ctx)

		assert.Len(t, mockLog.loggedMessages, 1)
		assert.Contains(t, mockLog.loggedMessages[0], "operation=validation")
		assert.NotContains(t, mockLog.loggedMessages[0], "reason=")
		assert.NotContains(t, mockLog.loggedMessages[0], "threshold_days=")
		assert.Equal(t, "info", mockLog.logLevels[0])
	})
}

func TestLogCertificateError(t *testing.T) {
	mockLog := &mockLogger{}

	t.Run("Logs error with context", func(t *testing.T) {
		mockLog.loggedMessages = []string{}
		mockLog.logLevels = []string{}

		err := errors.New("test error")
		context := map[string]interface{}{
			"reason":                "proactive",
			"days_until_expiration": 15,
		}

		LogCertificateError(mockLog, "renewal", "management", "test-cert", err, context)

		assert.Len(t, mockLog.loggedMessages, 1)
		assert.Contains(t, mockLog.loggedMessages[0], "operation=renewal")
		assert.Contains(t, mockLog.loggedMessages[0], "certificate_type=management")
		assert.Contains(t, mockLog.loggedMessages[0], "certificate_name=test-cert")
		assert.Contains(t, mockLog.loggedMessages[0], "error=test error")
		assert.Contains(t, mockLog.loggedMessages[0], "reason=proactive")
		assert.Contains(t, mockLog.loggedMessages[0], "days_until_expiration=15")
		assert.Equal(t, "error", mockLog.logLevels[0])
	})

	t.Run("Logs error without context", func(t *testing.T) {
		mockLog.loggedMessages = []string{}
		mockLog.logLevels = []string{}

		err := errors.New("test error")

		LogCertificateError(mockLog, "recovery", "management", "test-cert", err, nil)

		assert.Len(t, mockLog.loggedMessages, 1)
		assert.Contains(t, mockLog.loggedMessages[0], "operation=recovery")
		assert.Contains(t, mockLog.loggedMessages[0], "error=test error")
		assert.Equal(t, "error", mockLog.logLevels[0])
	})
}

func TestCertificateLogContext(t *testing.T) {
	t.Run("All fields can be set", func(t *testing.T) {
		ctx := CertificateLogContext{
			Operation:           "renewal",
			CertificateType:     "management",
			CertificateName:     "test-cert",
			DeviceName:          "test-device",
			Reason:              "proactive",
			ThresholdDays:       30,
			DaysUntilExpiration: 15,
			Duration:            5 * time.Second,
			Error:               errors.New("test"),
			Success:             true,
		}

		assert.Equal(t, "renewal", ctx.Operation)
		assert.Equal(t, "management", ctx.CertificateType)
		assert.Equal(t, "test-cert", ctx.CertificateName)
		assert.Equal(t, "test-device", ctx.DeviceName)
		assert.Equal(t, "proactive", ctx.Reason)
		assert.Equal(t, 30, ctx.ThresholdDays)
		assert.Equal(t, 15, ctx.DaysUntilExpiration)
		assert.Equal(t, 5*time.Second, ctx.Duration)
		assert.NotNil(t, ctx.Error)
		assert.True(t, ctx.Success)
	})
}
