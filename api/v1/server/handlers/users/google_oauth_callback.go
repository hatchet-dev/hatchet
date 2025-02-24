package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

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
func (u *UserService) UserUpdateGoogleOauthCallback(ctx echo.Context, _ gen.UserUpdateGoogleOauthCallbackRequestObject) (gen.UserUpdateGoogleOauthCallbackResponseObject, error) {
	isValid, _, err := authn.NewSessionHelpers(u.config).ValidateOAuthState(ctx, "google")

	if err != nil || !isValid {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	token, err := u.config.Auth.GoogleOAuthConfig.Exchange(context.Background(), ctx.Request().URL.Query().Get("code"))

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Forbidden")
	}

	if !token.Valid() {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, fmt.Errorf("invalid token"), "Forbidden")
	}

	user, err := u.upsertGoogleUserFromToken(ctx.Request().Context(), u.config, token)

	if err != nil {
		if errors.Is(err, ErrNotInRestrictedDomain) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Email is not in the restricted domain group.")
		}

		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	err = authn.NewSessionHelpers(u.config).SaveAuthenticated(ctx, user)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	return gen.UserUpdateGoogleOauthCallback302Response{
		Headers: gen.UserUpdateGoogleOauthCallback302ResponseHeaders{
			Location: u.config.Runtime.ServerURL,
		},
	}, nil
}

func (u *UserService) upsertGoogleUserFromToken(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token) (*dbsqlc.User, error) {
	gInfo, err := getGoogleUserInfoFromToken(tok)
	if err != nil {
		return nil, err
	}

	if err := u.checkUserRestrictions(config, gInfo.HD); err != nil {
		return nil, err
	}

	expiresAt := tok.Expiry

	// use the encryption service to encrypt the access and refresh token
	accessTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.AccessToken), "google_access_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	refreshTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.RefreshToken), "google_refresh_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %s", err.Error())
	}

	oauthOpts := &repository.OAuthOpts{
		Provider:       "google",
		ProviderUserId: gInfo.Sub,
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

type googleUserInfo struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HD            string `json:"hd"`
	Sub           string `json:"sub"`
	Name          string `json:"name"`
}

func getGoogleUserInfoFromToken(tok *oauth2.Token) (*googleUserInfo, error) {
	// use userinfo endpoint for Google OIDC to get claims
	url := "https://openidconnect.googleapis.com/v1/userinfo"

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, fmt.Errorf("failed creating request: %s", err.Error())
	}

	req.Header.Add("Authorization", "Bearer "+tok.AccessToken)

	client := &http.Client{}

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}

	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	// parse contents into Google userinfo claims
	gInfo := &googleUserInfo{}
	err = json.Unmarshal(contents, &gInfo)

	if err != nil {
		return nil, fmt.Errorf("failed parsing response body: %s", err.Error())
	}

	return gInfo, nil
}
