package certmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	api "github.com/flightctl/flightctl/api/v1beta1"
	"github.com/flightctl/flightctl/internal/agent/client"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/identity"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/samber/lo"
)

// BootstrapCertificateChecker defines the interface for checking bootstrap certificate availability.
// This interface is used to avoid import cycles between certmanager and device packages.
type BootstrapCertificateChecker interface {
	// HasValidBootstrapCertificate checks if a valid bootstrap certificate exists.
	HasValidBootstrapCertificate(ctx context.Context) (bool, error)
	// GetCertificateForAuth returns the appropriate certificate for authentication.
	// Falls back to bootstrap certificate if management certificate is expired.
	GetCertificateForAuth(ctx context.Context, managementCertPath, managementKeyPath string) (*tls.Certificate, error)
}

// CertificateState represents the current lifecycle state of a certificate.
type CertificateState string

const (
	// CertificateStateNormal indicates the certificate is valid and not expiring soon
	CertificateStateNormal CertificateState = "normal"

	// CertificateStateExpiringSoon indicates the certificate is expiring within the threshold
	CertificateStateExpiringSoon CertificateState = "expiring_soon"

	// CertificateStateExpired indicates the certificate has expired
	CertificateStateExpired CertificateState = "expired"

	// CertificateStateRenewing indicates the certificate renewal is in progress
	CertificateStateRenewing CertificateState = "renewing"

	// CertificateStateRecovering indicates expired certificate recovery is in progress
	CertificateStateRecovering CertificateState = "recovering"

	// CertificateStateRenewalFailed indicates the last renewal attempt failed
	CertificateStateRenewalFailed CertificateState = "renewal_failed"
)

// RecoveryAuthMethod represents the authentication method for recovery.
type RecoveryAuthMethod string

const (
	// RecoveryAuthMethodBootstrap uses bootstrap certificate for authentication
	RecoveryAuthMethodBootstrap RecoveryAuthMethod = "bootstrap"
	// RecoveryAuthMethodTPM uses TPM attestation for authentication
	RecoveryAuthMethodTPM RecoveryAuthMethod = "tpm"
)

// String returns the string representation of the certificate state.
func (s CertificateState) String() string {
	return string(s)
}

// IsValidState checks if the state is a valid certificate state.
func (s CertificateState) IsValidState() bool {
	switch s {
	case CertificateStateNormal,
		CertificateStateExpiringSoon,
		CertificateStateExpired,
		CertificateStateRenewing,
		CertificateStateRecovering,
		CertificateStateRenewalFailed:
		return true
	default:
		return false
	}
}

// CertificateLifecycleState holds the lifecycle state and related information for a certificate.
type CertificateLifecycleState struct {
	// State is the current lifecycle state
	State CertificateState `json:"state"`

	// DaysUntilExpiration is the number of days until expiration (negative if expired)
	DaysUntilExpiration int `json:"days_until_expiration,omitempty"`

	// ExpirationTime is when the certificate expires
	ExpirationTime *time.Time `json:"expiration_time,omitempty"`

	// LastChecked is when the state was last checked
	LastChecked time.Time `json:"last_checked,omitempty"`

	// LastError is the last error encountered during lifecycle operations
	LastError string `json:"last_error,omitempty"`

	// Mutex for thread-safe access
	mu sync.RWMutex `json:"-"`
}

// NewCertificateLifecycleState creates a new lifecycle state with the given state.
func NewCertificateLifecycleState(state CertificateState) *CertificateLifecycleState {
	return &CertificateLifecycleState{
		State:       state,
		LastChecked: time.Now().UTC(),
	}
}

// GetState returns the current state (thread-safe).
func (cls *CertificateLifecycleState) GetState() CertificateState {
	cls.mu.RLock()
	defer cls.mu.RUnlock()
	return cls.State
}

// SetState updates the state (thread-safe).
func (cls *CertificateLifecycleState) SetState(state CertificateState) {
	cls.mu.Lock()
	defer cls.mu.Unlock()
	cls.State = state
	cls.LastChecked = time.Now().UTC()
}

// Update updates the lifecycle state with new information.
func (cls *CertificateLifecycleState) Update(state CertificateState, daysUntilExpiration int, expirationTime *time.Time) {
	cls.mu.Lock()
	defer cls.mu.Unlock()
	cls.State = state
	cls.DaysUntilExpiration = daysUntilExpiration
	cls.ExpirationTime = expirationTime
	cls.LastChecked = time.Now().UTC()
	cls.LastError = "" // Clear error on successful update
}

