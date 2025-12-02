package certmanager

import (
	"context"
	"crypto/x509"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/flightctl/flightctl/internal/agent/config"
	"github.com/flightctl/flightctl/internal/agent/device/certmanager/provider"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
)

const DefaultRequeueDelay = 10 * time.Second

// CertManager manages the complete certificate lifecycle for flight control agents.
// It coordinates certificate provisioning, storage, and cleanup across multiple configuration providers.
// The manager supports pluggable provisioners (CSR, self-signed, etc.) and storage
// backends (filesystem, etc.) through factory patterns.
type CertManager struct {
	log provider.Logger
	// In-memory certificate state with optional persistent backing
	certificates *certStorage
	// Configuration providers (agent-config, file, static)
	configs map[string]provider.ConfigProvider
	// Certificate provisioner factories (CSR, self-signed, empty)
	provisioners map[string]provider.ProvisionerFactory
	// Storage provider factories (filesystem, empty)
	storages map[string]provider.StorageFactory
	// Queue for async certificate processing with retry logic
	processingQueue *CertificateProcessingQueue
	// Delay before retrying failed certificate operations
	requeueDelay time.Duration
	// ExpirationMonitor for checking certificate validity
	expirationMonitor *ExpirationMonitor
	// LifecycleManager for managing certificate lifecycle states
	lifecycleManager *LifecycleManager
	// config stores agent configuration for renewal settings
	config *config.Config
	// readWriter for file I/O operations (used for CA bundle loading)
	readWriter fileio.ReadWriter
	// metricsCollector for recording certificate metrics (optional)
	metricsCollector MetricsCollector
}

// MetricsCollector defines the interface for certificate metrics collection.
// This allows the certificate manager to record metrics without directly depending on Prometheus.
type MetricsCollector interface {
	RecordCertificateExpiration(certType, certName string, expirationTime time.Time, daysUntilExpiration int)
	RecordRenewalAttempt(certType, certName, reason string)
	RecordRenewalSuccess(certType, certName, reason string, duration time.Duration)
	RecordRenewalFailure(certType, certName, reason string, duration time.Duration)
	RecordRecoveryAttempt(certType, certName string)
	RecordRecoverySuccess(certType, certName string, duration time.Duration)
	RecordRecoveryFailure(certType, certName string, duration time.Duration)
}

// ManagerOption defines a functional option for configuring CertManager during initialization.
type ManagerOption func(*CertManager) error

// WithRequeueDelay sets a custom requeue delay for certificate provisioning checks.
// This delay is used when a certificate provisioning operation is not yet complete
// and needs to be retried (e.g., waiting for CSR approval).
func WithRequeueDelay(delay time.Duration) ManagerOption {
	return func(cm *CertManager) error {
		if delay <= 0 {
			return fmt.Errorf("requeue delay must be positive")
		}
		cm.requeueDelay = delay
		return nil
	}
}

// WithConfigProvider adds a configuration provider to the manager.
// Configuration providers supply certificate configurations and can notify of changes.
// Multiple providers can be registered (e.g., agent-config, file-based, static).
func WithConfigProvider(config provider.ConfigProvider) ManagerOption {
	return func(cm *CertManager) error {
		if config == nil {
			return fmt.Errorf("provided config provider is nil")
		}

		name := config.Name()
		if _, ok := cm.configs[config.Name()]; ok {
			return fmt.Errorf("config provider with name %q already exists", name)
		}

		cm.configs[name] = config
		return nil
	}
}

// WithProvisionerProvider registers a provisioner factory with the manager.
// Provisioner factories create certificate provisioners (CSR, self-signed, etc.)
// based on certificate configuration. Each factory handles a specific provisioner type.
func WithProvisionerProvider(prov provider.ProvisionerFactory) ManagerOption {
	return func(cm *CertManager) error {
		if prov == nil {
			return fmt.Errorf("provided provisioner factory is nil")
		}

		t := prov.Type()
		if _, exists := cm.provisioners[t]; exists {
			return fmt.Errorf("provisioner factory for type %q already exists", t)
		}

		cm.provisioners[t] = prov
		return nil
	}
}

// WithStorageProvider registers a storage factory with the manager.
// Storage factories create certificate storage providers (filesystem, etc.) that
// handle writing certificates and private keys to their final destinations.
func WithStorageProvider(store provider.StorageFactory) ManagerOption {
	return func(cm *CertManager) error {
		if store == nil {
			return fmt.Errorf("provided storage factory is nil")
		}

		t := store.Type()
		if _, exists := cm.storages[t]; exists {
			return fmt.Errorf("storage factory for type %q already exists", t)
		}

		cm.storages[t] = store
		return nil
	}
}

// WithMetricsCollector sets the metrics collector for certificate operations.
func WithMetricsCollector(collector MetricsCollector) ManagerOption {
	return func(cm *CertManager) error {
		cm.metricsCollector = collector
		return nil
	}
}

