package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"status-page/packages/auth/client"
)

type AuthGateway struct {
	client     *client.AuthClient
	cookieName string
}

func NewAuthGateway(authClient *client.AuthClient) *AuthGateway {
	cookieName := os.Getenv("COOKIE_NAME")
	if cookieName == "" {
		cookieName = "status_session"
	}
	return &AuthGateway{
		client:     authClient,
		cookieName: cookieName,
	}
}

func (g *AuthGateway) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/config", g.Config)
	mux.HandleFunc("POST /auth/login", g.Login)
	mux.HandleFunc("POST /auth/login/password", g.Login) // backward-compatible alias
	mux.HandleFunc("POST /auth/register", g.Register)
	mux.HandleFunc("POST /auth/logout", g.Logout)
	mux.HandleFunc("POST /auth/forgot-password", g.ForgotPassword)
	mux.HandleFunc("POST /auth/reset-password", g.ResetPassword)
	mux.HandleFunc("GET /auth/me", g.Me)
	mux.HandleFunc("GET /auth/login/sso", g.SSOLogin)
	mux.HandleFunc("GET /auth/login", g.SSOLogin) // backward-compatible alias
	mux.HandleFunc("GET /auth/callback", g.SSOCallback)
}

func (g *AuthGateway) Config(w http.ResponseWriter, r *http.Request) {
	providerType := os.Getenv("SSO_PROVIDER")
	if providerType == "" {
		providerType = "mock"
	}
	providerName := os.Getenv("SSO_PROVIDER_NAME")
	if providerName == "" {
		if providerType == "mock" {
			providerName = "Local Dev SSO"
		} else {
			providerName = "Enterprise SSO"
		}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"provider_type":          providerType,
		"provider_name":          providerName,
		"password_auth_enabled": true,
	})
}

func (g *AuthGateway) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call Auth Service via gRPC
	res, err := g.client.Login(r.Context(), body.Username, body.Password)
	if err != nil {
		respondError(w, http.StatusBadGateway, "auth service communication failed: "+err.Error())
		return
	}
	if !res.Success {
		respondError(w, http.StatusUnauthorized, res.Message)
		return
	}

	g.setCookie(w, res.Token)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "authenticated",
		"is_admin": true,
		"user":     res.User,
		"token":    res.Token,
	})
}

func (g *AuthGateway) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call Auth Service via gRPC
	res, err := g.client.Register(r.Context(), body.Username, body.Password, body.Email, body.Name)
	if err != nil {
		respondError(w, http.StatusBadGateway, "auth service communication failed: "+err.Error())
		return
	}
	if !res.Success {
		respondError(w, http.StatusBadRequest, res.Message)
		return
	}

	g.setCookie(w, res.Token)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "registered",
		"is_admin": true,
		"user":     res.User,
		"token":    res.Token,
	})
}

func (g *AuthGateway) Logout(w http.ResponseWriter, r *http.Request) {
	token := g.extractToken(r)

	// Call Auth Service via gRPC
	if token != "" {
		_, _ = g.client.Logout(r.Context(), token)
	}

	g.clearCookie(w)

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "logged_out",
	})
}

func (g *AuthGateway) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call Auth Service via gRPC
	res, err := g.client.ForgotPassword(r.Context(), body.Email)
	if err != nil {
		respondError(w, http.StatusBadGateway, "auth service communication failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, res)
}

func (g *AuthGateway) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ResetToken  string `json:"reset_token"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Call Auth Service via gRPC
	res, err := g.client.ResetPassword(r.Context(), body.ResetToken, body.NewPassword)
	if err != nil {
		respondError(w, http.StatusBadGateway, "auth service communication failed: "+err.Error())
		return
	}
	if !res.Success {
		respondError(w, http.StatusBadRequest, res.Message)
		return
	}

	g.setCookie(w, res.Token)

	respondJSON(w, http.StatusOK, res)
}

func (g *AuthGateway) Me(w http.ResponseWriter, r *http.Request) {
	token := g.extractToken(r)
	if token == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Call Auth Service via gRPC to verify token
	res, err := g.client.VerifyToken(r.Context(), token)
	if err != nil || !res.Valid {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"is_admin":      res.IsAdmin,
		"user":          res.User,
	})
}

func (g *AuthGateway) SSOLogin(w http.ResponseWriter, r *http.Request) {
	res, err := g.client.GetOIDCAuthURL(r.Context(), "state-sso")
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, res.URL, http.StatusFound)
}

func (g *AuthGateway) SSOCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "missing code")
		return
	}

	res, err := g.client.HandleOIDCCallback(r.Context(), code)
	if err != nil || !res.Success {
		respondError(w, http.StatusUnauthorized, "OIDC authentication failed")
		return
	}

	g.setCookie(w, res.Token)

	target := os.Getenv("SSO_SUCCESS_REDIRECT_URL")
	if target == "" {
		target = "/admin"
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (g *AuthGateway) setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     g.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (g *AuthGateway) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     g.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (g *AuthGateway) extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	cookie, err := r.Cookie(g.cookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}
