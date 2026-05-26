package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/enrollment"
)

type enrollRequest struct {
	HostUUID        string `json:"host_uuid"`
	PublicKeyPEM    string `json:"public_key_pem"`
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
}

type enrollResponse struct {
	HostUUID     string `json:"host_uuid"`
	CertPEM      string `json:"cert_pem"`
	CACertPEM    string `json:"ca_cert_pem"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ScanCron     string `json:"scan_cron"`
}

func handleEnroll(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req enrollRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// 1. Validate the HMAC enrollment token and consume it.
		orgID, err := enrollment.ValidateToken(r.Context(), deps.DB, req.EnrollmentToken, deps.HMACSecret)
		if err != nil {
			writeError(w, http.StatusForbidden, "invalid enrollment token: "+err.Error())
			return
		}

		// 2. Upsert the host record.
		hostID, err := deps.DB.UpsertHost(
			r.Context(), orgID, req.HostUUID,
			req.Hostname, req.OS, req.Arch, nil,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "host registration failed")
			return
		}

		// 3. Sign the agent's public key with the internal CA.
		certPEM, serial, err := deps.CA.SignPublicKey(req.PublicKeyPEM, hostID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cert signing failed")
			return
		}

		// 4. Store credential metadata.
		deps.DB.Pool.Exec(r.Context(), `
			INSERT INTO agent_credentials (host_id, cert_serial, cert_expires, public_key)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (host_id) DO UPDATE
			    SET cert_serial  = EXCLUDED.cert_serial,
			        cert_expires = EXCLUDED.cert_expires,
			        public_key   = EXCLUDED.public_key,
			        issued_at    = now()`,
			hostID, serial,
			time.Now().AddDate(0, 0, 365),
			req.PublicKeyPEM,
		) //nolint:errcheck

		// 5. Issue JWT pair.
		accessTTL := time.Duration(deps.JWTCfg.AccessTTLMin) * time.Minute
		refreshTTL := time.Duration(deps.JWTCfg.RefreshTTLHour) * time.Hour

		accessToken, err := enrollment.IssueAccessToken(hostID, orgID, accessTTL, deps.JWTSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token issuance failed")
			return
		}
		refreshToken, err := enrollment.IssueRefreshToken(hostID, orgID, refreshTTL, deps.JWTSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token issuance failed")
			return
		}

		writeJSON(w, http.StatusOK, enrollResponse{
			HostUUID:     hostID,
			CertPEM:      certPEM,
			CACertPEM:    deps.CA.CACertPEM(),
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ScanCron:     "0 2 * * *",
		})
	}
}

// handleEnroll references postgres.DB for credential storage.
var _ = (*postgres.DB)(nil)
