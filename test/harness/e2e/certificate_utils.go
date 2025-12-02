package e2e

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/flightctl/flightctl/api/v1beta1"
	agent_config "github.com/flightctl/flightctl/internal/agent/config"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	"github.com/sirupsen/logrus"
)

// CertificateInfo contains information about a certificate
type CertificateInfo struct {
	Fingerprint         string
	NotBefore           time.Time
	NotAfter            time.Time
	Subject             string
	IsExpired           bool
	DaysUntilExpiration int
}

// ReadCertificateFromVM reads a certificate file from the VM and parses it
func (h *Harness) ReadCertificateFromVM(certPath string) (*x509.Certificate, error) {
	if h.VM == nil {
		return nil, fmt.Errorf("VM is not initialized")
	}

	// Read certificate file from VM
	stdout, err := h.VM.RunSSH([]string{"sudo", "cat", certPath}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file %s: %w", certPath, err)
	}

	certPEM := stdout.Bytes()
	if len(certPEM) == 0 {
		return nil, fmt.Errorf("certificate file %s is empty", certPath)
	}

	// Parse certificate
	cert, err := fccrypto.ParsePEMCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", certPath, err)
	}

	return cert, nil
}

// GetManagementCertificate reads the management certificate from the VM
func (h *Harness) GetManagementCertificate() (*x509.Certificate, error) {
	certPath := fmt.Sprintf("/var/lib/flightctl/%s/%s", agent_config.DefaultCertsDirName, agent_config.GeneratedCertFile)
	return h.ReadCertificateFromVM(certPath)
}

// GetBootstrapCertificate reads the bootstrap certificate from the VM
func (h *Harness) GetBootstrapCertificate() (*x509.Certificate, error) {
	certPath := fmt.Sprintf("/var/lib/flightctl/%s/%s", agent_config.DefaultCertsDirName, agent_config.EnrollmentCertFile)
	return h.ReadCertificateFromVM(certPath)
}

// GetCertificateInfo extracts information from a certificate
func GetCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	if cert == nil {
		return nil
	}

	// Calculate SHA256 fingerprint
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintHex := hex.EncodeToString(fingerprint[:])

	now := time.Now()
	isExpired := now.After(cert.NotAfter)
	daysUntilExpiration := 0
	if !isExpired {
		daysUntilExpiration = int(cert.NotAfter.Sub(now).Hours() / 24)
	}

	return &CertificateInfo{
		Fingerprint:         fingerprintHex,
		NotBefore:           cert.NotBefore,
		NotAfter:            cert.NotAfter,
		Subject:             cert.Subject.CommonName,
		IsExpired:           isExpired,
		DaysUntilExpiration: daysUntilExpiration,
	}
}

// CompareCertificates compares two certificates by fingerprint
func CompareCertificates(cert1, cert2 *x509.Certificate) bool {
	if cert1 == nil || cert2 == nil {
		return false
	}

	info1 := GetCertificateInfo(cert1)
	info2 := GetCertificateInfo(cert2)

	return info1.Fingerprint == info2.Fingerprint
}

// WaitForCertificateRenewal waits for a certificate to be renewed (fingerprint changes)
func (h *Harness) WaitForCertificateRenewal(originalCert *x509.Certificate, timeout time.Duration, polling time.Duration) (*x509.Certificate, error) {
	if originalCert == nil {
		return nil, fmt.Errorf("original certificate is nil")
	}

	originalInfo := GetCertificateInfo(originalCert)
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		currentCert, err := h.GetManagementCertificate()
		if err != nil {
			logrus.Debugf("Failed to read certificate while waiting for renewal: %v", err)
			time.Sleep(polling)
			continue
		}

		currentInfo := GetCertificateInfo(currentCert)
		if currentInfo.Fingerprint != originalInfo.Fingerprint {
			logrus.Infof("Certificate renewed: old fingerprint %s, new fingerprint %s",
				originalInfo.Fingerprint[:16], currentInfo.Fingerprint[:16])
			return currentCert, nil
		}

		time.Sleep(polling)
	}

	return nil, fmt.Errorf("timeout waiting for certificate renewal after %v", timeout)
}

// CheckCertificateExists checks if a certificate file exists on the VM
func (h *Harness) CheckCertificateExists(certPath string) (bool, error) {
	if h.VM == nil {
		return false, fmt.Errorf("VM is not initialized")
	}

	stdout, err := h.VM.RunSSH([]string{"sudo", "test", "-f", certPath, "&&", "echo", "exists", "||", "echo", "notfound"}, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check certificate file %s: %w", certPath, err)
	}

	result := stdout.String()
	return result == "exists\n", nil
}

// GetCertificateRenewalCount gets the renewal count from device status
func (h *Harness) GetCertificateRenewalCount(deviceID string) (int, error) {
	device, err := h.GetDevice(deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device: %w", err)
	}

	if device == nil || device.Status == nil {
		return 0, fmt.Errorf("device status is nil")
	}

	// Certificate renewal count is stored in SystemInfo.CustomInfo
	// This is a temporary approach until CertificateStatus is added to DeviceStatus
	if device.Status.SystemInfo.CustomInfo != nil {
		if countStr, ok := (*device.Status.SystemInfo.CustomInfo)["certificate_renewal_count"]; ok {
			var count int
			if _, err := fmt.Sscanf(countStr, "%d", &count); err == nil {
				return count, nil
			}
		}
	}

	return 0, nil
}

// GetCertificateSigningRequest retrieves a CSR by name
func (h *Harness) GetCertificateSigningRequest(csrName string) (*v1beta1.CertificateSigningRequest, error) {
	resp, err := h.Client.GetCertificateSigningRequestWithResponse(h.Context, csrName)
	if err != nil {
		return nil, fmt.Errorf("failed to get CSR %s: %w", csrName, err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to get CSR %s: status code %d", csrName, resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("CSR %s not found", csrName)
	}

	return resp.JSON200, nil
}

// WaitForCSRApproval waits for a CSR to be approved and certificate to be issued
func (h *Harness) WaitForCSRApproval(csrName string, timeout time.Duration, polling time.Duration) ([]byte, error) {
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		csr, err := h.GetCertificateSigningRequest(csrName)
		if err != nil {
			logrus.Debugf("Failed to get CSR while waiting for approval: %v", err)
			time.Sleep(polling)
			continue
		}

		if csr.Status != nil && csr.Status.Certificate != nil && len(*csr.Status.Certificate) > 0 {
			logrus.Infof("CSR %s approved and certificate issued", csrName)
			return *csr.Status.Certificate, nil
		}

		// Check if CSR is denied
		if csr.Status != nil && csr.Status.Conditions != nil {
			for _, condition := range csr.Status.Conditions {
				if condition.Type == "Denied" && condition.Status == "True" {
					return nil, fmt.Errorf("CSR %s was denied: %s", csrName, condition.Message)
				}
			}
		}

		time.Sleep(polling)
	}

	return nil, fmt.Errorf("timeout waiting for CSR approval after %v", timeout)
}
