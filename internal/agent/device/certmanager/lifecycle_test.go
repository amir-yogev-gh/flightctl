package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertificateLifecycleState(t *testing.T) {
	t.Run("NewCertificateLifecycleState", func(t *testing.T) {
		state := NewCertificateLifecycleState(CertificateStateNormal)
		if state.GetState() != CertificateStateNormal {
			t.Errorf("expected state %v, got %v", CertificateStateNormal, state.GetState())
		}
		if state.LastChecked.IsZero() {
			t.Error("expected LastChecked to be set")
		}
	})

	t.Run("SetState", func(t *testing.T) {
		state := NewCertificateLifecycleState(CertificateStateNormal)
		oldChecked := state.LastChecked
		time.Sleep(10 * time.Millisecond) // Ensure time difference
		state.SetState(CertificateStateRenewing)
		if state.GetState() != CertificateStateRenewing {
			t.Errorf("expected state %v, got %v", CertificateStateRenewing, state.GetState())
		}
		if !state.LastChecked.After(oldChecked) {
			t.Error("expected LastChecked to be updated")
		}
	})

	t.Run("Update", func(t *testing.T) {
		state := NewCertificateLifecycleState(CertificateStateNormal)
		expiration := time.Now().Add(30 * 24 * time.Hour)
		state.Update(CertificateStateExpiringSoon, 30, &expiration)
		if state.GetState() != CertificateStateExpiringSoon {
			t.Errorf("expected state %v, got %v", CertificateStateExpiringSoon, state.GetState())
		}
		if state.DaysUntilExpiration != 30 {
			t.Errorf("expected DaysUntilExpiration 30, got %d", state.DaysUntilExpiration)
		}
		if state.ExpirationTime == nil || !state.ExpirationTime.Equal(expiration) {
			t.Error("expected ExpirationTime to be set")
		}
		if state.LastError != "" {
			t.Error("expected LastError to be cleared")
		}
	})

	t.Run("SetError", func(t *testing.T) {
		state := NewCertificateLifecycleState(CertificateStateNormal)
		err := &testError{msg: "test error"}
		state.SetError(err)
		if state.LastError != "test error" {
			t.Errorf("expected LastError 'test error', got %q", state.LastError)
		}
		state.SetError(nil)
		if state.LastError != "" {
			t.Error("expected LastError to be cleared")
		}
	})

	t.Run("Thread Safety - Concurrent Access", func(t *testing.T) {
		state := NewCertificateLifecycleState(CertificateStateNormal)
		const numGoroutines = 100
		done := make(chan bool, numGoroutines)

		// Concurrent writes
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				state.SetState(CertificateStateExpiringSoon)
				state.SetState(CertificateStateRenewing)
				state.SetState(CertificateStateNormal)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify state is valid after concurrent access
		finalState := state.GetState()
		if !finalState.IsValidState() {
			t.Errorf("final state %v is not valid after concurrent access", finalState)
		}
	})
}