// SetError records an error in the lifecycle state.
func (cls *CertificateLifecycleState) SetError(err error) {
	cls.mu.Lock()
	defer cls.mu.Unlock()
	if err != nil {
		cls.LastError = err.Error()
	} else {
		cls.LastError = ""
	}
}

// CertificateLifecycleManager defines the interface for managing certificate lifecycle.
type CertificateLifecycleManager interface {
	// CheckRenewal checks if a certificate needs renewal based on expiration threshold.
	// Returns true if renewal is needed, days until expiration, and any error.
	CheckRenewal(ctx context.Context, providerName, certName string, thresholdDays int) (bool, int, error)

	// GetCertificateState returns the current lifecycle state of a certificate.
	GetCertificateState(ctx context.Context, providerName, certName string) (*CertificateLifecycleState, error)

	// SetCertificateState updates the lifecycle state of a certificate.
	SetCertificateState(ctx context.Context, providerName, certName string, state CertificateState) error

	// UpdateCertificateState updates the lifecycle state with full information.
	UpdateCertificateState(ctx context.Context, providerName, certName string, state CertificateState, daysUntilExpiration int, expirationTime *time.Time) error

	// RecordError records an error for a certificate's lifecycle operations.
	RecordError(ctx context.Context, providerName, certName string, err error) error
}

// LifecycleManager implements CertificateLifecycleManager.
// It coordinates certificate lifecycle operations and state management.
type LifecycleManager struct {
	// certManager is the certificate manager to interact with
	certManager *CertManager

	// expirationMonitor is used to check certificate expiration
	expirationMonitor *ExpirationMonitor

	// bootstrapHandler handles bootstrap certificate operations for recovery
	bootstrapHandler BootstrapCertificateChecker

	// managementClient is used to submit CSRs and poll for certificates
	managementClient client.Management

	// identityProvider is used to generate CSRs and get keys for recovery
	identityProvider identity.Provider

	// tpmRenewalProvider is used to generate TPM attestation for recovery
	tpmRenewalProvider *identity.TPMRenewalProvider

	// managementCertPath is the path to the management certificate
	managementCertPath string

	// managementKeyPath is the path to the management key
	managementKeyPath string

	// lifecycleStates tracks the lifecycle state for each certificate
	// Key format: "providerName/certName"
	lifecycleStates map[string]*CertificateLifecycleState

	// Mutex for thread-safe access to lifecycle states
	mu sync.RWMutex

	// Logger for lifecycle operations
	log provider.Logger
}

// NewLifecycleManager creates a new lifecycle manager.
func NewLifecycleManager(certManager *CertManager, expirationMonitor *ExpirationMonitor, log provider.Logger, bootstrapHandler BootstrapCertificateChecker) *LifecycleManager {
	return &LifecycleManager{
		certManager:       certManager,
		expirationMonitor: expirationMonitor,
		bootstrapHandler:  bootstrapHandler,
		lifecycleStates:   make(map[string]*CertificateLifecycleState),
		log:               log,
	}
}

// SetManagementClient sets the management client for recovery operations.
func (lm *LifecycleManager) SetManagementClient(client client.Management) {
	lm.managementClient = client
}

// SetIdentityProvider sets the identity provider for CSR generation.
func (lm *LifecycleManager) SetIdentityProvider(provider identity.Provider) {
	lm.identityProvider = provider
	// Try to get TPM renewal provider from identity provider if it's a TPM provider
	if tpmProvider, ok := provider.(interface {
		GenerateRenewalAttestation(ctx context.Context) (*identity.RenewalAttestation, error)
	}); ok {
		// If the provider has GenerateRenewalAttestation, we can use it
		// For now, we'll need to access it differently
		_ = tpmProvider
	}
}

// SetTPMRenewalProvider sets the TPM renewal provider for recovery operations.
func (lm *LifecycleManager) SetTPMRenewalProvider(provider *identity.TPMRenewalProvider) {
	lm.tpmRenewalProvider = provider
}

// SetManagementCertPaths sets the paths to management certificate and key.
func (lm *LifecycleManager) SetManagementCertPaths(certPath, keyPath string) {
	lm.managementCertPath = certPath
	lm.managementKeyPath = keyPath
}

