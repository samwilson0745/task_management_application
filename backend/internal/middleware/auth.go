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

func bearerToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", true
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		httpx.Error(w, http.StatusUnauthorized, "invalid authorization header")
		return "", false
	}
	return parts[1], true
}

func serve(next http.Handler, w http.ResponseWriter, r *http.Request, claims *auth.Claims) {
	ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// Auth requires a session JWT in the Authorization header. Tokens are never
// accepted from the URL.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := bearerToken(w, r)
			if !ok {
				return
			}
			if tokenStr == "" {
				httpx.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			claims, err := auth.ParseToken(secret, tokenStr)
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			serve(next, w, r, claims)
		})
	}
}

// TicketAuth is for endpoints that browsers cannot send headers to (EventSource).
// It accepts a session JWT in the Authorization header, or a short-lived
// single-purpose ticket via the ?ticket= query parameter.
func TicketAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := bearerToken(w, r)
			if !ok {
				return
			}
			var claims *auth.Claims
			var err error
			switch {
			case tokenStr != "":
				claims, err = auth.ParseToken(secret, tokenStr)
			case r.URL.Query().Get("ticket") != "":
				claims, err = auth.ParseTicket(secret, r.URL.Query().Get("ticket"))
			default:
				httpx.Error(w, http.StatusUnauthorized, "missing authorization")
				return
			}
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			serve(next, w, r, claims)
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
