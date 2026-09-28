package server_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"

	"status-page/packages/auth"
	"status-page/packages/auth/client"
	"status-page/packages/auth/proto"
	"status-page/packages/auth/server"
)

func startTestGRPCServer(t *testing.T) (string, func()) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	cfg := auth.Config{
		ProviderType:        "mock",
		SessionSecret:       "test-secret-key-12345678",
		SessionTTL:          1 * time.Hour,
		PasswordAuthEnabled: true,
		AdminUser:           "superadmin",
		AdminPassword:       "superpass",
	}

	grpcServer := grpc.NewServer()
	srv := server.NewAuthServer(cfg)
	proto.RegisterAuthServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	addr := lis.Addr().String()
	cleanup := func() {
		grpcServer.Stop()
		_ = lis.Close()
	}

	return addr, cleanup
}

func TestGRPCAuth_EndToEnd(t *testing.T) {
	addr, cleanup := startTestGRPCServer(t)
	defer cleanup()

	c, err := client.NewAuthClient(addr)
	if err != nil {
		t.Fatalf("failed to create auth client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// 1. Test Login via gRPC
	loginRes, err := c.Login(ctx, "superadmin", "superpass")
	if err != nil || !loginRes.Success {
		t.Fatalf("expected login success, got err: %v, res: %+v", err, loginRes)
	}
	token := loginRes.Token
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	// 2. Test VerifyToken via gRPC
	verifyRes, err := c.VerifyToken(ctx, token)
	if err != nil || !verifyRes.Valid || !verifyRes.IsAdmin {
		t.Fatalf("expected valid admin verification, got err: %v, res: %+v", err, verifyRes)
	}

	// 3. Test Register via gRPC
	regRes, err := c.Register(ctx, "newuser", "pass123", "newuser@test.com", "New User")
	if err != nil || !regRes.Success {
		t.Fatalf("expected register success, got err: %v, res: %+v", err, regRes)
	}

	// 4. Test ForgotPassword & ResetPassword via gRPC
	forgotRes, err := c.ForgotPassword(ctx, "forgot@test.com")
	if err != nil || !forgotRes.Success || forgotRes.ResetToken == "" {
		t.Fatalf("expected forgot password success, got err: %v, res: %+v", err, forgotRes)
	}

	resetRes, err := c.ResetPassword(ctx, forgotRes.ResetToken, "new-secret-pass")
	if err != nil || !resetRes.Success || resetRes.Token == "" {
		t.Fatalf("expected reset password success, got err: %v, res: %+v", err, resetRes)
	}

	// 5. Test RequireAdmin Middleware calling auth via gRPC
	adminMiddleware := client.RequireAdmin(c, "status_session")
	protectedHandler := adminMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := client.GetUser(r.Context())
		if !ok || user.Email == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok:" + user.Email))
	}))

	req := httptest.NewRequest(http.MethodPost, "/secure-action", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	protectedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for secure action, got %d", rec.Code)
	}
}
