package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"status-page/packages/auth/proto"
)

type contextKey string

const userContextKey contextKey = "auth_grpc_user"

// RequireAuth calls the auth service via gRPC to verify authentication.
func RequireAuth(client *AuthClient, cookieName string) func(http.Handler) http.Handler {
	if cookieName == "" {
		cookieName = "status_session"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r, cookieName)
			if token == "" {
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "authentication required: missing session token",
				})
				return
			}

			resp, err := client.VerifyToken(r.Context(), token)
			if err != nil || !resp.Valid {
				msg := "invalid or expired authentication token"
				if resp != nil && resp.Error != "" {
					msg = resp.Error
				}
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": msg,
				})
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, resp.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin calls the auth service via gRPC to verify administrator privileges.
func RequireAdmin(client *AuthClient, cookieName string) func(http.Handler) http.Handler {
	if cookieName == "" {
		cookieName = "status_session"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r, cookieName)
			if token == "" {
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "authentication required: missing session token",
				})
				return
			}

			resp, err := client.VerifyToken(r.Context(), token)
			if err != nil || !resp.Valid {
				msg := "invalid or expired authentication token"
				if resp != nil && resp.Error != "" {
					msg = resp.Error
				}
				respondJSON(w, http.StatusUnauthorized, map[string]string{
					"error": msg,
				})
				return
			}

			if !resp.IsAdmin {
				respondJSON(w, http.StatusForbidden, map[string]string{
					"error": "forbidden: administrator privileges required",
				})
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, resp.User)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request, cookieName string) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	cookie, err := r.Cookie(cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// GetUser extracts the verified proto.User from context.
func GetUser(ctx context.Context) (*proto.User, bool) {
	user, ok := ctx.Value(userContextKey).(*proto.User)
	return user, ok
}

func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
