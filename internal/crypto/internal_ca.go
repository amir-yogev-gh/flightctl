package crypto

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/flightctl/flightctl/internal/config/ca"
	fccrypto "github.com/flightctl/flightctl/pkg/crypto"
	oscrypto "github.com/openshift/library-go/pkg/crypto"
)

type internalCA struct {
	Config          *TLSCertificateConfig
	SerialGenerator oscrypto.SerialGenerator
}

func ensureInternalCA(cfg *ca.Config) (CABackend, bool, error) {

	caCertFile := CertStorePath(cfg.InternalConfig.CertFile, cfg.InternalConfig.CertStore)
	caKeyFile := CertStorePath(cfg.InternalConfig.KeyFile, cfg.InternalConfig.CertStore)
	caSerialFile := cfg.InternalConfig.SerialFile
	if len(cfg.InternalConfig.SerialFile) > 0 {
		caSerialFile = CertStorePath(cfg.InternalConfig.SerialFile, cfg.InternalConfig.CertStore)
	}
	ca, err := GetCA(caCertFile, caKeyFile, caSerialFile)
	if err == nil {
		return ca, false, err
	}
	ca, err = MakeSelfSignedCA(caCertFile, caKeyFile, caSerialFile, cfg.InternalConfig.SignerCertName, cfg.InternalConfig.CertValidityDays)
	if err != nil {
		return nil, false, err
	}
	return ca, true, err
}

func GetCA(certFile, keyFile, serialFile string) (*internalCA, error) {
	ca, err := oscrypto.GetCA(certFile, keyFile, serialFile)
	if err != nil {
		return nil, err
	}
	config := TLSCertificateConfig(*ca.Config)
	return &internalCA{Config: &config, SerialGenerator: ca.SerialGenerator}, nil
}

func MakeSelfSignedCA(certFile, keyFile, serialFile, subjectName string, expiryDays int) (*internalCA, error) {

	var serialGenerator oscrypto.SerialGenerator
	var err error
	if len(serialFile) > 0 {
		// create / overwrite the serial file with a zero padded hex value (ending in a newline to have a valid file)
		if err := os.WriteFile(serialFile, []byte("00\n"), 0600); err != nil {
			return nil, err
		}
		serialGenerator, err = oscrypto.NewSerialFileGenerator(serialFile)
		if err != nil {
			return nil, err
		}
	} else {
		serialGenerator = &oscrypto.RandomSerialGenerator{}
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
	}

	caSerial, err := serialGenerator.Next(template)
	if err != nil {
		return nil, err
	}

	caConfig, err := makeSelfSignedCAConfig(
		pkix.Name{CommonName: subjectName},
		time.Duration(expiryDays)*24*time.Hour,
		caSerial,
	)
	if err != nil {
		return nil, err
	}
	if err = caConfig.WriteCertConfigFile(certFile, keyFile); err != nil {
		return nil, err
	}

	config := TLSCertificateConfig(*caConfig)
	return &internalCA{
		SerialGenerator: serialGenerator,
		Config:          &config,
	}, nil
}

func makeSelfSignedCAConfig(subject pkix.Name, caLifetime time.Duration, serial int64) (*oscrypto.TLSCertificateConfig, error) {
	rootcaPublicKey, rootcaPrivateKey, publicKeyHash, err := fccrypto.NewKeyPairWithHash()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	rootcaTemplate := &x509.Certificate{
		Subject: subject,

		SignatureAlgorithm: x509.ECDSAWithSHA256,

		NotBefore: now.Add(-1 * time.Second),
		NotAfter:  now.Add(caLifetime),

		SerialNumber: big.NewInt(serial),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,

		AuthorityKeyId: publicKeyHash,
		SubjectKeyId:   publicKeyHash,
	}
	rootcaCert, err := signCertificate(rootcaTemplate, rootcaPublicKey, rootcaTemplate, rootcaPrivateKey)
	if err != nil {
		return nil, err
	}
	caConfig := &oscrypto.TLSCertificateConfig{
		Certs: []*x509.Certificate{rootcaCert},
		Key:   rootcaPrivateKey,
	}
	return caConfig, nil
}

func (caBackend *internalCA) signCertificate(template *x509.Certificate, requestKey crypto.PublicKey) (*x509.Certificate, error) {
	// Increment and persist serial
	serial, err := caBackend.SerialGenerator.Next(template)
	if err != nil {
		return nil, err
	}
	template.SerialNumber = big.NewInt(serial)
	return signCertificate(template, requestKey, caBackend.Config.Certs[0], caBackend.Config.Key)
}

func signCertificate(template *x509.Certificate, requestKey crypto.PublicKey, issuer *x509.Certificate, issuerKey crypto.PrivateKey) (*x509.Certificate, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, template, issuer, requestKey, issuerKey)
	if err != nil {
		return nil, err
	}
	certs, err := x509.ParseCertificates(derBytes)
	if err != nil {
		return nil, err
	}
	if len(certs) != 1 {
		return nil, errors.New("expected a single certificate")
	}
	return certs[0], nil
}

// IssueRequestedClientCertificate issues a client certificate based on the provided
// Certificate Signing Request (CSR) and the desired expiration time in seconds.
// This currently processes both enrollment cert and management cert signing requests, which both are signed
// by the FC service's internal CA instance named 'ca'.
func (caBackend *internalCA) IssueRequestedCertificateAsX509(ctx context.Context, csr *x509.CertificateRequest, expirySeconds int, usage []x509.ExtKeyUsage, opts ...CertOption) (*x509.Certificate, error) {
	now := time.Now()
	expire := time.Duration(expirySeconds) * time.Second
	// Note Subject (and other fields where applicable) validation is performed by the callers.
	// This routine will sign what it is given, length checks and other validation should happen
	// further up the call chain.
	template := &x509.Certificate{
		Subject:               csr.Subject,
		Signature:             csr.Signature,
		SignatureAlgorithm:    csr.SignatureAlgorithm,
		PublicKey:             csr.PublicKey,
		PublicKeyAlgorithm:    csr.PublicKeyAlgorithm,
		IPAddresses:           csr.IPAddresses,
		DNSNames:              csr.DNSNames,
		Issuer:                caBackend.Config.Certs[0].Subject,
		NotBefore:             now.Add(-time.Second),
		NotAfter:              now.Add(expire),
		SerialNumber:          big.NewInt(1),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           usage,
		BasicConstraintsValid: true,
		AuthorityKeyId:        caBackend.Config.Certs[0].SubjectKeyId,
	}

	for _, opt := range opts {
		if err := opt(template); err != nil {
			return nil, fmt.Errorf("applying cert option: %w", err)
		}
	}
	return caBackend.signCertificate(template, csr.PublicKey)
}

func (caBackend *internalCA) GetCABundleX509() []*x509.Certificate {
	return caBackend.Config.Certs
}
