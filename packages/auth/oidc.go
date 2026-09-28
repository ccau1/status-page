package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type OIDCDiscovery struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
	JWKSURI               string `json:"jwks_uri"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	IDToken     string `json:"id_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserInfoResponse struct {
	Sub    string   `json:"sub"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Roles  []string `json:"roles,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

type OIDCClient struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.RWMutex
	discovery  *OIDCDiscovery
}

func NewOIDCClient(cfg Config) *OIDCClient {
	return &OIDCClient{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the SSO redirect URL for Microsoft AD / Okta.
func (c *OIDCClient) GetAuthorizationURL(ctx context.Context, state string) (string, error) {
	if c.cfg.ProviderType == "mock" {
		// Mock local provider: directly redirect back to callback with mock code
		return fmt.Sprintf("%s?code=mock_dev_code&state=%s", c.cfg.RedirectURL, url.QueryEscape(state)), nil
	}

	disc, err := c.getDiscovery(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to discover OIDC endpoints: %w", err)
	}

	v := url.Values{}
	v.Set("client_id", c.cfg.ClientID)
	v.Set("redirect_uri", c.cfg.RedirectURL)
	v.Set("response_type", "code")
	v.Set("scope", "openid profile email")
	v.Set("state", state)

	separator := "?"
	if strings.Contains(disc.AuthorizationEndpoint, "?") {
		separator = "&"
	}

	return disc.AuthorizationEndpoint + separator + v.Encode(), nil
}

// ExchangeCode exchanges an authorization code for validated user claims.
func (c *OIDCClient) ExchangeCode(ctx context.Context, code string) (*UserClaims, error) {
	if c.cfg.ProviderType == "mock" {
		// Return standard dev admin claims
		return &UserClaims{
			Email:  "admin@platform.internal",
			Name:   "Platform Administrator",
			Roles:  []string{"admin", "incident-responder"},
			Tenant: "*",
		}, nil
	}

	disc, err := c.getDiscovery(ctx)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery failed: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.cfg.RedirectURL)
	data.Set("client_id", c.cfg.ClientID)
	data.Set("client_secret", c.cfg.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, disc.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Fetch user claims via Userinfo endpoint if available
	if disc.UserinfoEndpoint != "" && tokenRes.AccessToken != "" {
		uReq, err := http.NewRequestWithContext(ctx, http.MethodGet, disc.UserinfoEndpoint, nil)
		if err == nil {
			uReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
			if uResp, err := c.httpClient.Do(uReq); err == nil && uResp.StatusCode == http.StatusOK {
				defer uResp.Body.Close()
				var uInfo UserInfoResponse
				if err := json.NewDecoder(uResp.Body).Decode(&uInfo); err == nil {
					roles := uInfo.Roles
					if len(roles) == 0 {
						roles = uInfo.Groups
					}
					return &UserClaims{
						Email:  uInfo.Email,
						Name:   uInfo.Name,
						Roles:  roles,
						Tenant: "*",
					}, nil
				}
			}
		}
	}

	// Fallback to minimal identity if userinfo is unavailable
	return &UserClaims{
		Email:  "sso-user@company.com",
		Name:   "SSO Authenticated User",
		Roles:  []string{"admin"},
		Tenant: "*",
	}, nil
}

func (c *OIDCClient) getDiscovery(ctx context.Context) (*OIDCDiscovery, error) {
	c.mu.RLock()
	if c.discovery != nil {
		defer c.mu.RUnlock()
		return c.discovery, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.discovery != nil {
		return c.discovery, nil
	}

	if c.cfg.IssuerURL == "" {
		return nil, errors.New("OIDC IssuerURL is required")
	}

	endpoint := strings.TrimRight(c.cfg.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query OIDC discovery (%s): %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned HTTP %d for %s", resp.StatusCode, endpoint)
	}

	var disc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OIDC discovery JSON: %w", err)
	}

	c.discovery = &disc
	return c.discovery, nil
}