// NewManager creates and initializes a new CertManager with the provided options.
func NewManager(ctx context.Context, log provider.Logger, opts ...ManagerOption) (*CertManager, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	cm := &CertManager{
		log:               log,
		configs:           make(map[string]provider.ConfigProvider),
		provisioners:      make(map[string]provider.ProvisionerFactory),
		storages:          make(map[string]provider.StorageFactory),
		certificates:      newCertStorage(),
		expirationMonitor: NewExpirationMonitor(log),
	}

	for _, opt := range opts {
		if optErr := opt(cm); optErr != nil {
			return nil, fmt.Errorf("failed to apply option: %w", optErr)
		}
	}

	if cm.requeueDelay == 0 {
		cm.requeueDelay = DefaultRequeueDelay
	}

	// Initialize expiration monitor if not already set
	if cm.expirationMonitor == nil {
		cm.expirationMonitor = NewExpirationMonitor(log)
	}

	// Initialize lifecycle manager
	// Note: bootstrapHandler will be set later if available
	cm.lifecycleManager = NewLifecycleManager(cm, cm.expirationMonitor, log, nil)

	cm.processingQueue = NewCertificateProcessingQueue(cm.ensureCertificate)
	go cm.processingQueue.Run(ctx)
	return cm, nil
}

// Sync performs a full synchronization of all certificate providers.
func (cm *CertManager) Sync(ctx context.Context, cfg *config.Config) error {
	// Store config for use in renewal checks
	cm.config = cfg

	cm.log.Debug("Starting certificate sync")
	if err := cm.sync(ctx); err != nil {
		cm.log.Errorf("certificate management sync failed: %v", err)
		return err
	}
	return nil
}

// sync performs a full synchronization of all certificate providers.
// It iterates through all registered configuration providers, syncs their certificates,
// and cleans up any providers that are no longer configured.
func (cm *CertManager) sync(ctx context.Context) error {
	handledProviders := make([]string, 0, len(cm.configs))

	defer func() {
		cleanupErr := cm.cleanupUntrackedProviders(handledProviders)
		if cleanupErr != nil {
			cm.log.Errorf("Failed to cleanup untracked providers: %v", cleanupErr)
		}
	}()

	for providerName, cfgProvider := range cm.configs {
		handledProviders = append(handledProviders, providerName)

		if err := cm.syncProvider(ctx, cfgProvider); err != nil {
			cm.log.Errorf("syncProvider failed for %q: %v", providerName, err)
		}
	}

	// Check for expired certificates after sync
	if cm.lifecycleManager != nil {
		if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
			cm.log.Warnf("Failed to check expired certificates during sync: %v", err)
			// Don't fail sync if expiration check fails
		}
	}

	return nil
}

// syncProvider synchronizes certificates from a specific configuration provider.
// It loads certificate configurations, ensures each certificate is properly managed,
// and cleans up any certificates that are no longer configured.
func (cm *CertManager) syncProvider(ctx context.Context, provider provider.ConfigProvider) error {
	handledCertificates := make([]string, 0)
	providerName := provider.Name()

	defer func() {
		cleanupErr := cm.cleanupUntrackedCertificates(providerName, handledCertificates)
		if cleanupErr != nil {
			cm.log.Errorf("Failed to cleanup untracked certificates: %v", cleanupErr)
		}
	}()

	configs, loadErr := provider.GetCertificateConfigs()
	if loadErr != nil {
		// Mark existing certificates as handled so they won't be deleted
		cm.log.Errorf("failed to load certificate configs from provider %q: %v", providerName, loadErr)

		if _, ensureErr := cm.certificates.EnsureProvider(providerName); ensureErr != nil {
			return fmt.Errorf("ensure provider %q: %w", providerName, ensureErr)
		}

		certs, snapErr := cm.certificates.ReadCertificates(providerName)
		if snapErr != nil {
			// Be conservative: without a snapshot we might delete valid certs.
			return fmt.Errorf("snapshot existing certificates for provider %q: %w", providerName, snapErr)
		}

		for _, c := range certs {
			handledCertificates = append(handledCertificates, c.Name)
		}

		return fmt.Errorf("load certificate configs from provider %q: %w", providerName, loadErr)
	}

	if _, err := cm.certificates.EnsureProvider(providerName); err != nil {
		return err
	}

	for _, cfg := range configs {
		if err := cm.syncCertificate(ctx, provider, cfg); err != nil {
			cm.log.Errorf("syncCertificate failed for %q/%q: %v", providerName, cfg.Name, err)
		}
		handledCertificates = append(handledCertificates, cfg.Name)
	}

	return nil
}