func TestCertificateState_IsValidState(t *testing.T) {
	tests := []struct {
		name  string
		state CertificateState
		want  bool
	}{
		{"normal", CertificateStateNormal, true},
		{"expiring_soon", CertificateStateExpiringSoon, true},
		{"expired", CertificateStateExpired, true},
		{"renewing", CertificateStateRenewing, true},
		{"recovering", CertificateStateRecovering, true},
		{"renewal_failed", CertificateStateRenewalFailed, true},
		{"invalid", CertificateState("invalid"), false},
		{"empty", CertificateState(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsValidState(); got != tt.want {
				t.Errorf("IsValidState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateState_String(t *testing.T) {
	tests := []struct {
		name     string
		state    CertificateState
		expected string
	}{
		{"normal", CertificateStateNormal, "normal"},
		{"expiring_soon", CertificateStateExpiringSoon, "expiring_soon"},
		{"expired", CertificateStateExpired, "expired"},
		{"renewing", CertificateStateRenewing, "renewing"},
		{"recovering", CertificateStateRecovering, "recovering"},
		{"renewal_failed", CertificateStateRenewalFailed, "renewal_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLifecycleManager_stateKey(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{} // Minimal cert manager for testing
	lm := NewLifecycleManager(cm, monitor, logger, nil)

	key := lm.stateKey("provider1", "cert1")
	expected := "provider1/cert1"
	if key != expected {
		t.Errorf("stateKey() = %v, want %v", key, expected)
	}
}

// createTestCertForLifecycle creates a test certificate for lifecycle testing
func createTestCertForLifecycle(t *testing.T, notBefore, notAfter time.Time) *x509.Certificate {
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
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

// mockStorageProviderForLifecycle is a mock storage provider for lifecycle testing
type mockStorageProviderForLifecycle struct {
	cert *x509.Certificate
}

func (m *mockStorageProviderForLifecycle) LoadCertificate(ctx context.Context) (*x509.Certificate, error) {
	if m.cert == nil {
		return nil, fmt.Errorf("certificate is nil")
	}
	return m.cert, nil
}

func (m *mockStorageProviderForLifecycle) Write(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProviderForLifecycle) WritePending(cert *x509.Certificate, keyPEM []byte) error {
	return nil
}

func (m *mockStorageProviderForLifecycle) LoadPendingCertificate(ctx context.Context) (*x509.Certificate, error) {
	return nil, nil
}

func (m *mockStorageProviderForLifecycle) LoadPendingKey(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (m *mockStorageProviderForLifecycle) HasPendingCertificate(ctx context.Context) (bool, error) {
	return false, nil
}

func (m *mockStorageProviderForLifecycle) CleanupPending(ctx context.Context) error {
	return nil
}

func (m *mockStorageProviderForLifecycle) AtomicSwap(ctx context.Context) error {
	return nil
}

func (m *mockStorageProviderForLifecycle) RollbackSwap(ctx context.Context, swapError error) error {
	return nil
}

func (m *mockStorageProviderForLifecycle) Delete(ctx context.Context) error {
	return nil
}

// mockStorageFactoryForLifecycle is a mock storage factory for lifecycle testing
type mockStorageFactoryForLifecycle struct {
	certMap map[string]*x509.Certificate
}

func (m *mockStorageFactoryForLifecycle) Type() string {
	return string(provider.StorageTypeFilesystem)
}

func (m *mockStorageFactoryForLifecycle) New(log provider.Logger, cc provider.CertificateConfig) (provider.StorageProvider, error) {
	cert := m.certMap[cc.Name]
	return &mockStorageProviderForLifecycle{cert: cert}, nil
}

func (m *mockStorageFactoryForLifecycle) Validate(log provider.Logger, cc provider.CertificateConfig) error {
	return nil
}

func TestLifecycleManager_CheckRenewal(t *testing.T) {
	tmpDir := t.TempDir()
	rw := fileio.NewReadWriter(fileio.WithTestRootDir(tmpDir))
	logger := log.NewPrefixLogger("test")
	ctx := context.Background()

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
	// Create factory with certificate map
	certMap := make(map[string]*x509.Certificate)
	cm.storages[string(provider.StorageTypeFilesystem)] = &mockStorageFactoryForLifecycle{certMap: certMap}

	lm := NewLifecycleManager(cm, cm.expirationMonitor, logger, nil)
	cm.lifecycleManager = lm

	now := time.Now().UTC()

	t.Run("Returns true for expired certificate", func(t *testing.T) {
		expiredCert := createTestCertForLifecycle(t, now.Add(-48*time.Hour), now.Add(-24*time.Hour))
		providerName := "test-provider"
		certName := "expired-cert"

		// Store certificate in factory's cert map
		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = expiredCert

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
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		assert.Less(t, days, 0, "days should be negative for expired certificate")

		// Verify state was updated
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateExpired, state.GetState())
	})

	t.Run("Returns true for expiring soon certificate", func(t *testing.T) {
		expiringSoonCert := createTestCertForLifecycle(t, now.Add(-24*time.Hour), now.Add(7*24*time.Hour))
		providerName := "test-provider"
		certName := "expiring-cert"

		// Store certificate in factory's cert map
		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = expiringSoonCert

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
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		assert.Greater(t, days, 0)
		assert.LessOrEqual(t, days, 30)

		// Verify state was updated
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())
	})

	t.Run("Returns false for normal certificate", func(t *testing.T) {
		normalCert := createTestCertForLifecycle(t, now.Add(-24*time.Hour), now.Add(60*24*time.Hour))
		providerName := "test-provider"
		certName := "normal-cert"

		// Store certificate in factory's cert map
		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = normalCert

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
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.False(t, needsRenewal)
		assert.Greater(t, days, 30)

		// Verify state was updated
		state, _ := lm.GetCertificateState(ctx, providerName, certName)
		assert.Equal(t, CertificateStateNormal, state.GetState())
	})

	t.Run("Returns error for negative threshold", func(t *testing.T) {
		needsRenewal, days, err := lm.CheckRenewal(ctx, "provider", "cert", -1)
		assert.Error(t, err)
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)
		assert.Contains(t, err.Error(), "threshold days must be non-negative")
	})

	t.Run("Returns error for missing certificate", func(t *testing.T) {
		needsRenewal, days, err := lm.CheckRenewal(ctx, "nonexistent", "cert", 30)
		assert.Error(t, err)
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)
		assert.Contains(t, err.Error(), "failed to read certificate")
	})

	t.Run("Certificate needs renewal - within threshold", func(t *testing.T) {
		// Certificate expiring in 25 days, threshold = 30
		expiringCert := createTestCertForLifecycle(t, now.Add(-24*time.Hour), now.Add(25*24*time.Hour))
		providerName := "test-provider"
		certName := "within-threshold-cert"

		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = expiringCert

		_, _ = cm.certificates.EnsureProvider(providerName)
		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiringCert.NotAfter,
				NotBefore: &expiringCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		assert.Greater(t, days, 0)
		assert.LessOrEqual(t, days, 30)
	})

	t.Run("Certificate does not need renewal - beyond threshold", func(t *testing.T) {
		// Certificate expiring in 35 days, threshold = 30
		normalCert := createTestCertForLifecycle(t, now.Add(-24*time.Hour), now.Add(35*24*time.Hour))
		providerName := "test-provider"
		certName := "beyond-threshold-cert"

		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = normalCert

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
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.False(t, needsRenewal)
		assert.Greater(t, days, 30)
	})

	t.Run("Certificate needs renewal - exactly at threshold", func(t *testing.T) {
		// Certificate expiring in exactly 30 days, threshold = 30
		expiringCert := createTestCertForLifecycle(t, now.Add(-24*time.Hour), now.Add(30*24*time.Hour+1*time.Hour))
		providerName := "test-provider"
		certName := "at-threshold-cert"

		factory := cm.storages[string(provider.StorageTypeFilesystem)].(*mockStorageFactoryForLifecycle)
		factory.certMap[certName] = expiringCert

		_, _ = cm.certificates.EnsureProvider(providerName)
		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{
				NotAfter:  &expiringCert.NotAfter,
				NotBefore: &expiringCert.NotBefore,
			},
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		require.NoError(t, err)
		assert.True(t, needsRenewal)
		// Days should be approximately 30 (allow tolerance)
		assert.GreaterOrEqual(t, days, 29)
		assert.LessOrEqual(t, days, 31)
	})

	t.Run("Certificate with no expiration info", func(t *testing.T) {
		providerName := "test-provider"
		certName := "no-expiration-cert"

		_, _ = cm.certificates.EnsureProvider(providerName)
		cert := &certificate{
			Name: certName,
			Info: CertificateInfo{}, // No expiration info
			Config: provider.CertificateConfig{
				Name: certName,
				Storage: provider.StorageConfig{
					Type: provider.StorageTypeFilesystem,
				},
			},
		}
		_ = cm.certificates.StoreCertificate(providerName, cert)

		needsRenewal, days, err := lm.CheckRenewal(ctx, providerName, certName, 30)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no expiration date")
		assert.False(t, needsRenewal)
		assert.Equal(t, 0, days)
	})
}

func TestLifecycleManager_GetCertificateState(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Returns default state for new certificate", func(t *testing.T) {
		state, err := lm.GetCertificateState(ctx, "provider", "new-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateNormal, state.GetState())
	})

	t.Run("Returns stored state", func(t *testing.T) {
		// Set a state first
		err := lm.SetCertificateState(ctx, "provider", "cert", CertificateStateRenewing)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("Normal state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "normal-cert", CertificateStateNormal)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "normal-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateNormal, state.GetState())
	})

	t.Run("Expiring soon state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "expiring-cert", CertificateStateExpiringSoon)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "expiring-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())
	})

	t.Run("Expired state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "expired-cert", CertificateStateExpired)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "expired-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateExpired, state.GetState())
	})

	t.Run("Renewing state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "renewing-cert", CertificateStateRenewing)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "renewing-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("Recovering state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "recovering-cert", CertificateStateRecovering)
		require.NoError(t, err)

		state, err := lm.GetCertificateState(ctx, "provider", "recovering-cert")
		require.NoError(t, err)
		assert.Equal(t, CertificateStateRecovering, state.GetState())
	})
}

