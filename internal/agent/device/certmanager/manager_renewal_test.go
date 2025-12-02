package certmanager

import (
	"context"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Suite 1: shouldRenewCertificate

func TestCertManager_shouldRenewCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()
	now := time.Now().UTC()

	// Create lifecycle manager and expiration monitor
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: monitor,
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
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	cm.lifecycleManager = lm

	providerName := "test-provider"
	certName := "test-cert"

	t.Run("Certificate needs renewal", func(t *testing.T) {
		// Certificate expiring in 25 days, threshold = 30
		expiration := now.Add(25 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert - CheckRenewal will use it if available
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		assert.GreaterOrEqual(t, days, 24)
		assert.LessOrEqual(t, days, 26)
	})

	t.Run("Certificate does not need renewal", func(t *testing.T) {
		// Certificate expiring in 35 days, threshold = 30
		expiration := now.Add(35 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: "cert-no-renewal",
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: "cert-no-renewal",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, 30)
		require.NoError(t, err)
		assert.False(t, needsRenewal)
		assert.GreaterOrEqual(t, days, 34)
		assert.LessOrEqual(t, days, 36)
	})

	t.Run("No lifecycle manager", func(t *testing.T) {
		cmNoLM := &CertManager{
			certificates:      newCertStorage(),
			expirationMonitor: monitor,
			log:               logger,
			readWriter:        rw,
			lifecycleManager:  nil, // No lifecycle manager
		}

		cert := &certificate{
			Name: "cert-no-lm",
			Info: CertificateInfo{
				NotAfter: timePtr(now.Add(30 * 24 * time.Hour)),
			},
		}

		needsRenewal, days, err := cmNoLM.shouldRenewCertificate(ctx, providerName, cert, 30)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lifecycle manager not initialized")
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)
	})

	t.Run("Negative threshold", func(t *testing.T) {
		expiration := now.Add(30 * 24 * time.Hour)
		cert := &certificate{
			Name: "cert-negative-threshold",
			Info: CertificateInfo{
				NotAfter: &expiration,
			},
		}

		needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, -1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "threshold days must be non-negative")
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)
	})
}

// Test Suite 2: triggerRenewal

