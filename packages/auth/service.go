package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

type AuthService struct {
	cfg        Config
	oidcClient *OIDCClient
}

func NewService(cfg Config) *AuthService {
	if cfg.ProviderType == "" {
		cfg.ProviderType = "mock"
	}
	if cfg.SessionSecret == "" {
		cfg.SessionSecret = "dev-secret-key-please-change-in-production-12345678"
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 24 * time.Hour
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "status_session"
	}
	if cfg.RedirectURL == "" {
		cfg.RedirectURL = "/auth/callback"
	}
	if cfg.SuccessRedirectURL == "" {
		cfg.SuccessRedirectURL = "/admin"
	}

	return &AuthService{
		cfg:        cfg,
		oidcClient: NewOIDCClient(cfg),
	}
}

// LoadConfigFromEnv builds an Auth Config from system environment variables.
func LoadConfigFromEnv() Config {
	providerType := os.Getenv("SSO_PROVIDER")
	if providerType == "" {
		providerType = "mock"
	}

	providerName := os.Getenv("SSO_PROVIDER_NAME")
	if providerName == "" {
		if providerType == "mock" {
			providerName = "Local Dev SSO"
		} else {
			providerName = "Enterprise SSO (Microsoft AD / Okta)"
		}
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-key-please-change-in-production-12345678"
	}

	var adminEmails []string
	if emails := os.Getenv("ADMIN_EMAILS"); emails != "" {
		for _, e := range strings.Split(emails, ",") {
			adminEmails = append(adminEmails, strings.TrimSpace(e))
		}
	}

	var adminRoles []string
	if roles := os.Getenv("ADMIN_ROLES"); roles != "" {
		for _, r := range strings.Split(roles, ",") {
			adminRoles = append(adminRoles, strings.TrimSpace(r))
		}
	}

	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin" // Default for quick setup/dev
	}

	passwordAuthEnabled := true
	if val := os.Getenv("PASSWORD_AUTH_ENABLED"); val == "false" || val == "0" {
		passwordAuthEnabled = false
	}

	localUsers := make(map[string]string)
	if rawUsers := os.Getenv("LOCAL_USERS"); rawUsers != "" {
		for _, pair := range strings.Split(rawUsers, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
			if len(parts) == 2 {
				localUsers[parts[0]] = parts[1]
			}
		}
	}

	return Config{
		ProviderType:        providerType,
		ProviderName:        providerName,
		IssuerURL:           os.Getenv("OIDC_ISSUER_URL"),
		ClientID:            os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret:        os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:         os.Getenv("OIDC_REDIRECT_URL"),
		SessionSecret:       sessionSecret,
		SessionTTL:          24 * time.Hour,
		CookieName:          "status_session",
		AdminEmails:         adminEmails,
		AdminRoles:          adminRoles,
		SuccessRedirectURL:  os.Getenv("SSO_SUCCESS_REDIRECT_URL"),
		PasswordAuthEnabled: passwordAuthEnabled,
		AdminUser:           adminUser,
		AdminPassword:       adminPassword,
		LocalUsers:          localUsers,
	}
}

// Config returns the current public auth configuration.
func (s *AuthService) Config() Config {
	return s.cfg
}

// GetLoginURL generates the SSO authorization redirect URL.
func (s *AuthService) GetLoginURL(r *http.Request, state string) (string, error) {
	return s.oidcClient.GetAuthorizationURL(r.Context(), state)
}

// HandleCallback exchanges the code for user claims and generates a session JWT.
func (s *AuthService) HandleCallback(r *http.Request, code string) (*UserClaims, string, error) {
	claims, err := s.oidcClient.ExchangeCode(r.Context(), code)
	if err != nil {
		return nil, "", err
	}

	token, err := SignToken(*claims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		return nil, "", err
	}

	return claims, token, nil
}

// VerifyToken validates an incoming session JWT.
func (s *AuthService) VerifyToken(tokenString string) (*UserClaims, error) {
	return VerifyToken(tokenString, s.cfg.SessionSecret)
}

