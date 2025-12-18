package users

import (
	"context"
	"errors"
	"fmt"

	githubsdk "github.com/google/go-github/v57/github"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateGithubOauthCallback(ctx echo.Context, _ gen.UserUpdateGithubOauthCallbackRequestObject) (gen.UserUpdateGithubOauthCallbackResponseObject, error) {
	isValid, _, err := authn.NewSessionHelpers(u.config).ValidateOAuthState(ctx, "github")

	if err != nil || !isValid {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	token, err := u.config.Auth.GithubOAuthConfig.Exchange(context.Background(), ctx.Request().URL.Query().Get("code"))

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Forbidden")
	}

	if !token.Valid() {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, fmt.Errorf("invalid token"), "Forbidden")
	}

	user, err := u.upsertGithubUserFromToken(ctx.Request().Context(), u.config, token)

	if err != nil {
		if errors.Is(err, ErrNotInRestrictedDomain) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Email is not in the restricted domain group.")
		}

		if errors.Is(err, ErrGithubNotVerified) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Please verify your email on Github.")
		}

		if errors.Is(err, ErrGithubNoEmail) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Github user must have an email.")
		}

		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, user)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	return gen.UserUpdateGithubOauthCallback302Response{
		Headers: gen.UserUpdateGithubOauthCallback302ResponseHeaders{
			Location: u.config.Runtime.ServerURL,
		},
	}, nil
}

func (u *UserService) upsertGithubUserFromToken(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token) (*dbsqlc.User, error) {
	gInfo, err := u.getGithubEmailFromToken(tok)

	if err != nil {
		return nil, err
	}

	if err := u.checkUserRestrictionsForEmail(config, gInfo.Email); err != nil {
		return nil, err
	}

	expiresAt := tok.Expiry

	// use the encryption service to encrypt the access and refresh token
	accessTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.AccessToken), "github_access_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	refreshTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.RefreshToken), "github_refresh_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %s", err.Error())
	}

	oauthOpts := &repository.OAuthOpts{
		Provider:       "github",
		ProviderUserId: gInfo.ID,
		AccessToken:    accessTokenEncrypted,
		RefreshToken:   refreshTokenEncrypted,
		ExpiresAt:      &expiresAt,
	}

	user, err := u.config.APIRepository.User().GetUserByEmail(ctx, gInfo.Email)

	switch err {
	case nil:
		user, err = u.config.APIRepository.User().UpdateUser(ctx, sqlchelpers.UUIDToStr(user.ID), &repository.UpdateUserOpts{
			EmailVerified: repository.BoolPtr(gInfo.EmailVerified),
			Name:          repository.StringPtr(gInfo.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to update user: %s", err.Error())
		}
	case pgx.ErrNoRows:
		user, err = u.config.APIRepository.User().CreateUser(ctx, &repository.CreateUserOpts{
			Email:         gInfo.Email,
			EmailVerified: repository.BoolPtr(gInfo.EmailVerified),
			Name:          repository.StringPtr(gInfo.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %s", err.Error())
		}
	default:
		return nil, fmt.Errorf("failed to get user: %s", err.Error())
	}

	return user, nil
}

var ErrGithubNotVerified = fmt.Errorf("Please verify your email on Github")
var ErrGithubNoEmail = fmt.Errorf("Github user must have an email")

type githubInfo struct {
	Email         string
	EmailVerified bool
	Name          string
	ID            string
}

func (u *UserService) getGithubEmailFromToken(tok *oauth2.Token) (*githubInfo, error) {
	client := githubsdk.NewClient(u.config.Auth.GithubOAuthConfig.Client(context.Background(), tok))

	user, _, err := client.Users.Get(context.Background(), "")

	if err != nil {
		return nil, err
	}

	emails, _, err := client.Users.ListEmails(context.Background(), &githubsdk.ListOptions{})

	if err != nil {
		return nil, err
	}

	primary := ""
	verified := false

	// get the primary email
	for _, email := range emails {
		if email.GetPrimary() {
			primary = email.GetEmail()
			verified = email.GetVerified()
			break
		}
	}

	if primary == "" {
		return nil, ErrGithubNoEmail
	}

	if !verified {
		return nil, ErrGithubNotVerified
	}

	id := user.GetID()

	if id == 0 {
		return nil, fmt.Errorf("Github user has no ID")
	}

	return &githubInfo{
		Email:         primary,
		EmailVerified: verified,
		Name:          user.GetName(),
		ID:            fmt.Sprintf("%d", id),
	}, nil
}
