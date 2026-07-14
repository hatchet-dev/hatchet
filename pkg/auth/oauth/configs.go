package oauth

import (
	"fmt"

	"golang.org/x/oauth2"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	BaseURL      string

	// TenantID is only used by the Azure AD (Entra ID) provider. It is the
	// {tenant} segment of the authorize/token endpoints and may be a specific
	// tenant GUID/domain or one of the meta-tenants ("organizations", "common",
	// "consumers"). Ignored by the other providers.
	TenantID string
}

const (
	GoogleAuthURL  string = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL string = "https://oauth2.googleapis.com/token" // #nosec G101
	GithubAuthURL  string = "https://github.com/login/oauth/authorize"
	GithubTokenURL string = "https://github.com/login/oauth/access_token" // #nosec G101
	SlackAuthURL   string = "https://slack.com/oauth/v2/authorize"
	SlackTokenURL  string = "https://slack.com/api/oauth.v2.access" // #nosec G101

	// AzureADBaseURL is the base for Azure AD (Entra ID) OAuth endpoints. The
	// {tenant} segment is interpolated per-instance from the configured tenant.
	AzureADBaseURL string = "https://login.microsoftonline.com"

	// DefaultAzureTenant is used when no tenant is configured. It mirrors the
	// Google/GitHub providers' behavior: any directory may authenticate, and
	// access is instead gated by RestrictedEmailDomains. Set a specific tenant
	// ID/domain to lock sign-in to a single Azure AD directory.
	DefaultAzureTenant string = "organizations"
)

func NewGoogleClient(cfg *Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  GoogleAuthURL,
			TokenURL: GoogleTokenURL,
		},
		RedirectURL: cfg.BaseURL + "/api/v1/users/google/callback",
		Scopes:      cfg.Scopes,
	}
}

func NewGithubClient(cfg *Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  GithubAuthURL,
			TokenURL: GithubTokenURL,
		},
		RedirectURL: cfg.BaseURL + "/api/v1/users/github/callback",
		Scopes:      cfg.Scopes,
	}
}

func NewSlackClient(cfg *Config) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  SlackAuthURL,
			TokenURL: SlackTokenURL,
		},
		RedirectURL: cfg.BaseURL + "/api/v1/users/slack/callback",
		Scopes:      cfg.Scopes,
	}
}

func NewAzureClient(cfg *Config) *oauth2.Config {
	tenant := cfg.TenantID
	if tenant == "" {
		tenant = DefaultAzureTenant
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/%s/oauth2/v2.0/authorize", AzureADBaseURL, tenant),
			TokenURL: fmt.Sprintf("%s/%s/oauth2/v2.0/token", AzureADBaseURL, tenant),
		},
		RedirectURL: cfg.BaseURL + "/api/v1/users/azure/callback",
		Scopes:      cfg.Scopes,
	}
}
