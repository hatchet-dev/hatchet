package githubapp

import (
	"context"
	"fmt"

	"github.com/google/go-github/v57/github"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (g *GithubAppService) UserUpdateGithubOauthCallback(ctx echo.Context, _ gen.UserUpdateGithubOauthCallbackRequestObject) (gen.UserUpdateGithubOauthCallbackResponseObject, error) {
	user := ctx.Get("user").(*db.UserModel)

	ghApp, err := GetGithubAppConfig(g.config)

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Github app is misconfigured on this Hatchet instance.")
	}

	isValid, isOAuthTriggered, err := authn.NewSessionHelpers(g.config).ValidateOAuthState(ctx, "github")

	if err != nil || !isValid {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Could not link Github account. Please try again and make sure cookies are enabled.")
	}

	token, err := ghApp.Exchange(context.Background(), ctx.Request().URL.Query().Get("code"))

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Forbidden")
	}

	if !token.Valid() {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, fmt.Errorf("invalid token"), "Forbidden")
	}

	ghClient := github.NewClient(ghApp.Client(context.Background(), token))

	githubUser, _, err := ghClient.Users.Get(context.Background(), "")

	if err != nil {
		return nil, err
	}

	if githubUser.ID == nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Could not get Github user ID.")
	}

	expiresAt := token.Expiry

	// use the encryption service to encrypt the access and refresh token
	accessTokenEncrypted, err := g.config.Encryption.Encrypt([]byte(token.AccessToken), "github_access_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	refreshTokenEncrypted, err := g.config.Encryption.Encrypt([]byte(token.RefreshToken), "github_refresh_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %s", err.Error())
	}

	// upsert in database
	_, err = g.config.APIRepository.Github().UpsertGithubAppOAuth(user.ID, &repository.CreateGithubAppOAuthOpts{
		GithubUserID: int(*githubUser.ID),
		AccessToken:  accessTokenEncrypted,
		RefreshToken: &refreshTokenEncrypted,
		ExpiresAt:    &expiresAt,
	})

	if err != nil {
		return nil, authn.GetRedirectWithError(ctx, g.config.Logger, err, "Internal error.")
	}

	var url string

	if isOAuthTriggered {
		url = fmt.Sprintf("https://github.com/apps/%s/installations/new", ghApp.GetAppName())
	} else {
		url = "/"
	}

	return gen.UserUpdateGithubOauthCallback302Response{
		Headers: gen.UserUpdateGithubOauthCallback302ResponseHeaders{
			Location: url,
		},
	}, nil

}
