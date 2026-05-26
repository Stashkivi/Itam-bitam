package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/itam/server/internal/enrollment"
)

// adminMiddleware checks a static bearer token stored in ITAM_ADMIN_TOKEN.
func adminMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != token {
				writeError(w, http.StatusUnauthorized, "invalid admin token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MountAdmin registers all /v1/admin/* routes onto the given router.
func MountAdmin(r chi.Router, deps *Deps, adminToken string) {
	r.Route("/v1/admin", func(r chi.Router) {
		r.Use(adminMiddleware(adminToken))

		// Enrollment token management
		r.Post("/tokens", handleCreateToken(deps))
		r.Get("/tokens", handleListTokens(deps))

		// Host management
		r.Get("/hosts", handleAdminListHosts(deps))
		r.Put("/hosts/{hostId}/baseline", handleApproveBaseline(deps))

		// Alert channel management
		r.Get("/channels", handleListChannels(deps))
		r.Post("/channels", handleCreateChannel(deps))
		r.Delete("/channels/{channelId}", handleDeleteChannel(deps))
	})
}

// ── Enrollment tokens ────────────────────────────────────────────────────────

func handleCreateToken(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrgID string `json:"org_id"`
			Label string `json:"label"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OrgID == "" {
			writeError(w, http.StatusBadRequest, "org_id required")
			return
		}

		tok, err := enrollment.GenerateToken(r.Context(), deps.DB, body.OrgID, body.Label, 24*time.Hour*7, deps.HMACSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{
			"token": tok,
			"note":  "share with agent; single-use only",
		})
	}
}

func handleListTokens(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.DB.Pool.Query(r.Context(), `
			SELECT id, org_id, label, used, created_at
			FROM enrollment_tokens
			ORDER BY created_at DESC
			LIMIT 100
		`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type row struct {
			ID        string `json:"id"`
			OrgID     string `json:"org_id"`
			Label     string `json:"label"`
			Used      bool   `json:"used"`
			CreatedAt string `json:"created_at"`
		}
		var out []row
		for rows.Next() {
			var rr row
			if err := rows.Scan(&rr.ID, &rr.OrgID, &rr.Label, &rr.Used, &rr.CreatedAt); err == nil {
				out = append(out, rr)
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ── Hosts ────────────────────────────────────────────────────────────────────

func handleAdminListHosts(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.URL.Query().Get("org_id")
		rows, err := deps.DB.Pool.Query(r.Context(), `
			SELECT id, org_id, hostname, os_family, arch, ip_primary,
			       enrolled_at, last_seen_at, status
			FROM hosts
			WHERE ($1 = '' OR org_id = $1)
			ORDER BY org_id, hostname`, orgID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type hostRow struct {
			ID         string  `json:"id"`
			OrgID      string  `json:"org_id"`
			Hostname   string  `json:"hostname"`
			OSFamily   string  `json:"os_family"`
			Arch       string  `json:"arch"`
			IPPrimary  *string `json:"ip_primary"`
			EnrolledAt string  `json:"enrolled_at"`
			LastSeenAt *string `json:"last_seen_at"`
			Status     string  `json:"status"`
		}
		var out []hostRow
		for rows.Next() {
			var h hostRow
			if err := rows.Scan(&h.ID, &h.OrgID, &h.Hostname, &h.OSFamily, &h.Arch,
				&h.IPPrimary, &h.EnrolledAt, &h.LastSeenAt, &h.Status); err == nil {
				out = append(out, h)
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleApproveBaseline(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostID := chi.URLParam(r, "hostId")
		if hostID == "" {
			writeError(w, http.StatusBadRequest, "hostId required")
			return
		}
		raw, _, err := deps.DB.LatestSnapshot(r.Context(), hostID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if raw == nil {
			writeError(w, http.StatusNotFound, "no snapshot for host")
			return
		}
		if err := deps.DB.SetBaseline(r.Context(), hostID, "admin", 0, raw); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "baseline approved"})
	}
}

// ── Alert channels ────────────────────────────────────────────────────────────

func handleListChannels(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := deps.DB.Pool.Query(r.Context(), `
			SELECT id, org_id, channel_type, config, enabled, created_at
			FROM alert_channels
			ORDER BY created_at DESC
		`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		type channel struct {
			ID          string          `json:"id"`
			OrgID       string          `json:"org_id"`
			ChannelType string          `json:"channel_type"`
			Config      json.RawMessage `json:"config"`
			Enabled     bool            `json:"enabled"`
			CreatedAt   string          `json:"created_at"`
		}
		var out []channel
		for rows.Next() {
			var c channel
			if err := rows.Scan(&c.ID, &c.OrgID, &c.ChannelType, &c.Config, &c.Enabled, &c.CreatedAt); err == nil {
				out = append(out, c)
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleCreateChannel(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrgID       string          `json:"org_id"`
			ChannelType string          `json:"channel_type"`
			Config      json.RawMessage `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.OrgID == "" || (body.ChannelType != "slack" && body.ChannelType != "telegram") {
			writeError(w, http.StatusBadRequest, "org_id and channel_type (slack|telegram) required")
			return
		}

		var id string
		err := deps.DB.Pool.QueryRow(r.Context(), `
			INSERT INTO alert_channels (org_id, channel_type, config, enabled)
			VALUES ($1, $2, $3, true)
			RETURNING id
		`, body.OrgID, body.ChannelType, body.Config).Scan(&id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id})
	}
}

func handleDeleteChannel(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := chi.URLParam(r, "channelId")
		_, err := deps.DB.Pool.Exec(r.Context(),
			`DELETE FROM alert_channels WHERE id = $1`, channelID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