// stateKey generates a key for the lifecycle state map.
func (lm *LifecycleManager) stateKey(providerName, certName string) string {
	return fmt.Sprintf("%s/%s", providerName, certName)
}

// CheckRenewal checks if a certificate needs renewal based on expiration threshold.
func (lm *LifecycleManager) CheckRenewal(ctx context.Context, providerName, certName string, thresholdDays int) (bool, int, error) {
	if thresholdDays < 0 {
		return false, 0, fmt.Errorf("threshold days must be non-negative, got %d", thresholdDays)
	}

	// Get certificate from cert manager
	cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return false, 0, fmt.Errorf("failed to read certificate: %w", err)
	}

	cert.mu.RLock()
	certInfo := cert.Info
	cert.mu.RUnlock()

	// Check if certificate has expiration info
	if certInfo.NotAfter == nil {
		lm.log.Debugf("Certificate %q/%q has no expiration info, cannot check renewal", providerName, certName)
		return false, 0, fmt.Errorf("certificate has no expiration date")
	}

	// Load certificate from storage to get full X.509 certificate
	storage, err := lm.certManager.initStorageProvider(cert.Config)
	if err != nil {
		return false, 0, fmt.Errorf("failed to init storage: %w", err)
	}

	x509Cert, err := storage.LoadCertificate(ctx)
	if err != nil {
		return false, 0, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Calculate days until expiration
	days, err := lm.expirationMonitor.CalculateDaysUntilExpiration(x509Cert)
	if err != nil {
		return false, 0, fmt.Errorf("failed to calculate days until expiration: %w", err)
	}

	// Check if expired
	isExpired, err := lm.expirationMonitor.IsExpired(x509Cert)
	if err != nil {
		return false, days, fmt.Errorf("failed to check if expired: %w", err)
	}

	if isExpired {
		lm.log.Debugf("Certificate %q/%q is expired (%d days ago)", providerName, certName, -days)
		// Update state to expired
		_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpired, days, certInfo.NotAfter)
		return true, days, nil // Expired certificates need renewal (recovery)
	}

	// Check if expiring soon
	isExpiringSoon, err := lm.expirationMonitor.IsExpiringSoon(x509Cert, thresholdDays)
	if err != nil {
		return false, days, fmt.Errorf("failed to check if expiring soon: %w", err)
	}

	if isExpiringSoon {
		lm.log.Debugf("Certificate %q/%q is expiring soon (%d days until expiration)", providerName, certName, days)
		// Update state to expiring_soon
		_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpiringSoon, days, certInfo.NotAfter)
		return true, days, nil // Needs renewal
	}

	// Certificate is normal (not expiring soon)
	_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateNormal, days, certInfo.NotAfter)
	return false, days, nil
}

// GetCertificateState returns the current lifecycle state of a certificate.
func (lm *LifecycleManager) GetCertificateState(ctx context.Context, providerName, certName string) (*CertificateLifecycleState, error) {
	key := lm.stateKey(providerName, certName)

	lm.mu.RLock()
	state, exists := lm.lifecycleStates[key]
	lm.mu.RUnlock()

	if !exists {
		// Return default state if not tracked yet
		return NewCertificateLifecycleState(CertificateStateNormal), nil
	}

	// Return a copy to avoid race conditions
	state.mu.RLock()
	copy := &CertificateLifecycleState{
		State:               state.State,
		DaysUntilExpiration: state.DaysUntilExpiration,
		ExpirationTime:      state.ExpirationTime,
		LastChecked:         state.LastChecked,
		LastError:           state.LastError,
	}
	state.mu.RUnlock()

	return copy, nil
}

// SetCertificateState updates the lifecycle state of a certificate.
func (lm *LifecycleManager) SetCertificateState(ctx context.Context, providerName, certName string, newState CertificateState) error {
	if !newState.IsValidState() {
		return fmt.Errorf("invalid certificate state: %s", newState)
	}

	key := lm.stateKey(providerName, certName)

	lm.mu.Lock()
	state, exists := lm.lifecycleStates[key]
	if !exists {
		state = NewCertificateLifecycleState(newState)
		lm.lifecycleStates[key] = state
	} else {
		state.SetState(newState)
	}
	lm.mu.Unlock()

	lm.log.Debugf("Certificate %q/%q state updated to %s", providerName, certName, newState)
	return nil
}