// syncCertificate synchronizes a single certificate.
func (cm *CertManager) syncCertificate(ctx context.Context, provider provider.ConfigProvider, cfg provider.CertificateConfig) error {
	var err error
	providerName := provider.Name()
	certName := cfg.Name

	cert, err := cm.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		cert = cm.createCertificate(ctx, provider, cfg)
	}

	cert.mu.Lock()
	defer cert.mu.Unlock()

	if cm.processingQueue.IsProcessing(providerName, cert.Name) {
		_, usedCfg := cm.processingQueue.Get(providerName, cert.Name)

		if !usedCfg.Equal(cfg) {
			// Remove old queued item
			cm.processingQueue.Remove(providerName, cert.Name)

			// Re-queue with new config
			if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
				return fmt.Errorf("failed to provision certificate %q from provider %q: %w", cert.Name, providerName, err)
			}
			cm.log.Debugf("Config changed during processing — re-queued provision for certificate %q of provider %q", certName, providerName)
		}
		return nil
	}

	if !cm.shouldprovisionCertificate(providerName, cert, cfg) {
		cert.Config = cfg

		// Check if renewal is needed based on expiration
		if cm.lifecycleManager != nil && cm.config != nil {
			thresholdDays := cm.config.Certificate.Renewal.ThresholdDays
			if thresholdDays == 0 {
				thresholdDays = 30 // Default threshold
			}

			// Only check renewal if certificate has expiration info
			if cert.Info.NotAfter != nil {
				needsRenewal, days, err := cm.shouldRenewCertificate(ctx, providerName, cert, thresholdDays)
				if err != nil {
					cm.log.Warnf("Failed to check renewal for certificate %q/%q: %v", providerName, certName, err)
				} else if needsRenewal {
					cm.log.Infof("Certificate %q/%q needs renewal (expires in %d days)", providerName, certName, days)

					// Check current state - don't trigger if already renewing
					currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, certName)
					if err == nil && currentState.GetState() != CertificateStateRenewing {
						if err := cm.triggerRenewal(ctx, providerName, cert, cfg); err != nil {
							cm.log.Errorf("Failed to trigger renewal for certificate %q/%q: %v", providerName, certName, err)
							// Continue - renewal will be retried on next sync
						}
					}
				}
			}
		}

		cm.log.Debugf("Certificate %q for provider %q: no provision required", certName, providerName)
		return nil
	}

	if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
		return fmt.Errorf("failed to provision certificate %q from provider %q: %w", cert.Name, providerName, err)
	}

	cm.log.Debugf("Provision triggered for certificate %q of provider %q", certName, providerName)
	return nil
}

// createCertificate creates a new certificate object and attempts to load existing
// certificate information from the storage provider if available.
func (cm *CertManager) createCertificate(ctx context.Context, provider provider.ConfigProvider, cfg provider.CertificateConfig) *certificate {
	providerName := provider.Name()
	certName := cfg.Name

	cert := &certificate{
		Name:   certName,
		Config: cfg,
	}

	// Remove from processing queue if already in flight (resetting any previous state)
	if cm.processingQueue.IsProcessing(providerName, certName) {
		cm.processingQueue.Remove(providerName, certName)
	}

	// Try to load existing certificate details from storage provider
	storage, err := cm.initStorageProvider(cfg)
	if err == nil {
		parsedCert, loadErr := storage.LoadCertificate(ctx)
		if loadErr == nil && parsedCert != nil {
			cm.addCertificateInfo(cert, parsedCert)
		} else if loadErr != nil {
			cm.log.Debugf("no existing cert loaded for %q/%q: %v", providerName, certName, loadErr)
		}
	} else {
		cm.log.Errorf("failed to init storage provider for certificate %q from provider %q: %v", certName, providerName, err)
	}

	if err := cm.certificates.StoreCertificate(providerName, cert); err != nil {
		cm.log.Errorf("failed to store certificate %q from provider %q: %v", certName, providerName, err)
	}

	return cert
}

// shouldprovisionCertificate determines whether a certificate needs provisioning.
func (cm *CertManager) shouldprovisionCertificate(providerName string, cert *certificate, cfg provider.CertificateConfig) bool {
	// Missing critical cert info — first provision.
	if cert.Info.NotAfter == nil || cert.Info.NotBefore == nil {
		cm.log.Debugf("Certificate %q for provider %q: missing NotBefore/NotAfter — initial provisioning", cert.Name, providerName)
		return true
	}

	if !cert.Config.Provisioner.Equal(cfg.Provisioner) || !cert.Config.Storage.Equal(cfg.Storage) {
		cm.log.Debugf("Certificate %q for provider %q: provisioner or storage changed - needs provisioning", cert.Name, providerName)
		return true
	}

	return false
}

// provisionCertificate queues a certificate for provisioning by adding it to the processing queue.
func (cm *CertManager) provisionCertificate(_ context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig) error {
	return cm.processingQueue.Process(providerName, cert, cfg)
}

// ensureCertificate is the main certificate processing function called by the processing queue.
func (cm *CertManager) ensureCertificate(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) *time.Duration {
	cert.mu.Lock()
	defer cert.mu.Unlock()

	defer func() {
		// Always persist certificate state after execution
		if err := cm.certificates.StoreCertificate(providerName, cert); err != nil {
			cm.log.Errorf("failed to store certificate %q from provider %q: %v", cert.Name, providerName, err)
		}
	}()

	// Attempt to ensure certificate (provision)
	retryDelay, err := cm.ensureCertificate_do(ctx, providerName, cert, cfg)
	if err != nil {
		cm.log.Errorf("failed to ensure certificate %q from provider %q: %v", cert.Name, providerName, err)

		// Update lifecycle state if this was a renewal
		if cm.lifecycleManager != nil {
			currentState, stateErr := cm.lifecycleManager.GetCertificateState(ctx, providerName, cert.Name)
			if stateErr == nil && currentState.GetState() == CertificateStateRenewing {
				// Renewal failed
				_ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
				// State will be set to renewal_failed by RecordError
			} else {
				// Record error in lifecycle manager
				_ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
			}
		}

		// On failure, reset provisioner and storage to force re-init next time
		cert.Provisioner = nil
		cert.Storage = nil
		return nil
	}

	// If no retry delay is returned, we consider it "final success"
	if retryDelay == nil {
		cert.Provisioner = nil
		cert.Storage = nil
	}

	return retryDelay
}

