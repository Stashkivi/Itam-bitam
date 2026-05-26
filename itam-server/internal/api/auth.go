package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/itam/server/internal/enrollment"
)

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func handleRefresh(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest

		// Accept token from body OR Authorization header for flexibility.
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			req.RefreshToken = strings.TrimPrefix(auth, "Bearer ")
		} else {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
				writeError(w, http.StatusBadRequest, "missing refresh_token")
				return
			}
		}

		claims, err := enrollment.ValidateToken(req.RefreshToken, deps.JWTSecret)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}

		// Reject access tokens submitted to the refresh endpoint.
		if !strings.HasPrefix(claims.ID, "refresh:") {
			writeError(w, http.StatusUnauthorized, "not a refresh token")
			return
		}

		accessTTL := time.Duration(deps.JWTCfg.AccessTTLMin) * time.Minute
		refreshTTL := time.Duration(deps.JWTCfg.RefreshTTLHour) * time.Hour

		newAccess, err := enrollment.IssueAccessToken(claims.HostUUID, claims.OrgID, accessTTL, deps.JWTSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token issuance failed")
			return
		}
		newRefresh, err := enrollment.IssueRefreshToken(claims.HostUUID, claims.OrgID, refreshTTL, deps.JWTSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token issuance failed")
			return
		}

		writeJSON(w, http.StatusOK, refreshResponse{
			AccessToken:  newAccess,
			RefreshToken: newRefresh,
		})
	}
}