// SetSessionCookie writes the session JWT into a secure HttpOnly cookie.
func (s *AuthService) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(s.cfg.SessionTTL),
		HttpOnly: true,
		Secure:   false, // Set to true if running over HTTPS in production
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie invalidates the user's session cookie.
func (s *AuthService) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.cfg.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ExtractClaimsFromRequest reads the token from Authorization header or session cookie.
func (s *AuthService) ExtractClaimsFromRequest(r *http.Request) (*UserClaims, error) {
	// 1. Try Bearer header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return s.VerifyToken(token)
	}

	// 2. Try cookie
	cookie, err := r.Cookie(s.cfg.CookieName)
	if err == nil && cookie.Value != "" {
		return s.VerifyToken(cookie.Value)
	}

	return nil, ErrMissingToken
}

// RegisterRoutes registers standard auth endpoints on any http.ServeMux router.
func (s *AuthService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /auth/config", s.handleConfig)
	mux.HandleFunc("GET /auth/login", s.handleLogin)
	mux.HandleFunc("POST /auth/login/password", s.handlePasswordLogin)
	mux.HandleFunc("GET /auth/callback", s.handleCallback)
	mux.HandleFunc("GET /auth/me", s.handleMe)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
}

func (s *AuthService) handleConfig(w http.ResponseWriter, r *http.Request) {
	respondAuthJSON(w, http.StatusOK, map[string]interface{}{
		"provider_type":          s.cfg.ProviderType,
		"provider_name":          s.cfg.ProviderName,
		"password_auth_enabled": s.cfg.PasswordAuthEnabled,
	})
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *AuthService) handlePasswordLogin(w http.ResponseWriter, r *http.Request) {
	var req passwordLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondAuthError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	claims, err := s.AuthenticatePassword(req.Username, req.Password)
	if err != nil {
		respondAuthError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := SignToken(*claims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		respondAuthError(w, http.StatusInternalServerError, "failed to generate session: "+err.Error())
		return
	}

	s.SetSessionCookie(w, token)

	respondAuthJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "authenticated",
		"is_admin": true,
		"user": map[string]interface{}{
			"email": claims.Email,
			"name":  claims.Name,
			"roles": claims.Roles,
		},
	})
}

// AuthenticatePassword verifies username and password using constant-time comparison.
func (s *AuthService) AuthenticatePassword(username, password string) (*UserClaims, error) {
	if !s.cfg.PasswordAuthEnabled {
		return nil, errors.New("password authentication is disabled")
	}
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	valid := false

	// 1. Check primary admin credentials
	if s.cfg.AdminUser != "" && s.cfg.AdminPassword != "" {
		userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(s.cfg.AdminUser)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(s.cfg.AdminPassword)) == 1
		if userMatch && passMatch {
			valid = true
		}
	}

	// 2. Check local users map
	if !valid && s.cfg.LocalUsers != nil {
		if expectedPass, exists := s.cfg.LocalUsers[username]; exists {
			if subtle.ConstantTimeCompare([]byte(password), []byte(expectedPass)) == 1 {
				valid = true
			}
		}
	}

	if !valid {
		return nil, errors.New("invalid username or password")
	}

	return &UserClaims{
		Email:  username,
		Name:   username,
		Roles:  []string{"admin"},
		Tenant: "*",
	}, nil
}

func (s *AuthService) handleLogin(w http.ResponseWriter, r *http.Request) {
	authURL, err := s.GetLoginURL(r, "state-token")
	if err != nil {
		respondAuthError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (s *AuthService) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondAuthError(w, http.StatusBadRequest, "missing authorization code")
		return
	}

	_, token, err := s.HandleCallback(r, code)
	if err != nil {
		respondAuthError(w, http.StatusUnauthorized, "SSO verification failed: "+err.Error())
		return
	}

	s.SetSessionCookie(w, token)

	targetURL := s.cfg.SuccessRedirectURL
	if targetURL == "" {
		targetURL = "/admin"
	}
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (s *AuthService) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, err := s.ExtractClaimsFromRequest(r)
	if err != nil {
		respondAuthJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	isAdmin := s.cfg.IsAdmin(claims)

	respondAuthJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"is_admin":      isAdmin,
		"user": map[string]interface{}{
			"email":  claims.Email,
			"name":   claims.Name,
			"roles":  claims.Roles,
			"tenant": claims.Tenant,
		},
	})
}

func (s *AuthService) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.ClearSessionCookie(w)
	respondAuthJSON(w, http.StatusOK, map[string]string{
		"status": "logged_out",
	})
}

func respondAuthJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondAuthError(w http.ResponseWriter, code int, message string) {
	respondAuthJSON(w, code, map[string]string{
		"error": message,
	})
}