func TestLifecycleManager_SetCertificateState(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Sets state successfully", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "cert", CertificateStateRenewing)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateRenewing, state.GetState())
	})

	t.Run("Rejects invalid state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "cert", CertificateState("invalid"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid certificate state")
	})

	t.Run("Updates existing state", func(t *testing.T) {
		err := lm.SetCertificateState(ctx, "provider", "cert2", CertificateStateNormal)
		require.NoError(t, err)

		err = lm.SetCertificateState(ctx, "provider", "cert2", CertificateStateExpiringSoon)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert2")
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())
	})
}

func TestLifecycleManager_UpdateCertificateState(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Updates state with full information", func(t *testing.T) {
		expiration := time.Now().Add(30 * 24 * time.Hour)
		err := lm.UpdateCertificateState(ctx, "provider", "cert", CertificateStateExpiringSoon, 30, &expiration)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())
		assert.Equal(t, 30, state.DaysUntilExpiration)
		assert.NotNil(t, state.ExpirationTime)
		assert.Equal(t, expiration, *state.ExpirationTime)
	})

	t.Run("Rejects invalid state", func(t *testing.T) {
		expiration := time.Now().Add(30 * 24 * time.Hour)
		err := lm.UpdateCertificateState(ctx, "provider", "cert", CertificateState("invalid"), 30, &expiration)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid certificate state")
	})

	t.Run("Update state with nil expiration time", func(t *testing.T) {
		err := lm.UpdateCertificateState(ctx, "provider", "nil-exp-cert", CertificateStateNormal, 0, nil)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "nil-exp-cert")
		assert.Equal(t, CertificateStateNormal, state.GetState())
		assert.Nil(t, state.ExpirationTime)
	})

	t.Run("Update state clears error", func(t *testing.T) {
		// Set state with error first
		err := lm.SetCertificateState(ctx, "provider", "error-cert", CertificateStateRenewing)
		require.NoError(t, err)
		testErr := &testError{msg: "test error"}
		err = lm.RecordError(ctx, "provider", "error-cert", testErr)
		require.NoError(t, err)

		// Update state - should clear error
		expiration := time.Now().Add(30 * 24 * time.Hour)
		err = lm.UpdateCertificateState(ctx, "provider", "error-cert", CertificateStateNormal, 30, &expiration)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "error-cert")
		assert.Equal(t, CertificateStateNormal, state.GetState())
		assert.Empty(t, state.LastError)
	})
}

