package proto

import (
	"context"

	"google.golang.org/grpc"
)

// Messages
type User struct {
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Roles  []string `json:"roles,omitempty"`
	Tenant string   `json:"tenant,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
	Message string `json:"message,omitempty"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type RegisterResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
	Message string `json:"message,omitempty"`
}

type LogoutRequest struct {
	Token string `json:"token"`
}

type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ForgotPasswordResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	ResetToken string `json:"reset_token,omitempty"`
}

type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

type ResetPasswordResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
	Message string `json:"message,omitempty"`
}

type VerifyTokenRequest struct {
	Token string `json:"token"`
}

type VerifyTokenResponse struct {
	Valid   bool   `json:"valid"`
	IsAdmin bool   `json:"is_admin"`
	User    *User  `json:"user,omitempty"`
	Error   string `json:"error,omitempty"`
}

type GetOIDCAuthURLRequest struct {
	State string `json:"state"`
}

type GetOIDCAuthURLResponse struct {
	URL string `json:"url"`
}

type HandleOIDCCallbackRequest struct {
	Code string `json:"code"`
}

type HandleOIDCCallbackResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	User    *User  `json:"user,omitempty"`
	Message string `json:"message,omitempty"`
}

// AuthServiceServer is the server API for AuthService service.
type AuthServiceServer interface {
	Login(context.Context, *LoginRequest) (*LoginResponse, error)
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	Logout(context.Context, *LogoutRequest) (*LogoutResponse, error)
	ForgotPassword(context.Context, *ForgotPasswordRequest) (*ForgotPasswordResponse, error)
	ResetPassword(context.Context, *ResetPasswordRequest) (*ResetPasswordResponse, error)
	VerifyToken(context.Context, *VerifyTokenRequest) (*VerifyTokenResponse, error)
	GetOIDCAuthURL(context.Context, *GetOIDCAuthURLRequest) (*GetOIDCAuthURLResponse, error)
	HandleOIDCCallback(context.Context, *HandleOIDCCallbackRequest) (*HandleOIDCCallbackResponse, error)
}

// AuthServiceClient is the client API for AuthService service.
type AuthServiceClient interface {
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error)
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutResponse, error)
	ForgotPassword(ctx context.Context, in *ForgotPasswordRequest, opts ...grpc.CallOption) (*ForgotPasswordResponse, error)
	ResetPassword(ctx context.Context, in *ResetPasswordRequest, opts ...grpc.CallOption) (*ResetPasswordResponse, error)
	VerifyToken(ctx context.Context, in *VerifyTokenRequest, opts ...grpc.CallOption) (*VerifyTokenResponse, error)
	GetOIDCAuthURL(ctx context.Context, in *GetOIDCAuthURLRequest, opts ...grpc.CallOption) (*GetOIDCAuthURLResponse, error)
	HandleOIDCCallback(ctx context.Context, in *HandleOIDCCallbackRequest, opts ...grpc.CallOption) (*HandleOIDCCallbackResponse, error)
}

type authServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAuthServiceClient(cc grpc.ClientConnInterface) AuthServiceClient {
	return &authServiceClient{cc}
}

func (c *authServiceClient) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginResponse, error) {
	out := new(LoginResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutResponse, error) {
	out := new(LogoutResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/Logout", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) ForgotPassword(ctx context.Context, in *ForgotPasswordRequest, opts ...grpc.CallOption) (*ForgotPasswordResponse, error) {
	out := new(ForgotPasswordResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/ForgotPassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) ResetPassword(ctx context.Context, in *ResetPasswordRequest, opts ...grpc.CallOption) (*ResetPasswordResponse, error) {
	out := new(ResetPasswordResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/ResetPassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) VerifyToken(ctx context.Context, in *VerifyTokenRequest, opts ...grpc.CallOption) (*VerifyTokenResponse, error) {
	out := new(VerifyTokenResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/VerifyToken", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) GetOIDCAuthURL(ctx context.Context, in *GetOIDCAuthURLRequest, opts ...grpc.CallOption) (*GetOIDCAuthURLResponse, error) {
	out := new(GetOIDCAuthURLResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/GetOIDCAuthURL", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *authServiceClient) HandleOIDCCallback(ctx context.Context, in *HandleOIDCCallbackRequest, opts ...grpc.CallOption) (*HandleOIDCCallbackResponse, error) {
	out := new(HandleOIDCCallbackResponse)
	err := c.cc.Invoke(ctx, "/auth.AuthService/HandleOIDCCallback", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Service registration
func RegisterAuthServiceServer(s grpc.ServiceRegistrar, srv AuthServiceServer) {
	s.RegisterService(&AuthService_ServiceDesc, srv)
}

var AuthService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "auth.AuthService",
	HandlerType: (*AuthServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Login",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(LoginRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).Login(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/Login",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).Login(ctx, req.(*LoginRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "Register",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(RegisterRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).Register(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/Register",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).Register(ctx, req.(*RegisterRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "Logout",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(LogoutRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).Logout(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/Logout",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).Logout(ctx, req.(*LogoutRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "ForgotPassword",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(ForgotPasswordRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).ForgotPassword(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/ForgotPassword",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).ForgotPassword(ctx, req.(*ForgotPasswordRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "ResetPassword",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(ResetPasswordRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).ResetPassword(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/ResetPassword",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).ResetPassword(ctx, req.(*ResetPasswordRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "VerifyToken",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(VerifyTokenRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).VerifyToken(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/VerifyToken",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).VerifyToken(ctx, req.(*VerifyTokenRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "GetOIDCAuthURL",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(GetOIDCAuthURLRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).GetOIDCAuthURL(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/GetOIDCAuthURL",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).GetOIDCAuthURL(ctx, req.(*GetOIDCAuthURLRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "HandleOIDCCallback",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(HandleOIDCCallbackRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(AuthServiceServer).HandleOIDCCallback(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/auth.AuthService/HandleOIDCCallback",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(AuthServiceServer).HandleOIDCCallback(ctx, req.(*HandleOIDCCallbackRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "auth.proto",
}
