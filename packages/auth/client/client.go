package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"status-page/packages/auth/proto"
)

type AuthClient struct {
	conn   *grpc.ClientConn
	client proto.AuthServiceClient
}

// NewAuthClient connects to the standalone Auth gRPC service.
func NewAuthClient(addr string) (*AuthClient, error) {
	if addr == "" {
		addr = "localhost:50051"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype(proto.CodecName)),
		grpc.WithBlock(),
	)
	if err != nil {
		// Non-blocking fallback if dialing asynchronously
		conn, err = grpc.Dial(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.CallContentSubtype(proto.CodecName)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to auth gRPC service at %s: %w", addr, err)
		}
	}

	return &AuthClient{
		conn:   conn,
		client: proto.NewAuthServiceClient(conn),
	}, nil
}

func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *AuthClient) Login(ctx context.Context, username, password string) (*proto.LoginResponse, error) {
	return c.client.Login(ctx, &proto.LoginRequest{
		Username: username,
		Password: password,
	})
}

func (c *AuthClient) Register(ctx context.Context, username, password, email, name string) (*proto.RegisterResponse, error) {
	return c.client.Register(ctx, &proto.RegisterRequest{
		Username: username,
		Password: password,
		Email:    email,
		Name:     name,
	})
}

func (c *AuthClient) Logout(ctx context.Context, token string) (*proto.LogoutResponse, error) {
	return c.client.Logout(ctx, &proto.LogoutRequest{
		Token: token,
	})
}

func (c *AuthClient) ForgotPassword(ctx context.Context, email string) (*proto.ForgotPasswordResponse, error) {
	return c.client.ForgotPassword(ctx, &proto.ForgotPasswordRequest{
		Email: email,
	})
}

func (c *AuthClient) ResetPassword(ctx context.Context, resetToken, newPassword string) (*proto.ResetPasswordResponse, error) {
	return c.client.ResetPassword(ctx, &proto.ResetPasswordRequest{
		ResetToken:  resetToken,
		NewPassword: newPassword,
	})
}

func (c *AuthClient) VerifyToken(ctx context.Context, token string) (*proto.VerifyTokenResponse, error) {
	return c.client.VerifyToken(ctx, &proto.VerifyTokenRequest{
		Token: token,
	})
}

func (c *AuthClient) GetOIDCAuthURL(ctx context.Context, state string) (*proto.GetOIDCAuthURLResponse, error) {
	return c.client.GetOIDCAuthURL(ctx, &proto.GetOIDCAuthURLRequest{
		State: state,
	})
}

func (c *AuthClient) HandleOIDCCallback(ctx context.Context, code string) (*proto.HandleOIDCCallbackResponse, error) {
	return c.client.HandleOIDCCallback(ctx, &proto.HandleOIDCCallbackRequest{
		Code: code,
	})
}