func TestLifecycleManager_RecordError(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Records error successfully", func(t *testing.T) {
		// Set state to renewing first
		err := lm.SetCertificateState(ctx, "provider", "cert", CertificateStateRenewing)
		require.NoError(t, err)

		// Record an error
		testErr := &testError{msg: "renewal failed"}
		err = lm.RecordError(ctx, "provider", "cert", testErr)
		require.NoError(t, err)

		// Verify error was recorded and state changed to renewal_failed
		state, _ := lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateRenewalFailed, state.GetState())
		assert.Contains(t, state.LastError, "renewal failed")
	})

	t.Run("Clears error when nil", func(t *testing.T) {
		// Set state and error first
		err := lm.SetCertificateState(ctx, "provider", "cert2", CertificateStateRenewing)
		require.NoError(t, err)

		testErr := &testError{msg: "test error"}
		err = lm.RecordError(ctx, "provider", "cert2", testErr)
		require.NoError(t, err)

		// Clear error
		err = lm.RecordError(ctx, "provider", "cert2", nil)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert2")
		assert.Empty(t, state.LastError)
	})

	t.Run("Record error creates lifecycle if nil", func(t *testing.T) {
		// Record error for certificate without lifecycle state
		testErr := &testError{msg: "new error"}
		err := lm.RecordError(ctx, "provider", "new-cert", testErr)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "new-cert")
		assert.NotNil(t, state)
		// When creating a new state, it starts as Normal and only becomes RenewalFailed if it was Renewing
		// Since this is a new state, it will be Normal with the error recorded
		assert.Equal(t, CertificateStateNormal, state.GetState())
		assert.Contains(t, state.LastError, "new error")
	})
}