// ensureCertificate_do performs the actual certificate provisioning work.
// It initializes provisioner and storage providers, requests certificate provisioning,
// and writes the certificate to storage when ready.
func (cm *CertManager) ensureCertificate_do(ctx context.Context, providerName string, cert *certificate, cfg *provider.CertificateConfig) (*time.Duration, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil configurations")
	}

	config := *cfg
	certName := cert.Name

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if cert.Storage == nil {
		s, err := cm.initStorageProvider(config)
		if err != nil {
			return nil, err
		}
		cert.Storage = s
	}

	if cert.Provisioner == nil {
		p, err := cm.initProvisionerProvider(config)
		if err != nil {
			return nil, err
		}
		cert.Provisioner = p
	}

	// Log CSR generation/submission start
	// Check if this is a renewal to determine reason
	csrIsRenewal := cert.Info.NotAfter != nil && cert.Info.NotBefore != nil
	csrReason := "provisioning"
	if csrIsRenewal {
		csrReason = "proactive"
		if cert.Info.NotAfter != nil && time.Now().After(*cert.Info.NotAfter) {
			csrReason = "expired"
		}
	}

	csrLogCtx := CertificateLogContext{
		Operation:       "csr_generation",
		CertificateType: "management",
		CertificateName: cert.Name,
		DeviceName:      cert.Name,
		Reason:          csrReason,
	}
	if csrIsRenewal && cert.Info.NotAfter != nil {
		now := time.Now()
		daysUntilExpiration := int(cert.Info.NotAfter.Sub(now).Hours() / 24)
		csrLogCtx.DaysUntilExpiration = daysUntilExpiration
	}
	LogCertificateOperation(cm.log, csrLogCtx)

	ready, crt, keyBytes, err := cert.Provisioner.Provision(ctx)
	if err != nil {
		csrLogCtx.Error = err
		csrLogCtx.Success = false
		LogCertificateOperation(cm.log, csrLogCtx)
		return nil, err
	}

	// Log CSR submission
	csrLogCtx.Operation = "csr_submission"
	LogCertificateOperation(cm.log, csrLogCtx)

	if !ready {
		return &cm.requeueDelay, nil
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// check storage drift
	if !cert.Config.Storage.Equal(cfg.Storage) {
		cm.log.Debugf("Certificate %q for provider %q: storage configuration changed, deleting old storage", certName, providerName)
		if err := cm.purgeStorage(ctx, providerName, cert); err != nil {
			cm.log.Error(err.Error())
		}
	}

	// Check if this is a renewal (certificate already exists)
	isRenewal := cert.Info.NotAfter != nil && cert.Info.NotBefore != nil

	var renewalStartTime time.Time
	var renewalReason string
	var logCtx CertificateLogContext
	if isRenewal {
		renewalStartTime = time.Now()
		renewalReason = "proactive"
		if cert.Info.NotAfter != nil && time.Now().After(*cert.Info.NotAfter) {
			renewalReason = "expired"
		}

		// Calculate days until expiration for logging
		daysUntilExpiration := 0
		if cert.Info.NotAfter != nil {
			now := time.Now()
			daysUntilExpiration = int(cert.Info.NotAfter.Sub(now).Hours() / 24)
		}

		// Log renewal start
		logCtx = CertificateLogContext{
			Operation:           "renewal",
			CertificateType:     "management",
			CertificateName:     cert.Name,
			DeviceName:          cert.Name, // Use certificate name as device identifier
			Reason:              renewalReason,
			ThresholdDays:       cm.config.Certificate.Renewal.ThresholdDays,
			DaysUntilExpiration: daysUntilExpiration,
		}
		LogCertificateOperation(cm.log, logCtx)

		// Record renewal attempt
		if cm.metricsCollector != nil {
			cm.metricsCollector.RecordRenewalAttempt("management", cert.Name, renewalReason)
		}
	}

	if isRenewal {
		// For renewals, write to pending location first
		if err := cert.Storage.WritePending(crt, keyBytes); err != nil {
			if cm.metricsCollector != nil {
				duration := time.Since(renewalStartTime)
				cm.metricsCollector.RecordRenewalFailure("management", cert.Name, renewalReason, duration)
			}
			logCtx.Duration = time.Since(renewalStartTime)
			logCtx.Error = err
			logCtx.Success = false
			LogCertificateOperation(cm.log, logCtx)
			return nil, fmt.Errorf("failed to write pending certificate: %w", err)
		}
		cm.log.Infof("Certificate %q/%q written to pending location for validation", providerName, cert.Name)

		// Validate pending certificate before activation
		// Get CA bundle path from config
		caBundlePath := cm.getCABundlePath()
		validator := NewCertificateValidator(caBundlePath, cert.Name, cm.log)

		// Load pending certificate and key for validation
		pendingCert, err := cert.Storage.LoadPendingCertificate(ctx)
		if err != nil {
			_ = cert.Storage.CleanupPending(ctx)
			if cm.metricsCollector != nil {
				duration := time.Since(renewalStartTime)
				cm.metricsCollector.RecordRenewalFailure("management", cert.Name, renewalReason, duration)
			}
			logCtx.Duration = time.Since(renewalStartTime)
			logCtx.Error = err
			logCtx.Success = false
			LogCertificateOperation(cm.log, logCtx)
			return nil, fmt.Errorf("failed to load pending certificate: %w", err)
		}

		pendingKey, err := cert.Storage.LoadPendingKey(ctx)
		if err != nil {
			_ = cert.Storage.CleanupPending(ctx)
			if cm.metricsCollector != nil {
				duration := time.Since(renewalStartTime)
				cm.metricsCollector.RecordRenewalFailure("management", cert.Name, renewalReason, duration)
			}
			logCtx.Duration = time.Since(renewalStartTime)
			logCtx.Error = err
			logCtx.Success = false
			LogCertificateOperation(cm.log, logCtx)
			return nil, fmt.Errorf("failed to load pending key: %w", err)
		}

		// Validate pending certificate
		if err := validator.ValidatePendingCertificate(ctx, pendingCert, pendingKey, cm.readWriter); err != nil {
			_ = cert.Storage.CleanupPending(ctx)
			if cm.metricsCollector != nil {
				duration := time.Since(renewalStartTime)
				cm.metricsCollector.RecordRenewalFailure("management", cert.Name, renewalReason, duration)
			}
			logCtx.Duration = time.Since(renewalStartTime)
			logCtx.Error = err
			logCtx.Success = false
			LogCertificateOperation(cm.log, logCtx)
			return nil, fmt.Errorf("pending certificate validation failed: %w", err)
		}

		cm.log.Infof("Pending certificate validation successful for %q/%q", providerName, cert.Name)

		// Perform atomic swap after validation
		if err := cert.Storage.AtomicSwap(ctx); err != nil {
			// Rollback already handled in AtomicSwap, but ensure cleanup
			_ = cert.Storage.CleanupPending(ctx)

			// Update lifecycle state to reflect failure
			if cm.lifecycleManager != nil {
				_ = cm.lifecycleManager.RecordError(ctx, providerName, cert.Name, err)
				// State will be set to renewal_failed by RecordError
				_ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateRenewalFailed)
			}

			if cm.metricsCollector != nil {
				duration := time.Since(renewalStartTime)
				cm.metricsCollector.RecordRenewalFailure("management", cert.Name, renewalReason, duration)
			}
			logCtx.Duration = time.Since(renewalStartTime)
			logCtx.Error = err
			logCtx.Success = false
			LogCertificateOperation(cm.log, logCtx)
			return nil, fmt.Errorf("failed to atomically swap certificate: %w", err)
		}

		cm.log.Infof("Certificate %q/%q atomically swapped to active location", providerName, cert.Name)

		// Record renewal success metrics
		if cm.metricsCollector != nil {
			duration := time.Since(renewalStartTime)
			cm.metricsCollector.RecordRenewalSuccess("management", cert.Name, renewalReason, duration)
		}

		// Log renewal success
		logCtx.Duration = time.Since(renewalStartTime)
		logCtx.Success = true
		LogCertificateOperation(cm.log, logCtx)
	} else {
		// For initial provisioning, write directly to active location
		if err := cert.Storage.Write(crt, keyBytes); err != nil {
			return nil, err
		}
	}

	cm.addCertificateInfo(cert, crt)

	// Update lifecycle state based on renewal context
	if cm.lifecycleManager != nil {
		// Check if this was a renewal (state was "renewing")
		currentState, err := cm.lifecycleManager.GetCertificateState(ctx, providerName, cert.Name)
		if err == nil && currentState.GetState() == CertificateStateRenewing {
			// Renewal completed successfully
			// Calculate days until expiration for the new certificate
			days, err := cm.expirationMonitor.CalculateDaysUntilExpiration(crt)
			if err == nil {
				_ = cm.lifecycleManager.UpdateCertificateState(ctx, providerName, cert.Name,
					CertificateStateNormal, days, &crt.NotAfter)
				cm.log.Infof("Certificate %q/%q renewal completed successfully", providerName, cert.Name)
			} else {
				_ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateNormal)
			}
		} else {
			// Initial provisioning or other operation
			_ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateNormal)
		}
	}

	cert.Config = config
	cert.Provisioner = nil
	cert.Storage = nil
	return nil, nil
}

