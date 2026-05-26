package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/model"
)

func handleIngestScan(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims == nil {
			writeError(w, http.StatusUnauthorized, "no claims")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10 MB cap
		if err != nil {
			writeError(w, http.StatusBadRequest, "read body")
			return
		}

		// Validate the schema minimally before queuing.
		var scan model.ScanPayload
		if err := json.Unmarshal(body, &scan); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "invalid scan payload")
			return
		}

		// The host_uuid in the payload must match the JWT subject.
		// This prevents one agent from submitting scans on behalf of another.
		if scan.HostUUID != claims.HostUUID {
			writeError(w, http.StatusForbidden, "host_uuid mismatch")
			return
		}

		topic := kafka.TopicScanFull
		if scan.PayloadType == "diff" {
			topic = kafka.TopicScanDiff
		}

		if err := deps.Producer.Publish(r.Context(), topic, scan.HostUUID, json.RawMessage(body)); err != nil {
			writeError(w, http.StatusServiceUnavailable, "queue unavailable")
			return
		}

		// Update last_seen_at immediately so the dashboard reflects live status.
		deps.DB.TouchLastSeen(r.Context(), claims.HostUUID) //nolint:errcheck

		w.WriteHeader(http.StatusAccepted)
	}
}

func handleHeartbeat(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims == nil {
			writeError(w, http.StatusUnauthorized, "no claims")
			return
		}

		deps.DB.TouchLastSeen(r.Context(), claims.HostUUID) //nolint:errcheck

		deps.Producer.Publish( //nolint:errcheck
			r.Context(), kafka.TopicHeartbeat, claims.HostUUID,
			map[string]string{"host_uuid": claims.HostUUID},
		)

		w.WriteHeader(http.StatusNoContent)
	}
}