// UpdateCertificateState updates the lifecycle state with full information.
func (lm *LifecycleManager) UpdateCertificateState(ctx context.Context, providerName, certName string, state CertificateState, daysUntilExpiration int, expirationTime *time.Time) error {
	if !state.IsValidState() {
		return fmt.Errorf("invalid certificate state: %s", state)
	}

	key := lm.stateKey(providerName, certName)

	lm.mu.Lock()
	lifecycleState, exists := lm.lifecycleStates[key]
	if !exists {
		lifecycleState = NewCertificateLifecycleState(state)
		lm.lifecycleStates[key] = lifecycleState
	}
	lifecycleState.Update(state, daysUntilExpiration, expirationTime)
	lm.mu.Unlock()

	lm.log.Debugf("Certificate %q/%q state updated: %s (expires in %d days)", providerName, certName, state, daysUntilExpiration)
	return nil
}

// RecordError records an error for a certificate's lifecycle operations.
func (lm *LifecycleManager) RecordError(ctx context.Context, providerName, certName string, err error) error {
	key := lm.stateKey(providerName, certName)

	lm.mu.Lock()
	state, exists := lm.lifecycleStates[key]
	if !exists {
		state = NewCertificateLifecycleState(CertificateStateNormal)
		lm.lifecycleStates[key] = state
	}
	state.SetError(err)

	// Update state to renewal_failed if currently renewing
	if state.GetState() == CertificateStateRenewing {
		state.SetState(CertificateStateRenewalFailed)
	}
	lm.mu.Unlock()

	if err != nil {
		lm.log.Warnf("Certificate %q/%q lifecycle error: %v", providerName, certName, err)
	}

	return nil
}

// GetCertificateStatus returns certificate status information for status reporting.
func (lm *LifecycleManager) GetCertificateStatus(ctx context.Context, providerName string, certName string) (*CertificateLifecycleState, int, *time.Time, error) {
	// Get certificate state
	state, err := lm.GetCertificateState(ctx, providerName, certName)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get certificate state: %w", err)
	}

	// Get certificate to access expiration info
	cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return state, 0, nil, nil // Return state even if cert not found
	}

	cert.mu.RLock()
	certInfo := cert.Info
	cert.mu.RUnlock()

	var days int
	if state != nil {
		days = state.DaysUntilExpiration
	}

	return state, days, certInfo.NotAfter, nil
}

// DetectExpiredCertificate detects if a certificate is expired or expiring soon.
// Returns the expiration state and days until expiration (negative if expired).
func (lm *LifecycleManager) DetectExpiredCertificate(ctx context.Context, providerName string, certName string) (CertificateState, int, error) {
	// Get certificate expiration info
	cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return CertificateStateNormal, 0, fmt.Errorf("failed to read certificate: %w", err)
	}

	cert.mu.RLock()
	certInfo := cert.Info
	cert.mu.RUnlock()

	// Check if certificate has expiration info
	if certInfo.NotAfter == nil {
		// No expiration info - assume normal (may be initial provisioning)
		return CertificateStateNormal, 0, nil
	}

	// Load certificate from storage to get full X.509 certificate
	// Use existing storage if available, otherwise initialize new one
	var storage provider.StorageProvider
	if cert.Storage != nil {
		storage = cert.Storage
	} else {
		var err error
		storage, err = lm.certManager.initStorageProvider(cert.Config)
		if err != nil {
			return CertificateStateNormal, 0, fmt.Errorf("failed to init storage: %w", err)
		}
	}

	x509Cert, err := storage.LoadCertificate(ctx)
	if err != nil {
		return CertificateStateNormal, 0, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Calculate days until expiration
	days, err := lm.expirationMonitor.CalculateDaysUntilExpiration(x509Cert)
	if err != nil {
		return CertificateStateNormal, 0, fmt.Errorf("failed to calculate days until expiration: %w", err)
	}

	// Check if expired
	isExpired, err := lm.expirationMonitor.IsExpired(x509Cert)
	if err != nil {
		return CertificateStateNormal, days, fmt.Errorf("failed to check if expired: %w", err)
	}

	if isExpired {
		// Certificate is expired
		_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpired, days, certInfo.NotAfter)
		return CertificateStateExpired, days, nil
	}

	// Check if expiring soon (within threshold)
	thresholdDays := 30 // Default threshold
	// Try to get threshold from config if available
	if lm.certManager.config != nil {
		thresholdDays = lm.certManager.config.Certificate.Renewal.ThresholdDays
		if thresholdDays == 0 {
			thresholdDays = 30
		}
	}

	isExpiringSoon, err := lm.expirationMonitor.IsExpiringSoon(x509Cert, thresholdDays)
	if err != nil {
		return CertificateStateNormal, days, fmt.Errorf("failed to check if expiring soon: %w", err)
	}

	if isExpiringSoon {
		// Certificate is expiring soon
		_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateExpiringSoon, days, certInfo.NotAfter)
		return CertificateStateExpiringSoon, days, nil
	}

	// Certificate is normal
	_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateNormal, days, certInfo.NotAfter)
	return CertificateStateNormal, days, nil
}