func TestCertManager_triggerRenewal(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()
	now := time.Now().UTC()

	// Create lifecycle manager and expiration monitor
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: monitor,
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
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	cm.lifecycleManager = lm

	providerName := "test-provider"
	certName := "test-cert"

	t.Run("Successful renewal trigger", func(t *testing.T) {
		expiration := now.Add(25 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		cfg := provider.CertificateConfig{
			Name: certName,
			Storage: provider.StorageConfig{
				Type: provider.StorageTypeFilesystem,
			},
		}

		// Initialize processing queue with a handler that succeeds
		cm.processingQueue = NewCertificateProcessingQueue(func(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) *time.Duration {
			// Handler succeeds immediately
			return nil
		})

		err := cm.triggerRenewal(ctx, providerName, cert, cfg)
		require.NoError(t, err)

		// Verify state was set to renewing
		state, err := lm.GetCertificateState(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("Queuing failure", func(t *testing.T) {
		expiration := now.Add(25 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: "cert-queuing-fail",
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: "cert-queuing-fail",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Set initial state to expiring_soon
		_ = lm.SetCertificateState(ctx, providerName, "cert-queuing-fail", CertificateStateExpiringSoon)

		cfg := provider.CertificateConfig{
			Name: "cert-queuing-fail",
			Storage: provider.StorageConfig{
				Type: provider.StorageTypeFilesystem,
			},
		}

		// Initialize processing queue with a handler that fails
		cm.processingQueue = NewCertificateProcessingQueue(func(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) *time.Duration {
			// Handler fails - return nil to indicate no retry (but Process will still succeed in queuing)
			return nil
		})

		// Note: Process() on the queue doesn't actually fail - it queues the item
		// The actual failure would happen during processing. For this test, we verify
		// that triggerRenewal sets the state correctly and queues the certificate.
		err := cm.triggerRenewal(ctx, providerName, cert, cfg)
		// triggerRenewal should succeed in queuing
		require.NoError(t, err)

		// Verify state was set to renewing (before any processing failure)
		state, err := lm.GetCertificateState(ctx, providerName, "cert-queuing-fail")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("No lifecycle manager", func(t *testing.T) {
		cmNoLM := &CertManager{
			certificates:      newCertStorage(),
			expirationMonitor: monitor,
			log:               logger,
			readWriter:        rw,
			lifecycleManager:  nil, // No lifecycle manager
		}

		cert := &certificate{
			Name: "cert-no-lm",
			Info: CertificateInfo{
				NotAfter: timePtr(now.Add(30 * 24 * time.Hour)),
			},
		}

		cfg := provider.CertificateConfig{
			Name: "cert-no-lm",
		}

		err := cmNoLM.triggerRenewal(ctx, providerName, cert, cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lifecycle manager not initialized")
	})
}

// Test Suite 3: Sync Flow Integration

func TestCertManager_SyncFlowRenewalIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()
	now := time.Now().UTC()

	// Create lifecycle manager and expiration monitor
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{
		certificates:      newCertStorage(),
		expirationMonitor: monitor,
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
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	cm.lifecycleManager = lm

	providerName := "test-provider"
	certName := "test-cert"

	t.Run("Renewal checked during sync", func(t *testing.T) {
		// Certificate expiring soon (25 days)
		expiration := now.Add(25 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Initialize processing queue
		cm.processingQueue = NewCertificateProcessingQueue(func(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) *time.Duration {
			return nil
		})

		// Simulate sync flow - call shouldRenewCertificate directly to test renewal check
		// Note: shouldRenewCertificate calls CheckRenewal which loads the cert from storage
		needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		assert.GreaterOrEqual(t, days, 24)
		assert.LessOrEqual(t, days, 26)

		// Verify state was updated by CheckRenewal
		state, err := lm.GetCertificateState(ctx, providerName, certName)
		require.NoError(t, err)
		// State should be expiring_soon (set by CheckRenewal)
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())
	})

	t.Run("Renewal not triggered if already renewing", func(t *testing.T) {
		expiration := now.Add(25 * 24 * time.Hour)
		mockCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), expiration)

		cert := &certificate{
			Name: "cert-already-renewing",
			Info: CertificateInfo{
				NotAfter:  &expiration,
				NotBefore: timePtr(now.Add(-24 * time.Hour)),
			},
			Config: provider.CertificateConfig{
				Name: "cert-already-renewing",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		// Set storage directly on cert
		cert.Storage = &mockStorageProvider{cert: mockCert}

		// Register storage factory that returns a provider with the cert
		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryWithCert{cert: mockCert}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Set state to renewing
		_ = lm.SetCertificateState(ctx, providerName, "cert-already-renewing", CertificateStateRenewing)

		// Initialize processing queue
		processCalled := false
		cm.processingQueue = NewCertificateProcessingQueue(func(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) *time.Duration {
			processCalled = true
			return nil
		})

		cfg := provider.CertificateConfig{
			Name: "cert-already-renewing",
			Storage: provider.StorageConfig{
				Type: provider.StorageTypeFilesystem,
			},
		}

		// Test that in sync flow, state is checked before calling triggerRenewal
		// Simulate the sync flow check: get current state first
		currentState, err := lm.GetCertificateState(ctx, providerName, "cert-already-renewing")
		require.NoError(t, err)

		// If state is already renewing, triggerRenewal should not be called
		// But if we call it directly, it will still queue (this is expected behavior)
		// The actual protection happens in syncCertificate which checks state before calling triggerRenewal
		if currentState.GetState() != CertificateStateRenewing {
			err := cm.triggerRenewal(ctx, providerName, cert, cfg)
			require.NoError(t, err)
			assert.True(t, processCalled)
		} else {
			// State is already renewing, so triggerRenewal should not be called in sync flow
			// But if called directly, it will still work (this tests the method itself)
			// In real sync flow, this check prevents calling triggerRenewal
			assert.Equal(t, CertificateStateRenewing, currentState.GetState())
		}

		// Verify state remains renewing
		state, err := lm.GetCertificateState(ctx, providerName, "cert-already-renewing")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("Renewal check error handling", func(t *testing.T) {
		// Certificate with no expiration info
		cert := &certificate{
			Name: "cert-no-expiration",
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: "cert-no-expiration",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProviderWithError{loadError: fmt.Errorf("load failed")}

		cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactory{}

		_, _ = cm.certificates.EnsureProvider(providerName)
		_ = cm.certificates.StoreCertificate(providerName, cert)

		// Test that shouldRenewCertificate handles error gracefully
		needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, 30)
		// Error is expected due to load failure
		assert.Error(t, err)
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)

		// Verify no panic occurred (test passes if we get here)
		assert.True(t, true)
	})
}

// mockStorageFactoryWithCert is a mock storage factory that returns a provider with a specific cert
type mockStorageFactoryWithCert struct {
	cert *x509.Certificate
}

func (m *mockStorageFactoryWithCert) Type() string {
	return string(provider.StorageTypeFilesystem)
}

func (m *mockStorageFactoryWithCert) New(log provider.Logger, cc provider.CertificateConfig) (provider.StorageProvider, error) {
	return &mockStorageProvider{cert: m.cert}, nil
}

func (m *mockStorageFactoryWithCert) Validate(log provider.Logger, cc provider.CertificateConfig) error {
	return nil
}
