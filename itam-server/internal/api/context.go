package api

import (
	"context"

	"github.com/itam/server/internal/enrollment"
)

type contextKey int

const claimsKey contextKey = 1

func withClaims(ctx context.Context, claims *enrollment.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

func claimsFromCtx(ctx context.Context) *enrollment.Claims {
	c, _ := ctx.Value(claimsKey).(*enrollment.Claims)
	return c
}