// CheckExpiredCertificates checks all managed certificates for expiration.
// This should be called periodically to detect expired certificates.
func (lm *LifecycleManager) CheckExpiredCertificates(ctx context.Context) error {
	lm.log.Debug("Checking for expired certificates")

	// Get all providers
	providers, err := lm.certManager.certificates.ListProviderNames()
	if err != nil {
		return fmt.Errorf("failed to list providers: %w", err)
	}

	// Iterate through all certificates
	for _, providerName := range providers {
		certs, err := lm.certManager.certificates.ReadCertificates(providerName)
		if err != nil {
			lm.log.Warnf("Failed to read certificates for provider %q: %v", providerName, err)
			continue
		}

		for _, cert := range certs {
			state, days, err := lm.DetectExpiredCertificate(ctx, providerName, cert.Name)
			if err != nil {
				lm.log.Warnf("Failed to detect expiration for certificate %q/%q: %v", providerName, cert.Name, err)
				continue
			}

			// Get current state to check if it changed
			currentState, err := lm.GetCertificateState(ctx, providerName, cert.Name)
			if err != nil {
				lm.log.Warnf("Failed to get current state for certificate %q/%q: %v", providerName, cert.Name, err)
				continue
			}

			// Update state if it changed
			if currentState.GetState() != state {
				expirationTime := cert.Info.NotAfter
				if err := lm.UpdateCertificateState(ctx, providerName, cert.Name, state, days, expirationTime); err != nil {
					lm.log.Warnf("Failed to update certificate state: %v", err)
				}
			}

			switch state {
			case CertificateStateExpired:
				lm.log.Warnf("Certificate %q/%q is expired (%d days ago), triggering recovery", providerName, cert.Name, -days)
				// Trigger recovery for expired certificates
				if err := lm.TriggerRecovery(ctx, providerName, cert.Name); err != nil {
					lm.log.Errorf("Failed to trigger recovery for expired certificate %q/%q: %v", providerName, cert.Name, err)
				}
			case CertificateStateExpiringSoon:
				lm.log.Infof("Certificate %q/%q is expiring soon (%d days until expiration)", providerName, cert.Name, days)
				// Renewal will be triggered by existing renewal logic
			case CertificateStateNormal:
				lm.log.Debugf("Certificate %q/%q is normal (%d days until expiration)", providerName, cert.Name, days)
			}

			// Update expiration metrics if metrics collector is available
			// Note: metricsCollector is accessed through certManager
			if lm.certManager.metricsCollector != nil && cert.Info.NotAfter != nil {
				lm.certManager.metricsCollector.RecordCertificateExpiration("management", cert.Name, *cert.Info.NotAfter, days)
			}
		}
	}

	return nil
}

// determineRecoveryAuthMethod determines which authentication method to use for recovery.
func (lm *LifecycleManager) determineRecoveryAuthMethod(ctx context.Context) (RecoveryAuthMethod, error) {
	// Step 1: Try bootstrap certificate first
	if lm.bootstrapHandler != nil {
		hasBootstrap, err := lm.bootstrapHandler.HasValidBootstrapCertificate(ctx)
		if err == nil && hasBootstrap {
			lm.log.Debug("Bootstrap certificate available, using for recovery")
			return RecoveryAuthMethodBootstrap, nil
		}
		if err != nil {
			lm.log.Warnf("Failed to check bootstrap certificate: %v", err)
		}
	}

	// Step 2: Fall back to TPM attestation
	if lm.tpmRenewalProvider != nil {
		lm.log.Debug("Bootstrap certificate not available, using TPM attestation for recovery")
		return RecoveryAuthMethodTPM, nil
	}

	return "", fmt.Errorf("no authentication method available for recovery (bootstrap expired and no TPM)")
}