func TestLifecycleManager_StateTransitions(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Normal to expiring_soon to renewing to normal", func(t *testing.T) {
		// Start with normal
		err := lm.SetCertificateState(ctx, "provider", "cert", CertificateStateNormal)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateNormal, state.GetState())

		// Transition to expiring_soon
		err = lm.SetCertificateState(ctx, "provider", "cert", CertificateStateExpiringSoon)
		require.NoError(t, err)

		state, _ = lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateExpiringSoon, state.GetState())

		// Transition to renewing
		err = lm.SetCertificateState(ctx, "provider", "cert", CertificateStateRenewing)
		require.NoError(t, err)

		state, _ = lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateRenewing, state.GetState())

		// Transition back to normal
		err = lm.SetCertificateState(ctx, "provider", "cert", CertificateStateNormal)
		require.NoError(t, err)

		state, _ = lm.GetCertificateState(ctx, "provider", "cert")
		assert.Equal(t, CertificateStateNormal, state.GetState())
	})

	t.Run("Renewing to renewal_failed on error", func(t *testing.T) {
		// Start with renewing
		err := lm.SetCertificateState(ctx, "provider", "cert2", CertificateStateRenewing)
		require.NoError(t, err)

		// Record error - should transition to renewal_failed
		testErr := &testError{msg: "renewal error"}
		err = lm.RecordError(ctx, "provider", "cert2", testErr)
		require.NoError(t, err)

		state, _ := lm.GetCertificateState(ctx, "provider", "cert2")
		assert.Equal(t, CertificateStateRenewalFailed, state.GetState())
	})
}

func TestLifecycleManager_ConcurrentStateUpdates(t *testing.T) {
	logger := log.NewPrefixLogger("test")
	monitor := NewExpirationMonitor(logger)
	cm := &CertManager{}
	lm := NewLifecycleManager(cm, monitor, logger, nil)
	ctx := context.Background()

	t.Run("Concurrent state updates", func(t *testing.T) {
		const numGoroutines = 50
		done := make(chan bool, numGoroutines)

		// Concurrently update state from multiple goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				state := CertificateStateNormal
				switch idx % 4 {
				case 0:
					state = CertificateStateNormal
				case 1:
					state = CertificateStateExpiringSoon
				case 2:
					state = CertificateStateRenewing
				case 3:
					state = CertificateStateExpired
				}
				_ = lm.SetCertificateState(ctx, "provider", "concurrent-cert", state)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify state is valid after concurrent access
		state, err := lm.GetCertificateState(ctx, "provider", "concurrent-cert")
		require.NoError(t, err)
		assert.True(t, state.GetState().IsValidState())
	})
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