// getCABundlePath returns the CA bundle path from configuration.
func (cm *CertManager) getCABundlePath() string {
	if cm.config != nil && cm.config.ManagementService.Config.Service.CertificateAuthority != "" {
		return cm.config.ManagementService.Config.Service.CertificateAuthority
	}
	// Default fallback
	if cm.config != nil {
		return filepath.Join(cm.config.ConfigDir, "certs", "ca.crt")
	}
	return "/etc/flightctl/certs/ca.crt"
}

// addCertificateInfo extracts and stores certificate information from a parsed X.509 certificate.
func (cm *CertManager) addCertificateInfo(cert *certificate, parsedCert *x509.Certificate) {
	cert.Info.NotBefore = &parsedCert.NotBefore
	cert.Info.NotAfter = &parsedCert.NotAfter

	// Record expiration metrics
	if cm.metricsCollector != nil {
		now := time.Now()
		daysUntilExpiration := int(parsedCert.NotAfter.Sub(now).Hours() / 24)
		cm.metricsCollector.RecordCertificateExpiration("management", cert.Name, parsedCert.NotAfter, daysUntilExpiration)
	}
}

// shouldRenewCertificate determines if a certificate needs renewal based on expiration threshold.
// Returns true if renewal is needed, days until expiration, and any error.
func (cm *CertManager) shouldRenewCertificate(ctx context.Context, providerName string, cert *certificate, thresholdDays int) (bool, int, error) {
	if cm.lifecycleManager == nil {
		return false, 0, fmt.Errorf("lifecycle manager not initialized")
	}

	if thresholdDays < 0 {
		return false, 0, fmt.Errorf("threshold days must be non-negative, got %d", thresholdDays)
	}

	// Use lifecycle manager to check if renewal is needed
	needsRenewal, days, err := cm.lifecycleManager.CheckRenewal(ctx, providerName, cert.Name, thresholdDays)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check renewal: %w", err)
	}

	return needsRenewal, days, nil
}