// TriggerRecovery triggers recovery for an expired certificate.
func (lm *LifecycleManager) TriggerRecovery(ctx context.Context, providerName string, certName string) error {
	lm.log.Infof("Triggering recovery for expired certificate %q/%q", providerName, certName)

	// Update state to recovering
	if err := lm.SetCertificateState(ctx, providerName, certName, CertificateStateRecovering); err != nil {
		lm.log.Warnf("Failed to set recovery state: %v", err)
		// Continue anyway
	}

	// Execute recovery flow with retry logic
	maxRetries := 3
	baseDelay := 1 * time.Minute

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			lm.log.Infof("Retrying recovery (attempt %d/%d)", attempt+1, maxRetries)
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue retry
			}
		}

		err := lm.RecoverExpiredCertificate(ctx, providerName, certName)
		if err == nil {
			// Recovery successful
			return nil
		}

		lm.log.Warnf("Recovery attempt %d/%d failed: %v", attempt+1, maxRetries, err)

		// If this is the last attempt, return error
		if attempt == maxRetries-1 {
			lm.RecordError(ctx, providerName, certName, err)
			return fmt.Errorf("recovery failed after %d attempts: %w", maxRetries, err)
		}
	}

	return nil
}

// RecoverExpiredCertificate implements the complete recovery flow for expired certificates.
func (lm *LifecycleManager) RecoverExpiredCertificate(ctx context.Context, providerName string, certName string) error {
	startTime := time.Now()

	// Log recovery start
	logCtx := CertificateLogContext{
		Operation:       "recovery",
		CertificateType: "management",
		CertificateName: certName,
		DeviceName:      certName, // Use certificate name as device identifier
		Reason:          "expired",
	}
	LogCertificateOperation(lm.log, logCtx)

	// Record recovery attempt
	if lm.certManager.metricsCollector != nil {
		lm.certManager.metricsCollector.RecordRecoveryAttempt("management", certName)
	}

	// Step 1: Update state to recovering
	if err := lm.SetCertificateState(ctx, providerName, certName, CertificateStateRecovering); err != nil {
		lm.log.Warnf("Failed to set recovery state: %v", err)
		// Continue anyway
	}

	// Step 2: Determine authentication method
	authMethod, err := lm.determineRecoveryAuthMethod(ctx)
	if err != nil {
		duration := time.Since(startTime)
		if lm.certManager.metricsCollector != nil {
			lm.certManager.metricsCollector.RecordRecoveryFailure("management", certName, duration)
		}
		logCtx.Duration = duration
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(lm.log, logCtx)
		return fmt.Errorf("failed to determine recovery authentication method: %w", err)
	}

	lm.log.Infof("Using authentication method: %s for recovery", authMethod)

	// Step 3: Generate renewal CSR with appropriate authentication
	csr, attestation, err := lm.generateRecoveryCSR(ctx, providerName, certName, authMethod)
	if err != nil {
		lm.RecordError(ctx, providerName, certName, err)
		duration := time.Since(startTime)
		if lm.certManager.metricsCollector != nil {
			lm.certManager.metricsCollector.RecordRecoveryFailure("management", certName, duration)
		}
		logCtx.Duration = duration
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(lm.log, logCtx)
		return fmt.Errorf("failed to generate recovery CSR: %w", err)
	}

	// Step 4: Submit CSR to service
	csrName, err := lm.submitRecoveryCSR(ctx, csr, attestation)
	if err != nil {
		lm.RecordError(ctx, providerName, certName, err)
		duration := time.Since(startTime)
		if lm.certManager.metricsCollector != nil {
			lm.certManager.metricsCollector.RecordRecoveryFailure("management", certName, duration)
		}
		logCtx.Duration = duration
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(lm.log, logCtx)
		return fmt.Errorf("failed to submit recovery CSR: %w", err)
	}

	lm.log.Infof("Recovery CSR submitted: %s", csrName)

	// Step 5: Poll for certificate approval and reception
	newCert, newKey, err := lm.pollForRecoveryCertificate(ctx, csrName)
	if err != nil {
		lm.RecordError(ctx, providerName, certName, err)
		duration := time.Since(startTime)
		if lm.certManager.metricsCollector != nil {
			lm.certManager.metricsCollector.RecordRecoveryFailure("management", certName, duration)
		}
		logCtx.Duration = duration
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(lm.log, logCtx)
		return fmt.Errorf("failed to receive recovery certificate: %w", err)
	}

	// Step 6: Validate and atomically swap certificate
	if err := lm.installRecoveryCertificate(ctx, providerName, certName, newCert, newKey); err != nil {
		lm.RecordError(ctx, providerName, certName, err)
		duration := time.Since(startTime)
		if lm.certManager.metricsCollector != nil {
			lm.certManager.metricsCollector.RecordRecoveryFailure("management", certName, duration)
		}
		logCtx.Duration = duration
		logCtx.Error = err
		logCtx.Success = false
		LogCertificateOperation(lm.log, logCtx)
		return fmt.Errorf("failed to install recovery certificate: %w", err)
	}

	// Step 7: Update state to normal
	// Calculate days until expiration using the newly installed certificate
	days, err := lm.expirationMonitor.CalculateDaysUntilExpiration(newCert)
	if err == nil {
		expiration := &newCert.NotAfter
		_ = lm.UpdateCertificateState(ctx, providerName, certName, CertificateStateNormal, days, expiration)
	}

	// Record recovery success
	duration := time.Since(startTime)
	if lm.certManager.metricsCollector != nil {
		lm.certManager.metricsCollector.RecordRecoverySuccess("management", certName, duration)
	}

	// Log recovery success
	logCtx.Duration = duration
	logCtx.Success = true
	LogCertificateOperation(lm.log, logCtx)

	return nil
}

