package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
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
func (u *UserService) UserUpdateOidcOauthCallback(ctx echo.Context, _ gen.UserUpdateOidcOauthCallbackRequestObject) (gen.UserUpdateOidcOauthCallbackResponseObject, error) {
	isValid, _, err := authn.NewSessionHelpers(u.config.SessionStore).ValidateOAuthState(ctx, "oidc")

	if err != nil || !isValid {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Could not log in. Please try again and make sure cookies are enabled.")
	}

	token, err := u.config.Auth.OIDCOAuthConfig.Exchange(ctx.Request().Context(), ctx.Request().URL.Query().Get("code"))

	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Forbidden")
	}

	if !token.Valid() {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, fmt.Errorf("invalid token"), "Forbidden")
	}

	user, err := u.upsertOIDCUserFromToken(ctx.Request().Context(), u.config, token)

	if err != nil {
		if errors.Is(err, ErrNotInRestrictedDomain) {
			return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Email is not in the restricted domain group.")
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
		map[string]interface{}{"provider": "oidc"},
	)

	return gen.UserUpdateOidcOauthCallback302Response{
		Headers: gen.UserUpdateOidcOauthCallback302ResponseHeaders{
			Location: u.config.Runtime.ServerURL,
		},
	}, nil
}

func (u *UserService) upsertOIDCUserFromToken(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token) (*sqlcv1.User, error) {
	claims, err := getOIDCClaimsFromToken(ctx, config, tok)
	if err != nil {
		return nil, err
	}

	if err := u.checkUserRestrictionsForEmail(config, claims.Email); err != nil {
		return nil, err
	}

	expiresAt := tok.Expiry

	accessTokenEncrypted, err := config.Encryption.Encrypt([]byte(tok.AccessToken), "oidc_access_token")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %s", err.Error())
	}

	refreshToken := tok.RefreshToken
	if refreshToken == "" {
		refreshToken = "none"
	}

	refreshTokenEncrypted, err := config.Encryption.Encrypt([]byte(refreshToken), "oidc_refresh_token")
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt refresh token: %s", err.Error())
	}

	oauthOpts := &v1.OAuthOpts{
		Provider:       "oidc",
		ProviderUserId: claims.Sub,
		AccessToken:    accessTokenEncrypted,
		RefreshToken:   refreshTokenEncrypted,
		ExpiresAt:      &expiresAt,
	}

	user, err := u.config.V1.User().GetUserByEmail(ctx, claims.Email)

	switch err {
	case nil:
		user, err = u.config.V1.User().UpdateUser(ctx, user.ID, &v1.UpdateUserOpts{
			EmailVerified: v1.BoolPtr(claims.EmailVerified),
			Name:          v1.StringPtr(claims.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to update user: %s", err.Error())
		}
	case pgx.ErrNoRows:
		user, err = u.config.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
			Email:         claims.Email,
			EmailVerified: v1.BoolPtr(claims.EmailVerified),
			Name:          v1.StringPtr(claims.Name),
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

type oidcClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Sub           string `json:"sub"`
}

func getOIDCClaimsFromToken(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token) (*oidcClaims, error) {
	verifier := config.Auth.OIDCProvider.Verifier(&oidc.Config{
		ClientID: config.Auth.OIDCOAuthConfig.ClientID,
	})

	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %s", err.Error())
	}

	claims := &oidcClaims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("failed to parse ID token claims: %s", err.Error())
	}

	if claims.Email == "" {
		return nil, fmt.Errorf("OIDC provider did not return an email claim")
	}

	if claims.Sub == "" {
		return nil, fmt.Errorf("OIDC provider did not return a sub claim")
	}

	return claims, nil
}