// triggerRenewal initiates the certificate renewal process.
// It sets the certificate state to "renewing" and queues the certificate for renewal.
func (cm *CertManager) triggerRenewal(ctx context.Context, providerName string, cert *certificate, cfg provider.CertificateConfig) error {
	if cm.lifecycleManager == nil {
		return fmt.Errorf("lifecycle manager not initialized")
	}

	// Set certificate state to "renewing"
	if err := cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateRenewing); err != nil {
		cm.log.Warnf("Failed to set certificate state to renewing for %q/%q: %v", providerName, cert.Name, err)
		// Continue anyway - state update failure shouldn't block renewal
	}

	// Queue certificate for renewal
	// Note: We use the same provisioning queue, but with renewal context
	// The CSR provisioner will handle renewal-specific logic in the next story
	if err := cm.provisionCertificate(ctx, providerName, cert, cfg); err != nil {
		// If queuing fails, reset state
		_ = cm.lifecycleManager.SetCertificateState(ctx, providerName, cert.Name, CertificateStateExpiringSoon)
		return fmt.Errorf("failed to queue certificate for renewal: %w", err)
	}

	cm.log.Infof("Triggered renewal for certificate %q/%q", providerName, cert.Name)
	return nil
}

// GetLifecycleManager returns the lifecycle manager for certificate operations.
func (cm *CertManager) GetLifecycleManager() *LifecycleManager {
	return cm.lifecycleManager
}

// CheckIncompleteSwaps checks for and recovers from incomplete certificate swaps.
// This should be called on agent startup.
func (cm *CertManager) CheckIncompleteSwaps(ctx context.Context) error {
	cm.log.Debug("Checking for incomplete certificate swaps")

	// Get list of providers
	providers, err := cm.certificates.ListProviderNames()
	if err != nil {
		return fmt.Errorf("failed to list provider names: %w", err)
	}

	// Iterate through all certificates
	for _, providerName := range providers {
		certs, err := cm.certificates.ReadCertificates(providerName)
		if err != nil {
			cm.log.Warnf("Failed to read certificates for provider %q: %v", providerName, err)
			continue
		}

		for _, cert := range certs {
			// Initialize storage if needed
			if cert.Storage == nil {
				storage, err := cm.initStorageProvider(cert.Config)
				if err != nil {
					cm.log.Warnf("Failed to init storage for %q/%q: %v", providerName, cert.Name, err)
					continue
				}
				cert.Storage = storage
			}

			// Check if certificate has pending files
			hasPending, err := cert.Storage.HasPendingCertificate(ctx)
			if err != nil {
				cm.log.Warnf("Failed to check pending certificate for %q/%q: %v", providerName, cert.Name, err)
				continue
			}

			if hasPending {
				cm.log.Warnf("Detected incomplete swap for certificate %q/%q", providerName, cert.Name)

				// Create validator and attempt recovery
				caBundlePath := cm.getCABundlePath()
				validator := NewCertificateValidator(caBundlePath, cert.Name, cm.log)

				if err := validator.DetectAndRecoverIncompleteSwap(ctx, cert.Storage); err != nil {
					cm.log.Errorf("Failed to recover incomplete swap for %q/%q: %v", providerName, cert.Name, err)
					// Continue with other certificates
				} else {
					cm.log.Infof("Recovered incomplete swap for certificate %q/%q", providerName, cert.Name)
				}
			}
		}
	}

	return nil
}

