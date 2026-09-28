package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims represents the authenticated user's identity and permissions.
type UserClaims struct {
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Roles  []string `json:"roles"`
	Tenant string   `json:"tenant"`
	jwt.RegisteredClaims
}

// Config holds all authentication and SSO options.
type Config struct {
	// ProviderType: "oidc" for standard Microsoft AD / Okta, or "mock" for local dev
	ProviderType string
	ProviderName string // Display label (e.g. "Microsoft Entra ID" or "Okta")

	IssuerURL    string // e.g. https://login.microsoftonline.com/{tenant}/v2.0
	ClientID     string
	ClientSecret string
	RedirectURL  string

	// SessionSecret is used to sign and verify stateless JWT session cookies
	SessionSecret string
	SessionTTL    time.Duration
	CookieName    string

	// Authorization rules
	AdminRoles  []string
	AdminEmails []string

	// Frontend redirect target after successful SSO login
	SuccessRedirectURL string

	// Password Authentication
	PasswordAuthEnabled bool
	AdminUser           string
	AdminPassword       string
	LocalUsers          map[string]string // username -> password
}

// IsAdmin checks if the user possesses admin privileges according to config.
func (c Config) IsAdmin(user *UserClaims) bool {
	if user == nil {
		return false
	}

	// Check local configured admin user
	if c.AdminUser != "" && (user.Email == c.AdminUser || user.Name == c.AdminUser) {
		return true
	}

	// Check local user accounts
	if c.LocalUsers != nil {
		if _, ok := c.LocalUsers[user.Email]; ok {
			return true
		}
	}

	// In mock mode or if no restrictions set, grant admin
	if c.ProviderType == "mock" && len(c.AdminEmails) == 0 && len(c.AdminRoles) == 0 {
		return true
	}

	for _, email := range c.AdminEmails {
		if email == user.Email {
			return true
		}
	}

	for _, reqRole := range c.AdminRoles {
		for _, userRole := range user.Roles {
			if userRole == reqRole {
				return true
			}
		}
	}

	return false
}
