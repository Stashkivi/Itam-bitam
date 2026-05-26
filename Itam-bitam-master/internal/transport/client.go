// Package transport handles mTLS HTTP communication with the ITAM server.
package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/itam/agent/internal/config"
)

// Client is a thread-safe ITAM server client. It uses mTLS for transport-layer
// identity and a short-lived JWT for application-layer authorization.
type Client struct {
	baseURL      string
	httpClient   *http.Client
	accessToken  string
	refreshToken string
	tokenExpiry  time.Time
	cfg          *config.Config
	cfgPath      string
	mu           sync.Mutex
}

// New constructs a Client from the enrolled state in cfg. The TLS client
// certificate is loaded from the paths stored during enrollment.
func New(cfg *config.Config, cfgPath string) (*Client, error) {
	enrolled := cfg.Enrolled
	tlsCfg, err := buildTLSConfig(
		enrolled.CertPath,
		enrolled.KeyPath,
		enrolled.CACertPath,
		cfg.Server.TLSSkipVerify,
	)
	if err != nil {
		return nil, fmt.Errorf("tls config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
		IdleConnTimeout: 90 * time.Second,
	}

	return &Client{
		baseURL:      cfg.Server.BaseURL,
		httpClient:   &http.Client{Transport: transport, Timeout: 30 * time.Second},
		accessToken:  enrolled.AccessToken,
		refreshToken: enrolled.RefreshToken,
		cfg:          cfg,
		cfgPath:      cfgPath,
	}, nil
}

// Post marshals body to JSON and sends it to path, refreshing the JWT if needed.
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*http.Response, error) {
	token, err := c.validToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("token: %w", err)
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	return c.httpClient.Do(req)
}

// IsReachable performs a lightweight HEAD request to /healthz.
func (c *Client) IsReachable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, c.baseURL+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// validToken returns a non-expired access token, refreshing it proactively
// when fewer than 2 minutes remain.
func (c *Client) validToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Until(c.tokenExpiry) > 2*time.Minute {
		return c.accessToken, nil
	}
	return c.refresh(ctx)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (c *Client) refresh(ctx context.Context) (string, error) {
	data, _ := json.Marshal(refreshRequest{RefreshToken: c.refreshToken})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/auth/refresh", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("refresh: status %d", resp.StatusCode)
	}

	var rr refreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return "", err
	}

	c.accessToken = rr.AccessToken
	c.refreshToken = rr.RefreshToken
	c.tokenExpiry = jwtExpiry(rr.AccessToken)

	// Persist updated tokens so a restart picks them up.
	c.cfg.Enrolled.AccessToken = rr.AccessToken
	c.cfg.Enrolled.RefreshToken = rr.RefreshToken
	config.Save(c.cfg, c.cfgPath) //nolint:errcheck — best-effort persist

	return c.accessToken, nil
}

// jwtExpiry parses the exp claim without verifying the signature.
// Signature is verified by the server on every request; we only need
// the expiry time locally to decide when to refresh proactively.
func jwtExpiry(tokenStr string) time.Time {
	p := jwt.NewParser()
	claims := jwt.MapClaims{}
	p.ParseUnverified(tokenStr, claims) //nolint:errcheck
	t, _ := claims.GetExpirationTime()
	if t == nil {
		return time.Now().Add(15 * time.Minute)
	}
	return t.Time
}

func buildTLSConfig(certPath, keyPath, caPath string, skipVerify bool) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: skipVerify, //nolint:gosec — controlled by explicit config flag
	}

	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if caPath != "" {
		caPEM, err := os.ReadFile(caPath)
		if err != nil {
			return nil, fmt.Errorf("read CA: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("invalid CA cert in %s", caPath)
		}
		tlsCfg.RootCAs = pool
	}

	return tlsCfg, nil
}

// DecodeJSON is a convenience helper to read a JSON response body.
func DecodeJSON(resp *http.Response, dst interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, dst)
}