// cleanupUntrackedProviders removes certificate providers that are no longer configured.
// It cancels any in-flight processing for certificates from removed providers.
func (cm *CertManager) cleanupUntrackedProviders(keepProviders []string) error {
	keepMap := make(map[string]struct{}, len(keepProviders))
	for _, name := range keepProviders {
		keepMap[name] = struct{}{}
	}

	providers, err := cm.certificates.ListProviderNames()
	if err != nil {
		return fmt.Errorf("failed to list provider names: %w", err)
	}

	for _, providerName := range providers {
		if _, ok := keepMap[providerName]; ok {
			continue
		}

		certs, err := cm.certificates.ReadCertificates(providerName)
		if err != nil {
			cm.log.Errorf("failed to read certificates for provider %q: %v", providerName, err)
			continue
		}

		for _, cert := range certs {
			if cm.processingQueue.IsProcessing(providerName, cert.Name) {
				cm.processingQueue.Remove(providerName, cert.Name)
			}
		}

		if err := cm.certificates.RemoveProvider(providerName); err != nil {
			cm.log.Errorf("failed to remove provider %q: %v", providerName, err)
			continue
		}

		cm.log.Debugf("Removed untracked provider %q and all associated certificates", providerName)
	}

	return nil
}

// cleanupUntrackedCertificates removes certificates that are no longer configured
// from a specific provider. It cancels any in-flight processing for removed certificates.
func (cm *CertManager) cleanupUntrackedCertificates(providerName string, keepCerts []string) error {
	if providerName == "" {
		return fmt.Errorf("provider name is empty")
	}

	keepMap := make(map[string]struct{}, len(keepCerts))
	for _, name := range keepCerts {
		keepMap[name] = struct{}{}
	}

	certs, err := cm.certificates.ReadCertificates(providerName)
	if err != nil {
		return fmt.Errorf("failed to read certificates for provider %q: %w", providerName, err)
	}

	for _, cert := range certs {
		if _, keep := keepMap[cert.Name]; keep {
			continue
		}

		if cm.processingQueue.IsProcessing(providerName, cert.Name) {
			cm.processingQueue.Remove(providerName, cert.Name)
		}

		if err := cm.certificates.RemoveCertificate(providerName, cert.Name); err != nil {
			cm.log.Errorf("failed to remove certificate %q from provider %q: %v", cert.Name, providerName, err)
			continue
		}

		cm.log.Debugf("Removed untracked certificate %q from provider %q", cert.Name, providerName)
	}

	return nil
}

// initProvisionerProvider creates a provisioner provider from the certificate configuration.
// It validates the configuration and returns a provisioner capable of generating certificates.
func (cm *CertManager) initProvisionerProvider(cfg provider.CertificateConfig) (provider.ProvisionerProvider, error) {
	if strings.TrimSpace(string(cfg.Provisioner.Type)) == "" {
		return nil, fmt.Errorf("provisioner type is not set for certificate %q", cfg.Name)
	}

	p, ok := cm.provisioners[string(cfg.Provisioner.Type)]
	if !ok {
		return nil, fmt.Errorf("provisioner type %q not registered", cfg.Provisioner.Type)
	}

	if err := p.Validate(cm.log, cfg); err != nil {
		return nil, fmt.Errorf("validation failed for provisioner type %q: %w", cfg.Provisioner.Type, err)
	}

	return p.New(cm.log, cfg)
}

// initStorageProvider creates a storage provider from the certificate configuration.
// It validates the configuration and returns a storage provider capable of writing certificates.
func (cm *CertManager) initStorageProvider(cfg provider.CertificateConfig) (provider.StorageProvider, error) {
	if strings.TrimSpace(string(cfg.Storage.Type)) == "" {
		return nil, fmt.Errorf("storage type is not set for certificate %q", cfg.Name)
	}

	p, ok := cm.storages[string(cfg.Storage.Type)]
	if !ok {
		return nil, fmt.Errorf("storage type %q not registered", cfg.Storage.Type)
	}

	if err := p.Validate(cm.log, cfg); err != nil {
		return nil, fmt.Errorf("validation failed for storage type %q: %w", cfg.Storage.Type, err)
	}

	return p.New(cm.log, cfg)
}

// purgeStorage removes certificate and key files from the storage provider.
func (cm *CertManager) purgeStorage(ctx context.Context, providerName string, cert *certificate) error {
	certName := cert.Name

	storage, err := cm.initStorageProvider(cert.Config)
	if err != nil {
		return fmt.Errorf("failed to initialize old storage provider for certificate %q from provider %q: %w", certName, providerName, err)
	}

	if err := storage.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete old storage for certificate %q from provider %q: %w", certName, providerName, err)
	}

	return nil
}

