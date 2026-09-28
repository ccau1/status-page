package auth_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"status-page/packages/auth"
)

func TestSignAndVerifyToken(t *testing.T) {
	secret := "test-super-secret-key-123456789012"
	claims := auth.UserClaims{
		Email:  "admin@acme.com",
		Name:   "Admin User",
		Roles:  []string{"admin"},
		Tenant: "acme",
	}

	tokenStr, err := auth.SignToken(claims, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	verified, err := auth.VerifyToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("failed to verify token: %v", err)
	}

	if verified.Email != "admin@acme.com" {
		t.Errorf("expected admin@acme.com, got %s", verified.Email)
	}
	if verified.Tenant != "acme" {
		t.Errorf("expected tenant acme, got %s", verified.Tenant)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	secret := "test-super-secret-key-123456789012"
	claims := auth.UserClaims{
		Email: "admin@acme.com",
	}

	// Token expired 1 hour ago
	tokenStr, err := auth.SignToken(claims, secret, -1*time.Hour)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = auth.VerifyToken(tokenStr, secret)
	if err == nil {
		t.Fatalf("expected error on expired token, got nil")
	}
}

func TestAuthService_MockFlow(t *testing.T) {
	cfg := auth.Config{
		ProviderType:  "mock",
		SessionSecret: "test-secret-key-12345678",
		AdminEmails:   []string{"admin@platform.internal"},
	}
	svc := auth.NewService(cfg)

	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 1. GET /auth/config
	req := httptest.NewRequest(http.MethodGet, "/auth/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// 2. GET /auth/callback?code=mock_dev_code
	req = httptest.NewRequest(http.MethodGet, "/auth/callback?code=mock_dev_code", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect 302, got %d", rec.Code)
	}

	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "status_session" || cookie.Value == "" {
		t.Fatalf("session cookie not set")
	}

	// 3. GET /auth/me with cookie
	reqMe := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqMe.AddCookie(cookie)
	recMe := httptest.NewRecorder()
	mux.ServeHTTP(recMe, reqMe)
	if recMe.Code != http.StatusOK {
		t.Errorf("expected 200 on /auth/me, got %d", recMe.Code)
	}

	// 4. Protected Admin route with middleware
	adminHandler := auth.RequireAdmin(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("admin-ok"))
	}))

	reqAdmin := httptest.NewRequest(http.MethodPost, "/admin-only", nil)
	reqAdmin.AddCookie(cookie)
	recAdmin := httptest.NewRecorder()
	adminHandler.ServeHTTP(recAdmin, reqAdmin)
	if recAdmin.Code != http.StatusOK {
		t.Errorf("expected 200 for authenticated admin, got %d", recAdmin.Code)
	}
}

func TestAuthService_PasswordLogin(t *testing.T) {
	cfg := auth.Config{
		PasswordAuthEnabled: true,
		AdminUser:           "root",
		AdminPassword:       "super-secret-pass",
		SessionSecret:       "test-secret-key-12345678",
	}
	svc := auth.NewService(cfg)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 1. Invalid credentials
	badBody := `{"username":"root","password":"wrong-password"}`
	reqBad := httptest.NewRequest(http.MethodPost, "/auth/login/password", strings.NewReader(badBody))
	recBad := httptest.NewRecorder()
	mux.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for bad password, got %d", recBad.Code)
	}

	// 2. Correct credentials
	goodBody := `{"username":"root","password":"super-secret-pass"}`
	reqGood := httptest.NewRequest(http.MethodPost, "/auth/login/password", strings.NewReader(goodBody))
	recGood := httptest.NewRecorder()
	mux.ServeHTTP(recGood, reqGood)
	if recGood.Code != http.StatusOK {
		t.Errorf("expected 200 for correct credentials, got %d", recGood.Code)
	}

	cookies := recGood.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "status_session" {
		t.Fatalf("expected session cookie to be set")
	}

	// 3. Test /auth/me with the password session cookie
	reqMe := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	reqMe.AddCookie(cookies[0])
	recMe := httptest.NewRecorder()
	mux.ServeHTTP(recMe, reqMe)
	if recMe.Code != http.StatusOK {
		t.Errorf("expected 200 on /auth/me, got %d", recMe.Code)
	}
}
