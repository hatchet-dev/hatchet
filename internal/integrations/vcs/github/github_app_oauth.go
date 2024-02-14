package github

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/internal/auth/oauth"

	githubsdk "github.com/google/go-github/v57/github"
)

const (
	GithubAuthURL  string = "https://github.com/login/oauth/authorize"
	GithubTokenURL string = "https://github.com/login/oauth/access_token" // #nosec G101
)

type GithubAppConf struct {
	oauth2.Config

	appName       string
	webhookSecret string
	secret        []byte
	appID         int64
}

func NewGithubAppConf(cfg *oauth.Config, appName, appSecretPath, appWebhookSecret, appID string) (*GithubAppConf, error) {
	intAppID, err := strconv.ParseInt(appID, 10, 64)

	if err != nil {
		return nil, err
	}

	appSecret, err := ioutil.ReadFile(appSecretPath)

	if err != nil {
		return nil, fmt.Errorf("could not read github app secret: %s", err)
	}

	return &GithubAppConf{
		appName:       appName,
		webhookSecret: appWebhookSecret,
		secret:        appSecret,
		appID:         intAppID,
		Config: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  GithubAuthURL,
				TokenURL: GithubTokenURL,
			},
			RedirectURL: cfg.BaseURL + "/api/v1/users/github/callback",
			Scopes:      cfg.Scopes,
		},
	}, nil
}

func (g *GithubAppConf) GetGithubClient(installationID int64) (*githubsdk.Client, error) {
	itr, err := ghinstallation.New(
		http.DefaultTransport,
		g.appID,
		installationID,
		g.secret,
	)

	if err != nil {
		return nil, err
	}

	return githubsdk.NewClient(&http.Client{Transport: itr}), nil
}

func (g *GithubAppConf) GetWebhookSecret() string {
	return g.webhookSecret
}

func (g *GithubAppConf) GetAppName() string {
	return g.appName
}