// CheckCertificateExpiration checks the expiration status of a certificate.
// Returns days until expiration, expiration time, and any error.
func (cm *CertManager) CheckCertificateExpiration(ctx context.Context, providerName, certName string) (int, *time.Time, error) {
	cert, err := cm.certificates.ReadCertificate(providerName, certName)
	if err != nil {
		return 0, nil, fmt.Errorf("certificate %q from provider %q not found: %w", certName, providerName, err)
	}

	cert.mu.RLock()
	hasExpirationInfo := cert.Info.NotAfter != nil
	cert.mu.RUnlock()

	// Try to load certificate from storage if not already loaded
	if !hasExpirationInfo {
		cert.mu.Lock()
		if cert.Storage == nil {
			// Initialize storage if needed
			storage, err := cm.initStorageProvider(cert.Config)
			if err != nil {
				cert.mu.Unlock()
				return 0, nil, fmt.Errorf("failed to init storage: %w", err)
			}
			cert.Storage = storage
		}
		cert.mu.Unlock()

		// Load certificate from storage
		x509Cert, err := cert.Storage.LoadCertificate(ctx)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to load certificate: %w", err)
		}

		// Update certificate info
		cert.mu.Lock()
		cert.Info.NotBefore = &x509Cert.NotBefore
		cert.Info.NotAfter = &x509Cert.NotAfter
		cert.mu.Unlock()

		// Persist updated info
		if err := cm.certificates.StoreCertificate(providerName, cert); err != nil {
			cm.log.Warnf("failed to store certificate info: %v", err)
		}
	}

	// Parse certificate expiration
	cert.mu.RLock()
	notAfter := cert.Info.NotAfter
	cert.mu.RUnlock()

	if notAfter == nil {
		return 0, nil, fmt.Errorf("certificate has no expiration date")
	}

	// Calculate days until expiration
	days, err := cm.expirationMonitor.CalculateDaysUntilExpiration(&x509.Certificate{
		NotAfter: *notAfter,
	})
	if err != nil {
		return 0, nil, err
	}

	return days, notAfter, nil
}

// CheckAllCertificatesExpiration checks expiration for all managed certificates.
// This method is intended to be called periodically.
func (cm *CertManager) CheckAllCertificatesExpiration(ctx context.Context) error {
	providers, err := cm.certificates.ListProviderNames()
	if err != nil {
		return fmt.Errorf("failed to list provider names: %w", err)
	}

	for _, providerName := range providers {
		certs, err := cm.certificates.ReadCertificates(providerName)
		if err != nil {
			cm.log.Warnf("failed to read certificates for provider %q: %v", providerName, err)
			continue
		}

		for _, cert := range certs {
			days, expiration, err := cm.CheckCertificateExpiration(ctx, providerName, cert.Name)
			if err != nil {
				cm.log.Warnf("failed to check expiration for certificate %q from provider %q: %v",
					cert.Name, providerName, err)
				continue
			}

			if days < 0 {
				cm.log.Warnf("certificate %q from provider %q expired %d days ago (expired: %v)",
					cert.Name, providerName, -days, expiration)
			} else {
				cm.log.Debugf("certificate %q from provider %q expires in %d days (expires: %v)",
					cert.Name, providerName, days, expiration)
			}
		}
	}

	return nil
}

// StartPeriodicExpirationCheck starts a goroutine that periodically checks certificate expiration.
// The check interval is configurable via the checkInterval parameter.
// The goroutine stops when ctx is cancelled.
func (cm *CertManager) StartPeriodicExpirationCheck(ctx context.Context, checkInterval time.Duration) {
	if checkInterval <= 0 {
		cm.log.Warnf("invalid check interval %v, using default 24h", checkInterval)
		checkInterval = 24 * time.Hour
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Check immediately on startup
	if err := cm.CheckAllCertificatesExpiration(ctx); err != nil {
		cm.log.Warnf("initial expiration check failed: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				cm.log.Debug("stopping periodic expiration check")
				return
			case <-ticker.C:
				if err := cm.CheckAllCertificatesExpiration(ctx); err != nil {
					cm.log.Warnf("periodic expiration check failed: %v", err)
				}
			}
		}
	}()
}

// CheckExpiredCertificatesOnStartup checks for expired certificates on agent startup.
// This should be called during agent initialization.
func (cm *CertManager) CheckExpiredCertificatesOnStartup(ctx context.Context) error {
	if cm.lifecycleManager == nil {
		return nil // No lifecycle manager - skip check
	}

	cm.log.Debug("Checking for expired certificates on startup")

	// Check all certificates
	if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
		cm.log.Warnf("Failed to check expired certificates on startup: %v", err)
		return err
	}

	return nil
}

// StartExpirationMonitoring starts a goroutine that periodically checks for expired certificates.
func (cm *CertManager) StartExpirationMonitoring(ctx context.Context) {
	if cm.lifecycleManager == nil {
		return // No lifecycle manager - skip monitoring
	}

	checkInterval := 24 * time.Hour // Default: check daily
	if cm.config != nil {
		checkInterval = time.Duration(cm.config.Certificate.Renewal.CheckInterval)
		if checkInterval == 0 {
			checkInterval = 24 * time.Hour
		}
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		// Check immediately on startup
		if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
			cm.log.Warnf("Failed to check expired certificates: %v", err)
		}

		for {
			select {
			case <-ctx.Done():
				cm.log.Debug("Expiration monitoring stopped")
				return
			case <-ticker.C:
				if err := cm.lifecycleManager.CheckExpiredCertificates(ctx); err != nil {
					cm.log.Warnf("Failed to check expired certificates: %v", err)
				}
			}
		}
	}()
}
