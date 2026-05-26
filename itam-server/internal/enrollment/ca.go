package enrollment

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// CA is a minimal in-process Certificate Authority. In production, replace
// with HashiCorp Vault PKI Secrets Engine or step-ca for HSM key protection.
type CA struct {
	cert    *x509.Certificate
	certPEM string
	key     *ecdsa.PrivateKey
	ttlDays int
}

// LoadOrCreateCA loads the CA cert+key from disk, or generates a new
// self-signed CA and persists it if the files don't exist.
func LoadOrCreateCA(certPath, keyPath string, ttlDays int) (*CA, error) {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return generateCA(certPath, keyPath, ttlDays)
	}
	return loadCA(certPath, keyPath, ttlDays)
}

func generateCA(certPath, keyPath string, ttlDays int) (*CA, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "ITAM Internal CA",
			Organization: []string{"ITAM"},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}

	return &CA{cert: cert, certPEM: string(certPEM), key: key, ttlDays: ttlDays}, nil
}

func loadCA(certPath, keyPath string, ttlDays int) (*CA, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(certData)
	if block == nil {
		return nil, fmt.Errorf("invalid CA cert PEM in %s", certPath)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("invalid CA key PEM in %s", keyPath)
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return &CA{cert: cert, certPEM: string(certData), key: key, ttlDays: ttlDays}, nil
}

// CACertPEM returns the CA certificate in PEM format for distribution to agents.
func (ca *CA) CACertPEM() string { return ca.certPEM }

// SignPublicKey issues a TLS client certificate for the supplied Ed25519
// public key (submitted by the agent during enrollment). The certificate CN
// is set to the host UUID for identity verification on mTLS connections.
func (ca *CA) SignPublicKey(pubKeyPEM, hostUUID string) (certPEM string, serial string, err error) {
	block, _ := pem.Decode([]byte(pubKeyPEM))
	if block == nil {
		return "", "", fmt.Errorf("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("parse public key: %w", err)
	}

	serialNum, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", err
	}

	tmpl := &x509.Certificate{
		SerialNumber: serialNum,
		Subject:      pkix.Name{CommonName: hostUUID},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().AddDate(0, 0, ca.ttlDays),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, pub, ca.key)
	if err != nil {
		return "", "", err
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return certPEM, serialNum.Text(16), nil
}
