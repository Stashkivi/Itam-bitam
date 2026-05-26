// Package api owns the HTTP router and middleware chain.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/enrollment"
	"github.com/itam/server/internal/hub"
	"github.com/itam/server/internal/kafka"
)

// Deps bundles all shared dependencies the handlers need.
type Deps struct {
	DB         *postgres.DB
	Producer   *kafka.Producer
	Hub        *hub.Hub
	CA         *enrollment.CA
	JWTSecret  []byte
	HMACSecret []byte
	AdminToken string
	JWTCfg     struct {
		AccessTTLMin   int
		RefreshTTLHour int
	}
}

// NewRouter builds and returns the fully-wired chi router.
func NewRouter(deps *Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public endpoints (no auth required).
	r.Get("/healthz", handleHealth)
	r.Post("/v1/enroll", handleEnroll(deps))
	r.Post("/v1/auth/refresh", handleRefresh(deps))

	// Authenticated endpoints (mTLS + JWT).
	r.Group(func(r chi.Router) {
		r.Use(jwtMiddleware(deps.JWTSecret))
		r.Post("/v1/ingest/scan", handleIngestScan(deps))
		r.Post("/v1/ingest/heartbeat", handleHeartbeat(deps))
		r.Get("/v1/ws", handleWebSocket(deps))
	})

	// Admin endpoints (static bearer token).
	MountAdmin(r, deps, deps.AdminToken)

	return r
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// jwtMiddleware validates the Bearer token and injects claims into the context.
func jwtMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			claims, err := enrollment.ValidateToken(strings.TrimPrefix(auth, "Bearer "), secret)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			next.ServeHTTP(w, r.WithContext(withClaims(r.Context(), claims)))
		})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