// generateRecoveryCSR generates a renewal CSR for recovery with appropriate authentication.
func (lm *LifecycleManager) generateRecoveryCSR(ctx context.Context, providerName string, certName string, authMethod RecoveryAuthMethod) (*api.CertificateSigningRequest, *identity.RenewalAttestation, error) {
	// Get certificate config
	cert, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	cert.mu.RLock()
	_ = cert.Config
	cert.mu.RUnlock()

	// Generate CSR using identity provider
	if lm.identityProvider == nil {
		return nil, nil, fmt.Errorf("identity provider not available")
	}

	var attestation *identity.RenewalAttestation

	// If using TPM attestation, generate it
	if authMethod == RecoveryAuthMethodTPM {
		if lm.tpmRenewalProvider != nil {
			att, err := lm.tpmRenewalProvider.GenerateRenewalAttestation(ctx)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to generate TPM attestation: %w", err)
			}
			attestation = att
		}
		// For TPM, the CSR is embedded in the attestation or needs to be generated separately
		// This is a placeholder - full implementation will use TPM CSR generation
		return nil, attestation, fmt.Errorf("TPM CSR generation not yet fully implemented - requires TPM CSR format")
	}

	// For bootstrap auth, generate standard CSR
	// This requires access to identity provider's CSR generation
	// For now, return error indicating this needs to be implemented
	return nil, attestation, fmt.Errorf("CSR generation for recovery not yet fully implemented - requires identity provider integration")
}

// submitRecoveryCSR submits a recovery CSR to the service.
// It uses the appropriate authentication method (bootstrap cert or TPM).
func (lm *LifecycleManager) submitRecoveryCSR(ctx context.Context, csr *api.CertificateSigningRequest, attestation *identity.RenewalAttestation) (string, error) {
	if lm.managementClient == nil {
		return "", fmt.Errorf("management client not available")
	}

	// Submit CSR
	submittedCSR, statusCode, err := lm.managementClient.CreateCertificateSigningRequest(ctx, *csr)
	if err != nil {
		return "", fmt.Errorf("failed to submit recovery CSR: %w", err)
	}
	if statusCode != 200 && statusCode != 201 {
		return "", fmt.Errorf("unexpected status code %d when submitting recovery CSR", statusCode)
	}

	csrName := lo.FromPtr(submittedCSR.Metadata.Name)
	return csrName, nil
}

