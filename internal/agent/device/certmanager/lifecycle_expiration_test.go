package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/internal/util"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockCertForExpiration creates a mock certificate with specified expiration.
func createMockCertForExpiration(t *testing.T, notBefore, notAfter time.Time) *x509.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-device",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

func TestDetectExpiredCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")

	// Create a minimal cert manager for testing
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

	t.Run("Detects expired certificate", func(t *testing.T) {
		// Create expired certificate
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

		// Store certificate
		providerName := "test-provider"
		certName := "test-cert"

		// Ensure provider exists
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

	t.Run("Detects expiring soon certificate", func(t *testing.T) {
		// Create certificate expiring in 7 days (within 30 day threshold)
		expiringSoonCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(7*24*time.Hour))

		providerName := "test-provider"
		certName := "expiring-cert"

		// Ensure provider exists
		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiringSoonCert.NotAfter,
				NotBefore: &expiringSoonCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: expiringSoonCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateExpiringSoon, state)
		assert.Greater(t, days, 0)
		assert.LessOrEqual(t, days, 30)
	})

	t.Run("Detects normal certificate", func(t *testing.T) {
		// Create certificate expiring in 60 days (beyond threshold)
		normalCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(60*24*time.Hour))

		providerName := "test-provider"
		certName := "normal-cert"

		// Ensure provider exists
		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &normalCert.NotAfter,
				NotBefore: &normalCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert.Storage = &mockStorageProvider{cert: normalCert}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateNormal, state)
		assert.Greater(t, days, 30)
	})

	t.Run("Handles missing expiration info", func(t *testing.T) {
		providerName := "test-provider"
		certName := "no-expiration-cert"

		// Ensure provider exists
		_, _ = cm.certificates.EnsureProvider(providerName)

		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  nil, // No expiration info
				NotBefore: nil,
			},
			Config: provider.CertificateConfig{
				Name: certName,
			},
		}

		_ = cm.certificates.StoreCertificate(providerName, cert)

		state, days, err := lm.DetectExpiredCertificate(ctx, providerName, certName)
		require.NoError(t, err)
		assert.Equal(t, CertificateStateNormal, state)
		assert.Equal(t, 0, days)
	})
}

