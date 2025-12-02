package load

import (
	"context"
	"crypto"
	"encoding/base32"
	"fmt"
	"strings"
	"sync"
	"time"

	api "github.com/flightctl/flightctl/api/v1beta1"
	apiclient "github.com/flightctl/flightctl/internal/api/client"
	fcrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

// DeviceSimulator simulates multiple devices for load testing
type DeviceSimulator struct {
	client           *apiclient.ClientWithResponses
	ctx              context.Context
	logger           *logrus.Logger
	devices          []*SimulatedDevice
	deviceMutex      sync.RWMutex
	metricsCollector *MetricsCollector
}

// SimulatedDevice represents a simulated device
type SimulatedDevice struct {
	ID           string
	Name         string
	PublicKey    crypto.PublicKey
	PrivateKey   crypto.PrivateKey
	CSR          []byte
	Enrolled     bool
	Certificate  *api.CertificateSigningRequest
	LastRenewal  time.Time
	RenewalCount int
}

// LoadPattern defines the pattern for generating load
type LoadPattern struct {
	Type            string        // "concurrent", "staggered", "burst"
	DeviceCount     int           // Number of devices to simulate
	RenewalInterval time.Duration // Interval between renewals for staggered pattern
	BurstSize       int           // Number of concurrent requests in burst pattern
	BurstInterval   time.Duration // Interval between bursts
}

// NewDeviceSimulator creates a new device simulator
func NewDeviceSimulator(ctx context.Context, client *apiclient.ClientWithResponses, logger *logrus.Logger) *DeviceSimulator {
	return &DeviceSimulator{
		client:           client,
		ctx:              ctx,
		logger:           logger,
		devices:          make([]*SimulatedDevice, 0),
		metricsCollector: nil, // Set via SetMetricsCollector
	}
}

// SetMetricsCollector sets the metrics collector for the simulator
func (ds *DeviceSimulator) SetMetricsCollector(collector *MetricsCollector) {
	ds.deviceMutex.Lock()
	defer ds.deviceMutex.Unlock()
	ds.metricsCollector = collector
}

// GenerateDevices generates a specified number of simulated devices
func (ds *DeviceSimulator) GenerateDevices(count int) error {
	ds.deviceMutex.Lock()
	defer ds.deviceMutex.Unlock()

	ds.logger.Infof("Generating %d simulated devices", count)

	for i := 0; i < count; i++ {
		device, err := ds.createSimulatedDevice()
		if err != nil {
			return fmt.Errorf("failed to create simulated device %d: %w", i, err)
		}
		ds.devices = append(ds.devices, device)
	}

	ds.logger.Infof("Generated %d simulated devices", len(ds.devices))
	return nil
}

// createSimulatedDevice creates a single simulated device with key pair and CSR
func (ds *DeviceSimulator) createSimulatedDevice() (*SimulatedDevice, error) {
	publicKey, privateKey, err := fcrypto.NewKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	publicKeyHash, err := fcrypto.HashPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash public key: %w", err)
	}

	deviceName := strings.ToLower(base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(publicKeyHash))

	csrPEM, err := fcrypto.MakeCSR(privateKey.(crypto.Signer), deviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSR: %w", err)
	}

	device := &SimulatedDevice{
		ID:           uuid.New().String(),
		Name:         deviceName,
		PublicKey:    publicKey,
		PrivateKey:   privateKey,
		CSR:          csrPEM,
		Enrolled:     false,
		RenewalCount: 0,
	}

	return device, nil
}

// SimulateRenewalRequest simulates a certificate renewal request for a device
func (ds *DeviceSimulator) SimulateRenewalRequest(device *SimulatedDevice) error {
	if !device.Enrolled {
		return fmt.Errorf("device %s is not enrolled", device.Name)
	}

	csr := api.CertificateSigningRequest{
		ApiVersion: "v1beta1",
		Kind:       "CertificateSigningRequest",
		Metadata: api.ObjectMeta{
			Name: lo.ToPtr(device.Name),
			Labels: &map[string]string{
				"flightctl.io/renewal-reason": "proactive",
				"test":                        "load-test",
			},
		},
		Spec: api.CertificateSigningRequestSpec{
			Request:           device.CSR,
			SignerName:        "flightctl.io/device-svc-client",
			Usages:            &[]string{"clientAuth", "CA:false"},
			ExpirationSeconds: lo.ToPtr(int32(604800)), // 7 days
		},
	}

	startTime := time.Now()
	resp, err := ds.client.CreateCertificateSigningRequestWithResponse(ds.ctx, csr)
	duration := time.Since(startTime)

	// Record metrics
	if ds.metricsCollector != nil {
		ds.metricsCollector.RecordResponseTime(duration)
		if err != nil {
			ds.metricsCollector.RecordError(err)
		} else if resp.StatusCode() >= 400 {
			ds.metricsCollector.RecordError(fmt.Errorf("renewal request failed: status %d", resp.StatusCode()))
		} else {
			// Record certificate issuance when CSR is approved
			// In a real implementation, we would poll for CSR approval
			ds.metricsCollector.RecordCertIssuance()
		}
	}

	if err != nil {
		return fmt.Errorf("failed to submit renewal request for device %s: %w", device.Name, err)
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("renewal request failed for device %s: status %d", device.Name, resp.StatusCode())
	}

	device.LastRenewal = time.Now()
	device.RenewalCount++

	ds.logger.Debugf("Renewal request submitted for device %s in %v", device.Name, duration)
	return nil
}

