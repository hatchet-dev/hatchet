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
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateAzureOauthCallback(ctx echo.Context, _ gen.UserUpdateAzureOauthCallbackRequestObject) (gen.UserUpdateAzureOauthCallbackResponseObject, error) {
	isValid, _, err := authn.NewSessionHelpers(u.config.SessionStore).ValidateOAuthState(ctx, "azure")

	if err != nil || !isValid {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	token, err := u.config.Auth.AzureOAuthConfig.Exchange(context.Background(), ctx.Request().URL.Query().Get("code"))

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Forbidden")
	}

	if !token.Valid() {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, fmt.Errorf("invalid token"), "Forbidden")
	}

	user, err := u.upsertAzureUserFromToken(ctx.Request().Context(), u.config, token)

	if err != nil {
		if errors.Is(err, ErrNotInRestrictedDomain) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Email is not in the restricted domain group.")
		}

		if errors.Is(err, ErrAzureNoEmail) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Azure account must have an email.")
		}

		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	err = authn.NewSessionHelpers(u.config.SessionStore).SaveAuthenticated(ctx, user)

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	analyticsCtx := context.WithValue(ctx.Request().Context(), analytics.UserIDKey, user.ID)
	analyticsCtx = context.WithValue(analyticsCtx, analytics.SourceKey, analytics.SourceUI)
	u.config.Analytics.Enqueue(
		analyticsCtx,
		analytics.User, analytics.Login,
		user.ID.String(),
		map[string]interface{}{"provider": "azure"},
	)
	return gen.UserUpdateAzureOauthCallback302Response{
		Headers: gen.UserUpdateAzureOauthCallback302ResponseHeaders{
			Location: u.config.Runtime.ServerURL,
		},
	}, nil
}

func (u *UserService) upsertAzureUserFromToken(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token) (*sqlcv1.User, error) {
	aInfo, err := getAzureUserInfoFromToken(tok)
	if err != nil {
		return nil, err
	}

	// Azure AD does not expose a Google-style "hd" (hosted domain) claim, so we
	// gate on the email's domain like the GitHub provider does.
	if err := u.checkUserRestrictionsForEmail(config, aInfo.Email); err != nil {
		return nil, err
	}

	expiresAt := tok.Expiry

	// use the encryption service to encrypt the access and refresh token
	accessTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.AccessToken), "azure_access_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	refreshTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.RefreshToken), "azure_refresh_token")

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %s", err.Error())
	}

	oauthOpts := &v1.OAuthOpts{
		Provider:       "azure",
		ProviderUserId: aInfo.Sub,
		AccessToken:    accessTokenEncrypted,
		RefreshToken:   refreshTokenEncrypted,
		ExpiresAt:      &expiresAt,
	}

	user, err := u.config.V1.User().GetUserByEmail(ctx, aInfo.Email)

	switch err {
	case nil:
		user, err = u.config.V1.User().UpdateUser(ctx, user.ID, &v1.UpdateUserOpts{
			EmailVerified: v1.BoolPtr(aInfo.EmailVerified),
			Name:          v1.StringPtr(aInfo.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to update user: %s", err.Error())
		}
	case pgx.ErrNoRows:
		user, err = u.config.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
			Email:         aInfo.Email,
			EmailVerified: v1.BoolPtr(aInfo.EmailVerified),
			Name:          v1.StringPtr(aInfo.Name),
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

var ErrAzureNoEmail = fmt.Errorf("azure account must have an email")

type azureUserInfo struct {
	Email         string
	EmailVerified bool
	Sub           string
	Name          string
}

// azureUserInfoResponse mirrors the claims returned by the Microsoft identity
// platform OIDC userinfo endpoint. The `email` claim is only present when the
// "email" scope is granted and the account has a mail attribute; when it is
// absent we fall back to `preferred_username`, which for work/school accounts
// is the UPN (typically the user's email address).
type azureUserInfoResponse struct {
	Sub               string `json:"sub"`
	Name              string `json:"name"`
	Email             string `json:"email"`
	PreferredUsername string `json:"preferred_username"`
	// email_verified is not consistently returned by Azure AD. When absent we
	// treat the email as verified, since the user has authenticated against the
	// Azure AD directory. When Azure explicitly returns false, we honor it.
	EmailVerified *bool `json:"email_verified"`
}

func getAzureUserInfoFromToken(tok *oauth2.Token) (*azureUserInfo, error) {
	// use the Microsoft identity platform OIDC userinfo endpoint to get claims
	url := "https://graph.microsoft.com/oidc/userinfo"

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

	// parse contents into Azure userinfo claims
	resp := &azureUserInfoResponse{}
	err = json.Unmarshal(contents, &resp)

	if err != nil {
		return nil, fmt.Errorf("failed parsing response body: %s", err.Error())
	}

	email := resp.Email
	if email == "" {
		email = resp.PreferredUsername
	}

	if email == "" {
		return nil, ErrAzureNoEmail
	}

	emailVerified := true
	if resp.EmailVerified != nil {
		emailVerified = *resp.EmailVerified
	}

	return &azureUserInfo{
		Email:         email,
		EmailVerified: emailVerified,
		Sub:           resp.Sub,
		Name:          resp.Name,
	}, nil
}
