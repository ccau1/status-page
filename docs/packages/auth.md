# `packages/auth` — Reusable Enterprise SSO & Stateless Auth

`packages/auth` is an independent, reusable Go module providing OpenID Connect (OIDC) single sign-on (Microsoft Entra ID / Azure AD, Okta, and generic OIDC providers) and stateless JWT verification.

---

## 🎯 Design Goals

1. **Reusability across Services:** Any Go microservice or API can import `status-page/packages/auth` to protect routes, extract claims, or verify JWT session cookies without re-implementing auth logic.
2. **Stateless App Tier:** Generates and validates HMAC-SHA256 signed JWTs with zero database lookups, making it ideal for horizontally scaled multi-node EC2 instances behind an Application Load Balancer.
3. **Standardized Provider Support:** Connects to Microsoft Entra ID (Azure AD), Okta, Google Workspace, or any OIDC compliant provider via automatic discovery (`/.well-known/openid-configuration`).
4. **Offline Mock Mode:** Built-in `mock` provider for zero-friction local development and CI testing without requiring Azure or Okta credentials.
5. **Username / Password Authentication:** Supports local administrator username & password verification (`POST /auth/login/password`) using constant-time comparison (`crypto/subtle.ConstantTimeCompare`) to prevent timing attacks.

---

## 🔑 Authentication Methods Supported

### 1. Enterprise SSO (OpenID Connect)
* Redirects user via `GET /auth/login` to Microsoft Entra ID / Okta.
* Exchanges authorization code via `GET /auth/callback` and sets stateless `status_session` cookie.

### 2. Regular Username / Password Login
* Submits credentials via `POST /auth/login/password` with JSON payload:
  ```json
  { "username": "admin", "password": "your-password" }
  ```
* Validates credentials against `ADMIN_USER` / `ADMIN_PASSWORD` or `LOCAL_USERS`.
* Generates the identical stateless JWT session cookie so all downstream middleware operates uniformly.

## 📦 Package Architecture

```text
packages/auth/
├── proto/
│   ├── auth.proto           # Protocol Buffer schema definition
│   ├── codec.go             # High-performance gRPC JSON codec
│   └── messages.go          # Message types, client & server interfaces
├── server/
│   ├── server.go            # Stateless gRPC server implementation
│   └── server_test.go       # gRPC end-to-end tests
├── client/
│   ├── client.go            # Reusable gRPC client for any Go service
│   └── middleware.go        # HTTP middleware calling gRPC to verify tokens
├── cmd/auth-server/
│   └── main.go              # Standalone gRPC microservice entrypoint
├── Dockerfile               # Multi-stage Docker container for Auth service
├── types.go                 # Shared domain types (UserClaims, Config)
├── jwt.go                   # Stateless JWT helpers
└── oidc.go                  # OIDC discovery & flow
```

---

## 🚀 gRPC Service RPC Methods

The `AuthService` exposes the following RPCs:

| RPC Method | Input | Output | Description |
|---|---|---|---|
| `Login` | `LoginRequest` | `LoginResponse` | Authenticates credentials, returns signed session token |
| `Register` | `RegisterRequest` | `RegisterResponse` | Provisions operator account statelessly, returns session token |
| `Logout` | `LogoutRequest` | `LogoutResponse` | Invalidation confirmation |
| `ForgotPassword` | `ForgotPasswordRequest` | `ForgotPasswordResponse` | Generates signed stateless 15-minute reset token |
| `ResetPassword` | `ResetPasswordRequest` | `ResetPasswordResponse` | Verifies reset token, sets new password, returns new session |
| `VerifyToken` | `VerifyTokenRequest` | `VerifyTokenResponse` | Cryptographically validates token and returns user claims & `is_admin` |
| `GetOIDCAuthURL` | `GetOIDCAuthURLRequest` | `GetOIDCAuthURLResponse` | Returns enterprise SSO redirect URL (Microsoft AD / Okta) |
| `HandleOIDCCallback` | `HandleOIDCCallbackRequest` | `HandleOIDCCallbackResponse` | Exchanges authorization code for claims and signed session token |

---

## 🔒 Reusable gRPC-Backed HTTP Middleware

Any Go microservice can connect to `packages/auth` and protect endpoints with one line of code:

```go
import "status-page/packages/auth/client"

authClient, err := client.NewAuthClient("auth:50051")
defer authClient.Close()

// Require valid authentication (calls auth service via gRPC)
mux.Handle("/secure-endpoint", client.RequireAuth(authClient, "status_session")(myHandler))

// Require administrator privileges (calls auth service via gRPC)
mux.Handle("/admin-endpoint", client.RequireAdmin(authClient, "status_session")(myAdminHandler))
```

---

## 🚀 How to Reuse in Any Go Application

### 1. Import the package
```go
import "status-page/packages/auth"
```

### 2. Initialize the Service
```go
authCfg := auth.LoadConfigFromEnv()
authService := auth.NewService(authCfg)

// Mount standard endpoints: /auth/login, /auth/callback, /auth/me, /auth/logout
authService.RegisterRoutes(mux)
```

### 3. Protect Administrative Endpoints
```go
// Enforce admin privileges using middleware
mux.Handle("POST /api/incidents", auth.RequireAdmin(authService)(http.HandlerFunc(myHandler)))

// Access user identity inside any handler
func myHandler(w http.ResponseWriter, r *http.Request) {
    user, ok := auth.GetUser(r.Context())
    if ok {
        log.Printf("Action performed by: %s (%s)", user.Name, user.Email)
    }
}
```

---

## ⚙️ Configuration Reference

| Environment Variable | Default | Description |
|---|---|---|
| `SSO_PROVIDER` | `mock` | `oidc` (production) or `mock` (local development) |
| `SSO_PROVIDER_NAME` | `Local Dev SSO` | Display label on the admin login screen (e.g. `Microsoft Entra ID`) |
| `OIDC_ISSUER_URL` | *(empty)* | OIDC discovery base URL (e.g. `https://login.microsoftonline.com/{tenant-id}/v2.0`) |
| `OIDC_CLIENT_ID` | *(empty)* | Enterprise Application Client ID |
| `OIDC_CLIENT_SECRET` | *(empty)* | Enterprise Application Client Secret |
| `OIDC_REDIRECT_URL` | `http://localhost:8085/auth/callback` | Authorized redirect URI |
| `SESSION_SECRET` | *(random default)* | HMAC-SHA256 secret key for stateless JWT signing |
| `ADMIN_EMAILS` | *(empty)* | Comma-separated list of authorized administrator emails |
| `ADMIN_ROLES` | *(empty)* | Comma-separated list of required AD security groups/roles |
