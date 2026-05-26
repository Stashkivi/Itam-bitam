package enrollment

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the application-layer JWT payload attached to every agent request.
type Claims struct {
	HostUUID string `json:"host_uuid"`
	OrgID    string `json:"org_id"`
	jwt.RegisteredClaims
}

// IssueAccessToken creates a short-lived JWT for API authorization.
func IssueAccessToken(hostUUID, orgID string, ttl time.Duration, secret []byte) (string, error) {
	claims := Claims{
		HostUUID: hostUUID,
		OrgID:    orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   hostUUID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			ID:        uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// IssueRefreshToken creates a longer-lived opaque token (also a signed JWT).
func IssueRefreshToken(hostUUID, orgID string, ttl time.Duration, secret []byte) (string, error) {
	claims := Claims{
		HostUUID: hostUUID,
		OrgID:    orgID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   hostUUID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			ID:        "refresh:" + uuid.NewString(),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

// ValidateToken verifies signature and expiry, returning the embedded Claims.
func ValidateToken(tokenStr string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
