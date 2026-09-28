package auth

import (
	"context"
	"net/http"
)

type contextKey string

const userContextKey contextKey = "auth_user_claims"

// RequireAuth is HTTP middleware ensuring a valid authenticated user.
func RequireAuth(service *AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := service.ExtractClaimsFromRequest(r)
			if err != nil {
				respondAuthError(w, http.StatusUnauthorized, "authentication required: "+err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin is HTTP middleware ensuring the caller has administrator privileges.
func RequireAdmin(service *AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := service.ExtractClaimsFromRequest(r)
			if err != nil {
				respondAuthError(w, http.StatusUnauthorized, "authentication required: "+err.Error())
				return
			}

			if !service.Config().IsAdmin(claims) {
				respondAuthError(w, http.StatusForbidden, "forbidden: administrator privileges required")
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser extracts the authenticated UserClaims from context if present.
func GetUser(ctx context.Context) (*UserClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*UserClaims)
	return claims, ok
}
