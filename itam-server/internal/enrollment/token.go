// Package enrollment handles the three-phase agent security lifecycle:
// HMAC token validation → certificate signing → JWT issuance.
package enrollment

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/itam/server/internal/db/postgres"
)

// GenerateToken creates a new admin enrollment token for orgID.
// The raw token is returned to the admin; only its HMAC hash is stored.
func GenerateToken(ctx context.Context, db *postgres.DB, orgID, label string, ttl time.Duration, hmacSecret []byte) (rawToken string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	rawToken = hex.EncodeToString(raw)
	tokenHash := hmacHex([]byte(rawToken), hmacSecret)

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO enrollment_tokens (token_hash, org_id, label, expires_at)
		VALUES ($1, $2, $3, $4)`,
		tokenHash, orgID, label, time.Now().Add(ttl),
	)
	return rawToken, err
}

// ValidateToken checks the submitted raw token against the stored HMAC hash.
// It marks the token as used atomically so replay attacks are impossible.
// Returns (orgID, nil) on success, ("", error) on failure.
func ValidateToken(ctx context.Context, db *postgres.DB, rawToken string, hmacSecret []byte) (orgID string, err error) {
	tokenHash := hmacHex([]byte(rawToken), hmacSecret)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var expiresAt time.Time
	var usedAt *time.Time
	var isRevoked bool

	err = tx.QueryRow(ctx, `
		SELECT org_id, expires_at, used_at, is_revoked
		FROM enrollment_tokens
		WHERE token_hash = $1
		FOR UPDATE`, tokenHash,
	).Scan(&orgID, &expiresAt, &usedAt, &isRevoked)
	if err != nil {
		return "", fmt.Errorf("token not found")
	}

	switch {
	case isRevoked:
		return "", fmt.Errorf("token revoked")
	case usedAt != nil:
		return "", fmt.Errorf("token already used")
	case time.Now().After(expiresAt):
		return "", fmt.Errorf("token expired")
	}

	if _, err := tx.Exec(ctx,
		`UPDATE enrollment_tokens SET used_at = now() WHERE token_hash = $1`, tokenHash,
	); err != nil {
		return "", err
	}

	return orgID, tx.Commit(ctx)
}

func hmacHex(data, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
