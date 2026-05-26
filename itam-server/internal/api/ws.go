package api

import (
	"net/http"
)

// handleWebSocket upgrades the connection and delegates lifecycle to the hub.
// The JWT middleware has already validated the token by this point.
func handleWebSocket(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims == nil {
			writeError(w, http.StatusUnauthorized, "no claims")
			return
		}

		if err := deps.Hub.Upgrade(w, r, claims.OrgID); err != nil {
			// Upgrade already sent an HTTP error response on failure.
			return
		}
		// The hub manages the connection lifecycle from here.
	}
}
