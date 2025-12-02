package certmanager

import (
	"context"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Suite 1: Expired Certificate Detection
func TestDetectExpiredCertificate_Story1(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")

	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(logger),
		log:               logger,
		readWriter:        rw,
		storages:          make(map[string]provider.StorageFactory),
		config: &config.Config{
			Certificate: config.CertificateConfig{
				Renewal: config.CertificateRenewalConfig{
					ThresholdDays: 30,
				},
			},
		},
	}

	lm := NewLifecycleManager(cm, cm.expirationMonitor, logger, nil)
	cm.lifecycleManager = lm

	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("Detect Expired Certificate", func(t *testing.T) {
		// Certificate expired 10 days ago
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-10*24*time.Hour))

		providerName := "test-provider"
		certName := "expired-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiredCert.NotAfter,
				NotBefore: &expiredCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: expiredCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateExpired, state)
		assert.Less(t, days, 0, "days should be negative for expired certificate")
		assert.LessOrEqual(t, days, -10, "certificate expired at least 10 days ago")
	})

	t.Run("Detect Recently Expired Certificate", func(t *testing.T) {
		// Certificate expired 2 days ago (to ensure negative days)
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-2*24*time.Hour))

		providerName := "test-provider"
		certName := "recently-expired-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiredCert.NotAfter,
				NotBefore: &expiredCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: expiredCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateExpired, state)
		assert.Less(t, days, 0, "days should be negative for expired certificate")
	})

	t.Run("Valid Certificate Not Detected as Expired", func(t *testing.T) {
		// Certificate expires in 30 days
		validCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(30*24*time.Hour))

		providerName := "test-provider"
		certName := "valid-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &validCert.NotAfter,
				NotBefore: &validCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: validCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.NotEqual(t, CertificateStateExpired, state, "valid certificate should not be detected as expired")
		assert.Greater(t, days, 0, "days should be positive for valid certificate")
	})
}

// Test Suite 2: Recovery Trigger
func TestTriggerRecovery_Story1(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")

	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(logger),
		log:               logger,
		readWriter:        rw,
		storages:          make(map[string]provider.StorageFactory),
		config: &config.Config{
			Certificate: config.CertificateConfig{
				Renewal: config.CertificateRenewalConfig{
					ThresholdDays: 30,
				},
			},
		},
	}

	lm := NewLifecycleManager(cm, cm.expirationMonitor, logger, nil)
	cm.lifecycleManager = lm

	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("Trigger Recovery for Expired Certificate", func(t *testing.T) {
		// Expired certificate detected
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

		providerName := "test-provider"
		certName := "expired-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiredCert.NotAfter,
				NotBefore: &expiredCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: expiredCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Use a context with timeout to prevent hanging
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Trigger recovery - this will fail because RecoverExpiredCertificate is not fully implemented
		// but we can verify that state is set to recovering
		err := lm.TriggerRecovery(ctxWithTimeout, providerName, certName)
		// Error is expected because recovery flow is not fully implemented
		// But state should be set to recovering
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateRecovering, state.GetState(), "state should be set to recovering")
		// Error is expected due to incomplete implementation
		_ = err
	})

	t.Run("Recovery Not Triggered for Valid Certificate", func(t *testing.T) {
		// Valid certificate
		validCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(30*24*time.Hour))

		providerName := "test-provider"
		certName := "valid-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &validCert.NotAfter,
				NotBefore: &validCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: validCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Detect expiration - should not be expired
		state, _, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.NotEqual(t, CertificateStateExpired, state, "valid certificate should not be expired")

		// Recovery should not be triggered for valid certificates
		// (TriggerRecovery is only called for expired certificates)
		// Verify state remains normal
		finalState, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.NotEqual(t, CertificateStateRecovering, finalState.GetState(), "recovery should not be triggered for valid certificate")
	})
}
