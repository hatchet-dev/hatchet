package oauth

import (
	"golang.org/x/oauth2"
)

type Config struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	BaseURL      string
}

const (
	GoogleAuthURL  string = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL string = "https://oauth2.googleapis.com/token" // #nosec G101
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
