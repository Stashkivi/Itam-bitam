package transport

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/itam/agent/internal/config"
	"github.com/itam/agent/pkg/model"
)

// Enroll performs the full first-run enrollment handshake:
//   1. Generate Ed25519 keypair
//   2. POST /v1/enroll with the public key and enrollment token
//   3. Persist the signed certificate, CA cert, and tokens to dataDir
//   4. Update cfg.Enrolled in place (caller must call config.Save)
func Enroll(ctx context.Context, cfg *config.Config) error {
	dataDir := config.DataDir(cfg)
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("keygen: %w", err)
	}

	pubPEM, err := marshalPublicKeyPEM(pub)
	if err != nil {
		return err
	}

	hostInfo, err := collectBasicHostInfo()
	if err != nil {
		return err
	}

	req := model.EnrollRequest{
		HostUUID:        cfg.Enrolled.HostUUID,
		PublicKeyPEM:    pubPEM,
		EnrollmentToken: cfg.Server.EnrollmentToken,
		Hostname:        hostInfo.Hostname,
		OS:              hostInfo.OS,
		Arch:            hostInfo.Arch,
	}

	resp, err := postEnroll(ctx, cfg.Server.BaseURL, cfg.Server.CACertPath, req)
	if err != nil {
		return fmt.Errorf("enroll request: %w", err)
	}

	// Write cert, key, and CA cert to the data directory.
	certPath := filepath.Join(dataDir, "agent.crt")
	keyPath := filepath.Join(dataDir, "agent.key")
	caPath := filepath.Join(dataDir, "ca.crt")

	if err := os.WriteFile(certPath, []byte(resp.CertPEM), 0600); err != nil {
		return err
	}
	if err := writePrivKeyPEM(keyPath, priv); err != nil {
		return err
	}
	if err := os.WriteFile(caPath, []byte(resp.CACertPEM), 0644); err != nil {
		return err
	}

	cfg.Enrolled = config.EnrolledState{
		HostUUID:     resp.HostUUID,
		CertPath:     certPath,
		KeyPath:      keyPath,
		CACertPath:   caPath,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ScanCron:     resp.ScanCron,
	}
	if cfg.Enrolled.ScanCron == "" {
		cfg.Enrolled.ScanCron = cfg.Agent.ScanCron
	}

	return nil
}

// postEnroll sends the EnrollRequest using plain TLS (server CA is pinned
// via the bundled CA cert). No client cert is presented — it is the result
// of this call.
func postEnroll(ctx context.Context, baseURL, caCertPath string, req model.EnrollRequest) (*model.EnrollResponse, error) {
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13}

	if caCertPath != "" {
		caPEM, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("read server CA: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(caPEM)
		tlsCfg.RootCAs = pool
	}

	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
		Timeout:   30 * time.Second,
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/enroll", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	var enrolled model.EnrollResponse
	if err := DecodeJSON(resp, &enrolled); err != nil {
		return nil, err
	}
	return &enrolled, nil
}

func marshalPublicKeyPEM(pub ed25519.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})), nil
}

func writePrivKeyPEM(path string, priv ed25519.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	return os.WriteFile(path, data, 0600)
}

// collectBasicHostInfo gathers the minimum fields required by the enroll request.
func collectBasicHostInfo() (model.HostInfo, error) {
	hostname, _ := os.Hostname()
	return model.HostInfo{
		Hostname: hostname,
		OS:       detectOS(),
	}, nil
}