// pollForRecoveryCertificate polls for recovery certificate approval and reception.
func (lm *LifecycleManager) pollForRecoveryCertificate(ctx context.Context, csrName string) (*x509.Certificate, []byte, error) {
	lm.log.Debugf("Polling for recovery certificate: %s", csrName)

	if lm.managementClient == nil {
		return nil, nil, fmt.Errorf("management client not available")
	}

	// Poll with exponential backoff
	maxAttempts := 30
	baseDelay := 10 * time.Second
	maxDelay := 5 * time.Minute

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check CSR status
		csr, statusCode, err := lm.managementClient.GetCertificateSigningRequest(ctx, csrName)
		if err != nil {
			lm.log.Warnf("Failed to get CSR status (attempt %d/%d): %v", attempt+1, maxAttempts, err)
			// Continue polling
		} else if statusCode == 200 && csr != nil {
			// Check if approved and certificate is available
			if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestApproved) &&
				csr.Status.Certificate != nil {
				// Certificate is ready
				certPEM := *csr.Status.Certificate
				cert, err := fccrypto.ParsePEMCertificate(certPEM)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
				}

				// Get key from identity provider
				// For recovery, we need to get the key that matches the CSR
				// This depends on how the CSR was generated
				keyPEM, err := lm.getRecoveryKey(ctx, csrName)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get recovery key: %w", err)
				}

				return cert, keyPEM, nil
			}

			// Check if denied or failed
			if api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestDenied) ||
				api.IsStatusConditionTrue(csr.Status.Conditions, api.ConditionTypeCertificateSigningRequestFailed) {
				return nil, nil, fmt.Errorf("recovery CSR was denied or failed")
			}
		}

		// Wait before next poll
		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(delay):
			// Continue polling
		}
	}

	return nil, nil, fmt.Errorf("timeout waiting for recovery certificate after %d attempts", maxAttempts)
}

// getRecoveryKey retrieves the private key for the recovery certificate.
// This is a placeholder that needs to be implemented based on how keys are stored.
func (lm *LifecycleManager) getRecoveryKey(ctx context.Context, csrName string) ([]byte, error) {
	// TODO: Implement key retrieval
	// The key should be stored when the CSR is generated
	// For now, return error indicating this needs to be implemented
	return nil, fmt.Errorf("key retrieval for recovery not yet fully implemented - requires key storage integration")
}

// installRecoveryCertificate installs the recovery certificate using atomic swap.
func (lm *LifecycleManager) installRecoveryCertificate(ctx context.Context, providerName string, certName string, cert *x509.Certificate, keyPEM []byte) error {
	lm.log.Infof("Installing recovery certificate for %q/%q", providerName, certName)

	// Get certificate storage
	certObj, err := lm.certManager.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	certObj.mu.Lock()
	defer certObj.mu.Unlock()

	// Step 1: Write to pending location
	if err := certObj.Storage.WritePending(cert, keyPEM); err != nil {
		return fmt.Errorf("failed to write pending certificate: %w", err)
	}

	// Step 2: Validate pending certificate
	caBundlePath := lm.certManager.getCABundlePath()
	validator := NewCertificateValidator(caBundlePath, certName, lm.log)

	pendingCert, err := certObj.Storage.LoadPendingCertificate(ctx)
	if err != nil {
		_ = certObj.Storage.CleanupPending(ctx)
		return fmt.Errorf("failed to load pending certificate: %w", err)
	}

	pendingKey, err := certObj.Storage.LoadPendingKey(ctx)
	if err != nil {
		_ = certObj.Storage.CleanupPending(ctx)
		return fmt.Errorf("failed to load pending key: %w", err)
	}

	if err := validator.ValidatePendingCertificate(ctx, pendingCert, pendingKey, lm.certManager.readWriter); err != nil {
		_ = certObj.Storage.CleanupPending(ctx)
		return fmt.Errorf("pending certificate validation failed: %w", err)
	}

	// Step 3: Atomically swap certificate
	if err := certObj.Storage.AtomicSwap(ctx); err != nil {
		_ = certObj.Storage.CleanupPending(ctx)
		return fmt.Errorf("failed to atomically swap certificate: %w", err)
	}

	// Step 4: Update certificate info
	lm.certManager.addCertificateInfo(certObj, cert)

	lm.log.Infof("Recovery certificate installed successfully for %q/%q", providerName, certName)
	return nil
}
