package middleware

import (
	"context"
	"monitoring_backend/internal/auth"
	"net/http"
	"strings"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxRole   ctxKey = "role"
)

func JWT(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					http.Error(w, "authorization required", http.StatusUnauthorized)
					return
				}
				tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
				if tokenStr == authHeader {
					http.Error(w, "invalid auth header", http.StatusUnauthorized)
					return
				}

				claims, err := jwtManager.Parse(tokenStr)
				if err != nil {
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
				ctx = context.WithValue(ctx, ctxRole, claims.Role)

				next.ServeHTTP(w, r.WithContext(ctx))
			})
	}
}