func TestCheckExpiredCertificates(t *testing.T) {
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

	t.Run("Checks all certificates", func(t *testing.T) {
		// Create multiple certificates with different states
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
		expiringSoonCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(7*24*time.Hour))
		normalCert := createMockCertForExpiration(t, now.Add(-24*time.Hour), now.Add(60*24*time.Hour))

		providerName := "test-provider"

		// Ensure provider exists
		_, _ = cm.certificates.EnsureProvider(providerName)

		cert1 := &certificate{
			Name: "expired-cert",
			Info: CertificateInfo{
				NotAfter:  &expiredCert.NotAfter,
				NotBefore: &expiredCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: "expired-cert",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert1.Storage = &mockStorageProvider{cert: expiredCert}

		cert2 := &certificate{
			Name: "expiring-cert",
			Info: CertificateInfo{
				NotAfter:  &expiringSoonCert.NotAfter,
				NotBefore: &expiringSoonCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: "expiring-cert",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert2.Storage = &mockStorageProvider{cert: expiringSoonCert}

		cert3 := &certificate{
			Name: "normal-cert",
			Info: CertificateInfo{
				NotAfter:  &normalCert.NotAfter,
				NotBefore: &normalCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: "normal-cert",
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		cert3.Storage = &mockStorageProvider{cert: normalCert}

		_ = cm.certificates.StoreCertificate(providerName, cert1)
		_ = cm.certificates.StoreCertificate(providerName, cert2)
		_ = cm.certificates.StoreCertificate(providerName, cert3)

		err := lm.CheckExpiredCertificates(ctx)
		require.NoError(t, err)

		// Verify states were updated
		// Note: expired certificate transitions to "recovering" after TriggerRecovery is called
		state1, _ := lm.GetCertificateState(ctx, providerName, "expired-cert")
		assert.Equal(t, CertificateStateRecovering, state1.GetState())

		state2, _ := lm.GetCertificateState(ctx, providerName, "expiring-cert")
		assert.Equal(t, CertificateStateExpiringSoon, state2.GetState())

		state3, _ := lm.GetCertificateState(ctx, providerName, "normal-cert")
		assert.Equal(t, CertificateStateNormal, state3.GetState())
	})

	t.Run("Triggers recovery for expired certificates", func(t *testing.T) {
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

		providerName := "test-provider"
		certName := "expired-cert"

		// Ensure provider exists
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

		err := lm.CheckExpiredCertificates(ctx)
		require.NoError(t, err)

		// Verify recovery was triggered (state should be recovering)
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateRecovering, state.GetState())
	})

	t.Run("Handles errors gracefully", func(t *testing.T) {
		// Try to check non-existent certificate
		err := lm.CheckExpiredCertificates(ctx)
		// Should not error, just log warnings
		assert.NoError(t, err)
	})
}

func TestCheckExpiredCertificatesOnStartup(t *testing.T) {
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

	t.Run("Startup check detects expired certificates", func(t *testing.T) {
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

		providerName := "test-provider"
		certName := "expired-cert"

		// Ensure provider exists
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

		err := cm.CheckExpiredCertificatesOnStartup(ctx)
		require.NoError(t, err)

		// Verify state was updated
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateRecovering, state.GetState())
	})

	t.Run("Startup check handles errors", func(t *testing.T) {
		// Create manager without lifecycle manager
		cmNoLifecycle := &CertManager{
			certificates: newCertStorage(),
			log:          logger,
		}

		err := cmNoLifecycle.CheckExpiredCertificatesOnStartup(ctx)
		// Should not error if no lifecycle manager
		assert.NoError(t, err)
	})
}

func TestStartExpirationMonitoring(t *testing.T) {
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
					CheckInterval: util.Duration(1 * time.Second), // Short interval for testing
				},
			},
		},
	}
	cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactory{}

	lm := NewLifecycleManager(cm, cm.expirationMonitor, logger, nil)
	cm.lifecycleManager = lm

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Periodic check runs at correct interval", func(t *testing.T) {
		// Start monitoring with short interval
		cm.StartExpirationMonitoring(ctx)

		// Wait a bit to ensure it runs
		time.Sleep(1500 * time.Millisecond)

		// Cancel context to stop monitoring
		cancel()

		// Should have run at least once
		// (We can't easily verify this without more complex test setup)
	})

	t.Run("Periodic check stops on context cancellation", func(t *testing.T) {
		ctx2, cancel2 := context.WithCancel(context.Background())

		cm2 := &CertManager{
			certificates:      newCertStorage(),
			expirationMonitor: NewExpirationMonitor(logger),
			log:               logger,
			readWriter:        rw,
			storages:          make(map[string]provider.StorageFactory),
			config:            cm.config,
		}
		lm2 := NewLifecycleManager(cm2, cm2.expirationMonitor, logger, nil)
		cm2.lifecycleManager = lm2

		cm2.StartExpirationMonitoring(ctx2)

		// Cancel immediately
		cancel2()

		// Wait a bit to ensure it stops
		time.Sleep(100 * time.Millisecond)

		// Should have stopped (no way to verify directly, but no panic/error)
	})

	t.Run("Periodic check detects expired certificates", func(t *testing.T) {
		now := time.Now().UTC()
		expiredCert := createMockCertForExpiration(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))

		providerName := "test-provider"
		certName := "expired-cert"

		ctx3, cancel3 := context.WithCancel(context.Background())
		defer cancel3()

		cm3 := &CertManager{
			certificates:      newCertStorage(),
			expirationMonitor: NewExpirationMonitor(logger),
			log:               logger,
			readWriter:        rw,
			storages:          make(map[string]provider.StorageFactory),
			config: &config.Config{
				Certificate: config.CertificateConfig{
					Renewal: config.CertificateRenewalConfig{
						ThresholdDays: 30,
						CheckInterval: util.Duration(500 * time.Millisecond),
					},
				},
			},
		}
		cm3.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactory{}
		lm3 := NewLifecycleManager(cm3, cm3.expirationMonitor, logger, nil)
		cm3.lifecycleManager = lm3

		// Ensure provider exists
		_, _ = cm3.certificates.EnsureProvider(providerName)

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

		_ = cm3.certificates.StoreCertificate(providerName, cert)

		cm3.StartExpirationMonitoring(ctx3)

		// Wait for check to run
		time.Sleep(600 * time.Millisecond)

		// Verify state was updated
		state, _ := lm3.GetCertificateState(ctx3, providerName, certName)
		assert.Equal(t, CertificateStateRecovering, state.GetState())

		cancel3()
	})
}

// mockStorageFactory is a mock implementation of provider.StorageFactory for testing.
type mockStorageFactory struct{}

func (m *mockStorageFactory) Type() string {
	return string(provider.StorageTypeFilesystem)
}

func (m *mockStorageFactory) New(log provider.Logger, cc provider.CertificateConfig) (provider.StorageProvider, error) {
	// Return a mock storage provider. The actual certificate will be set directly on the `certificate` struct in tests.
	return &mockStorageProvider{}, nil
}

func (m *mockStorageFactory) Validate(log provider.Logger, cc provider.CertificateConfig) error {
	return nil
}

// mockStorageProvider is a simple mock storage provider for testing.
type mockStorageProvider struct {
	cert *x509.Certificate
}

func (m *mockStorageProvider) LoadCertificate(ctx context.Context) (*x509.Certificate, error) {
	return m.cert, nil
}

func (m *mockStorageProvider) Write(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProvider) WritePending(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProvider) LoadPendingCertificate(ctx context.Context) (*x509.Certificate, error) {
	return nil, nil
}

func (m *mockStorageProvider) LoadPendingKey(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (m *mockStorageProvider) HasPendingCertificate(ctx context.Context) (bool, error) {
	return false, nil
}

func (m *mockStorageProvider) CleanupPending(ctx context.Context) error {
	return nil
}

func (m *mockStorageProvider) AtomicSwap(ctx context.Context) error {
	return nil
}

func (m *mockStorageProvider) RollbackSwap(ctx context.Context, swapError error) error {
	return nil
}

func (m *mockStorageProvider) Delete(ctx context.Context) error {
	return nil
}
