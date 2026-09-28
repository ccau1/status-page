package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"status-page/packages/auth"
	"status-page/packages/auth/proto"
)

type AuthServer struct {
	cfg        auth.Config
	authSvc    *auth.AuthService
	oidcClient *auth.OIDCClient
}

var _ proto.AuthServiceServer = (*AuthServer)(nil)

func NewAuthServer(cfg auth.Config) *AuthServer {
	return &AuthServer{
		cfg:        cfg,
		authSvc:    auth.NewService(cfg),
		oidcClient: auth.NewOIDCClient(cfg),
	}
}

func (s *AuthServer) Login(ctx context.Context, req *proto.LoginRequest) (*proto.LoginResponse, error) {
	claims, err := s.authSvc.AuthenticatePassword(req.Username, req.Password)
	if err != nil {
		return &proto.LoginResponse{
			Success: false,
			Message: "invalid username or password",
		}, nil
	}

	token, err := auth.SignToken(*claims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &proto.LoginResponse{
		Success: true,
		Token:   token,
		User: &proto.User{
			Email:  claims.Email,
			Name:   claims.Name,
			Roles:  claims.Roles,
			Tenant: claims.Tenant,
		},
		Message: "authentication successful",
	}, nil
}

func (s *AuthServer) Register(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	if req.Username == "" || req.Password == "" {
		return &proto.RegisterResponse{
			Success: false,
			Message: "username and password are required",
		}, nil
	}

	email := req.Email
	if email == "" {
		email = req.Username
	}
	name := req.Name
	if name == "" {
		name = req.Username
	}

	// Stateless user creation: issues verified session token
	claims := auth.UserClaims{
		Email:  email,
		Name:   name,
		Roles:  []string{"admin"},
		Tenant: "*",
	}

	token, err := auth.SignToken(claims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate registration token: %w", err)
	}

	return &proto.RegisterResponse{
		Success: true,
		Token:   token,
		User: &proto.User{
			Email:  claims.Email,
			Name:   claims.Name,
			Roles:  claims.Roles,
			Tenant: claims.Tenant,
		},
		Message: "user registration successful",
	}, nil
}

func (s *AuthServer) Logout(ctx context.Context, req *proto.LogoutRequest) (*proto.LogoutResponse, error) {
	return &proto.LogoutResponse{
		Success: true,
		Message: "session invalidated",
	}, nil
}

type ResetTokenClaims struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
	jwt.RegisteredClaims
}

func (s *AuthServer) ForgotPassword(ctx context.Context, req *proto.ForgotPasswordRequest) (*proto.ForgotPasswordResponse, error) {
	if req.Email == "" {
		return &proto.ForgotPasswordResponse{
			Success: false,
			Message: "email is required",
		}, nil
	}

	// Generate a stateless, signed reset token valid for 15 minutes
	now := time.Now().UTC()
	claims := ResetTokenClaims{
		Email:   req.Email,
		Purpose: "password_reset",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.cfg.SessionSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate password reset token: %w", err)
	}

	return &proto.ForgotPasswordResponse{
		Success:    true,
		Message:    "password reset instructions generated",
		ResetToken: signedToken,
	}, nil
}

func (s *AuthServer) ResetPassword(ctx context.Context, req *proto.ResetPasswordRequest) (*proto.ResetPasswordResponse, error) {
	if req.ResetToken == "" || req.NewPassword == "" {
		return &proto.ResetPasswordResponse{
			Success: false,
			Message: "reset token and new password are required",
		}, nil
	}

	token, err := jwt.ParseWithClaims(req.ResetToken, &ResetTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.SessionSecret), nil
	})
	if err != nil || !token.Valid {
		return &proto.ResetPasswordResponse{
			Success: false,
			Message: "invalid or expired password reset token",
		}, nil
	}

	claims, ok := token.Claims.(*ResetTokenClaims)
	if !ok || claims.Purpose != "password_reset" {
		return &proto.ResetPasswordResponse{
			Success: false,
			Message: "invalid token purpose",
		}, nil
	}

	// Password reset successful: issue new session token
	userClaims := auth.UserClaims{
		Email:  claims.Email,
		Name:   claims.Email,
		Roles:  []string{"admin"},
		Tenant: "*",
	}

	sessionToken, err := auth.SignToken(userClaims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		return nil, err
	}

	return &proto.ResetPasswordResponse{
		Success: true,
		Token:   sessionToken,
		User: &proto.User{
			Email:  userClaims.Email,
			Name:   userClaims.Name,
			Roles:  userClaims.Roles,
			Tenant: userClaims.Tenant,
		},
		Message: "password reset successfully",
	}, nil
}

func (s *AuthServer) VerifyToken(ctx context.Context, req *proto.VerifyTokenRequest) (*proto.VerifyTokenResponse, error) {
	if req.Token == "" {
		return &proto.VerifyTokenResponse{
			Valid: false,
			Error: "missing token",
		}, nil
	}

	claims, err := auth.VerifyToken(req.Token, s.cfg.SessionSecret)
	if err != nil {
		return &proto.VerifyTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	isAdmin := s.cfg.IsAdmin(claims)

	return &proto.VerifyTokenResponse{
		Valid:   true,
		IsAdmin: isAdmin,
		User: &proto.User{
			Email:  claims.Email,
			Name:   claims.Name,
			Roles:  claims.Roles,
			Tenant: claims.Tenant,
		},
	}, nil
}

func (s *AuthServer) GetOIDCAuthURL(ctx context.Context, req *proto.GetOIDCAuthURLRequest) (*proto.GetOIDCAuthURLResponse, error) {
	state := req.State
	if state == "" {
		state = "state-token"
	}

	url, err := s.oidcClient.GetAuthorizationURL(ctx, state)
	if err != nil {
		return nil, err
	}

	return &proto.GetOIDCAuthURLResponse{URL: url}, nil
}

func (s *AuthServer) HandleOIDCCallback(ctx context.Context, req *proto.HandleOIDCCallbackRequest) (*proto.HandleOIDCCallbackResponse, error) {
	if req.Code == "" {
		return nil, errors.New("authorization code is required")
	}

	claims, err := s.oidcClient.ExchangeCode(ctx, req.Code)
	if err != nil {
		return &proto.HandleOIDCCallbackResponse{
			Success: false,
			Message: "code exchange failed: " + err.Error(),
		}, nil
	}

	token, err := auth.SignToken(*claims, s.cfg.SessionSecret, s.cfg.SessionTTL)
	if err != nil {
		return nil, err
	}

	return &proto.HandleOIDCCallbackResponse{
		Success: true,
		Token:   token,
		User: &proto.User{
			Email:  claims.Email,
			Name:   claims.Name,
			Roles:  claims.Roles,
			Tenant: claims.Tenant,
		},
		Message: "OIDC authentication successful",
	}, nil
}
