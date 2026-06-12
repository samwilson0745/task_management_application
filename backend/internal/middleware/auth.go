package middleware

import (
	"context"
	"net/http"
	"strings"

	"taskmanager/internal/auth"
	"taskmanager/internal/httpx"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UserRoleKey contextKey = "user_role"
)

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			header := r.Header.Get("Authorization")
			if header != "" {
				parts := strings.SplitN(header, " ", 2)
				if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
					httpx.Error(w, http.StatusUnauthorized, "invalid authorization header")
					return
				}
				tokenStr = parts[1]
			} else if q := r.URL.Query().Get("token"); q != "" {
				// EventSource cannot set custom headers, so allow the token as a
				// query param for the SSE stream endpoint.
				tokenStr = q
			} else {
				httpx.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			claims, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func UserRoleFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(UserRoleKey).(string); ok {
		return v
	}
	return ""
}