// SimulateRecoveryRequest simulates a certificate recovery request for a device
func (ds *DeviceSimulator) SimulateRecoveryRequest(device *SimulatedDevice) error {
	if !device.Enrolled {
		return fmt.Errorf("device %s is not enrolled", device.Name)
	}

	csr := api.CertificateSigningRequest{
		ApiVersion: "v1beta1",
		Kind:       "CertificateSigningRequest",
		Metadata: api.ObjectMeta{
			Name: lo.ToPtr(device.Name),
			Labels: &map[string]string{
				"flightctl.io/renewal-reason": "expired",
				"test":                        "load-test",
			},
		},
		Spec: api.CertificateSigningRequestSpec{
			Request:           device.CSR,
			SignerName:        "flightctl.io/device-svc-client",
			Usages:            &[]string{"clientAuth", "CA:false"},
			ExpirationSeconds: lo.ToPtr(int32(604800)), // 7 days
		},
	}

	startTime := time.Now()
	resp, err := ds.client.CreateCertificateSigningRequestWithResponse(ds.ctx, csr)
	duration := time.Since(startTime)

	// Record metrics
	if ds.metricsCollector != nil {
		ds.metricsCollector.RecordResponseTime(duration)
		if err != nil {
			ds.metricsCollector.RecordError(err)
		} else if resp.StatusCode() >= 400 {
			ds.metricsCollector.RecordError(fmt.Errorf("recovery request failed: status %d", resp.StatusCode()))
		} else {
			// Record certificate issuance when CSR is approved
			ds.metricsCollector.RecordCertIssuance()
		}
	}

	if err != nil {
		return fmt.Errorf("failed to submit recovery request for device %s: %w", device.Name, err)
	}

	if resp.StatusCode() >= 400 {
		return fmt.Errorf("recovery request failed for device %s: status %d", device.Name, resp.StatusCode())
	}

	ds.logger.Debugf("Recovery request submitted for device %s in %v", device.Name, duration)
	return nil
}

// SimulateConcurrentRenewals simulates concurrent renewal requests
func (ds *DeviceSimulator) SimulateConcurrentRenewals(deviceCount int) error {
	ds.deviceMutex.RLock()
	devices := ds.devices[:deviceCount]
	if len(devices) > len(ds.devices) {
		devices = ds.devices
	}
	ds.deviceMutex.RUnlock()

	ds.logger.Infof("Simulating concurrent renewals for %d devices", len(devices))

	var wg sync.WaitGroup
	errors := make(chan error, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(d *SimulatedDevice) {
			defer wg.Done()
			if err := ds.SimulateRenewalRequest(d); err != nil {
				errors <- err
			}
		}(device)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		ds.logger.Errorf("Renewal error: %v", err)
		errorCount++
	}

	ds.logger.Infof("Concurrent renewals completed: %d successful, %d errors", len(devices)-errorCount, errorCount)
	return nil
}

// SimulateStaggeredRenewals simulates staggered renewal requests over time
func (ds *DeviceSimulator) SimulateStaggeredRenewals(deviceCount int, interval time.Duration) error {
	ds.deviceMutex.RLock()
	devices := ds.devices[:deviceCount]
	if len(devices) > len(ds.devices) {
		devices = ds.devices
	}
	ds.deviceMutex.RUnlock()

	ds.logger.Infof("Simulating staggered renewals for %d devices with interval %v", len(devices), interval)

	for i, device := range devices {
		if i > 0 {
			time.Sleep(interval)
		}
		if err := ds.SimulateRenewalRequest(device); err != nil {
			ds.logger.Errorf("Staggered renewal failed for device %s: %v", device.Name, err)
		}
	}

	ds.logger.Infof("Staggered renewals completed for %d devices", len(devices))
	return nil
}

// SimulateConcurrentRecoveries simulates concurrent recovery requests
func (ds *DeviceSimulator) SimulateConcurrentRecoveries(deviceCount int) error {
	ds.deviceMutex.RLock()
	devices := ds.devices[:deviceCount]
	if len(devices) > len(ds.devices) {
		devices = ds.devices
	}
	ds.deviceMutex.RUnlock()

	ds.logger.Infof("Simulating concurrent recoveries for %d devices", len(devices))

	var wg sync.WaitGroup
	errors := make(chan error, len(devices))

	for _, device := range devices {
		wg.Add(1)
		go func(d *SimulatedDevice) {
			defer wg.Done()
			if err := ds.SimulateRecoveryRequest(d); err != nil {
				errors <- err
			}
		}(device)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		ds.logger.Errorf("Recovery error: %v", err)
		errorCount++
	}

	ds.logger.Infof("Concurrent recoveries completed: %d successful, %d errors", len(devices)-errorCount, errorCount)
	return nil
}

// GetDeviceCount returns the number of simulated devices
func (ds *DeviceSimulator) GetDeviceCount() int {
	ds.deviceMutex.RLock()
	defer ds.deviceMutex.RUnlock()
	return len(ds.devices)
}

// GetDevices returns all simulated devices
func (ds *DeviceSimulator) GetDevices() []*SimulatedDevice {
	ds.deviceMutex.RLock()
	defer ds.deviceMutex.RUnlock()
	return ds.devices
}
