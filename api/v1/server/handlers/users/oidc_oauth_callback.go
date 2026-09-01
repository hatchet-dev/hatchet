package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/authn"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oidcrbac"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Note: we want all errors to redirect, otherwise the user will be greeted with raw JSON in the middle of the login flow.
func (u *UserService) UserUpdateOidcOauthCallback(ctx echo.Context, _ gen.UserUpdateOidcOauthCallbackRequestObject) (gen.UserUpdateOidcOauthCallbackResponseObject, error) {
	if u.config.Auth.OIDCOAuthConfig == nil || u.config.Auth.OIDCProvider == nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, nil, "OIDC authentication is not configured.")
	}

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

	claims, err := getOIDCClaimsFromToken(ctx.Request().Context(), u.config, token)
	if err != nil {
		return nil, redirect.GetRedirectWithError(ctx, u.config.Logger, err, "Internal error.")
	}

	user, err := u.upsertOIDCUserFromClaims(ctx.Request().Context(), u.config, token, claims)

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
	return u.upsertOIDCUserFromClaims(ctx, config, tok, claims)
}

func (u *UserService) upsertOIDCUserFromClaims(ctx context.Context, config *server.ServerConfig, tok *oauth2.Token, claims *oidcClaims) (*sqlcv1.User, error) {
	if err := u.checkUserRestrictionsForEmail(config, claims.Email); err != nil {
		return nil, err
	}
	releaseSubjectLock, err := acquireOIDCSubjectLock(ctx, config, claims.Issuer, claims.Sub)
	if err != nil {
		return nil, err
	}
	defer releaseSubjectLock()

	emailVerified := claims.EmailVerified || config.Auth.ConfigFile.SetEmailVerified

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

	providerUserID := authn.OIDCProviderUserID(claims.Issuer, claims.Sub)
	oauthOpts := &v1.OAuthOpts{
		Provider:       "oidc",
		ProviderUserId: providerUserID,
		AccessToken:    accessTokenEncrypted,
		RefreshToken:   refreshTokenEncrypted,
		ExpiresAt:      &expiresAt,
	}
	subjectBinding, subjectErr := u.config.V1.User().GetUserOAuthByProviderUserID(ctx, "oidc", providerUserID)
	if subjectErr != nil && !errors.Is(subjectErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get OIDC subject binding: %s", subjectErr.Error())
	}

	user, err := u.config.V1.User().GetUserByEmail(ctx, claims.Email)

	switch err {
	case nil:
		if subjectErr == nil && subjectBinding.UserId != user.ID {
			return nil, fmt.Errorf("OIDC subject is already linked to another account")
		}
		existingOAuth, oauthErr := u.config.V1.User().GetUserOAuth(ctx, user.ID, "oidc")
		if oauthErr == nil && existingOAuth.ProviderUserId != providerUserID {
			return nil, fmt.Errorf("OIDC subject does not match the existing account binding")
		}
		if oauthErr != nil && !errors.Is(oauthErr, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to get OIDC account binding: %s", oauthErr.Error())
		}
		if errors.Is(oauthErr, pgx.ErrNoRows) && !emailVerified {
			return nil, fmt.Errorf("cannot link OIDC identity to an existing account without a verified email")
		}
		user, err = u.config.V1.User().UpdateUser(ctx, user.ID, &v1.UpdateUserOpts{
			EmailVerified: v1.BoolPtr(emailVerified),
			Name:          v1.StringPtr(claims.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to update user: %s", err.Error())
		}
	case pgx.ErrNoRows:
		if subjectErr == nil {
			return nil, fmt.Errorf("OIDC subject is already linked to another account")
		}
		if !config.Runtime.AllowSignup {
			return nil, fmt.Errorf("user signup is disabled")
		}

		user, err = u.config.V1.User().CreateUser(ctx, &v1.CreateUserOpts{
			Email:         claims.Email,
			EmailVerified: v1.BoolPtr(emailVerified),
			Name:          v1.StringPtr(claims.Name),
			OAuth:         oauthOpts,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %s", err.Error())
		}
	default:
		return nil, fmt.Errorf("failed to get user: %s", err.Error())
	}

	groups := []string(nil)
	if claims.Groups != nil {
		groups = *claims.Groups
	}
	if err := oidcrbac.ReconcileTenantMemberships(ctx, config, user.ID, claims.Issuer, groups); err != nil {
		return nil, fmt.Errorf("failed to reconcile OIDC tenant memberships: %w", err)
	}

	return user, nil
}

func acquireOIDCSubjectLock(ctx context.Context, config *server.ServerConfig, issuer, subject string) (func(), error) {
	if config.Layer == nil || config.Layer.Pool == nil {
		return nil, fmt.Errorf("OIDC authentication requires a database")
	}
	conn, err := config.Layer.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	lockKey := issuer + "\n" + subject
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtextextended($1, 0))", lockKey); err != nil {
		conn.Release()
		return nil, err
	}
	return func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock(hashtextextended($1, 0))", lockKey)
		conn.Release()
	}, nil
}

type oidcClaims struct {
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Name          string    `json:"name"`
	Sub           string    `json:"sub"`
	Groups        *[]string `json:"groups"`
	Issuer        string    `json:"-"`
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
	claims.Issuer = idToken.Issuer
	requireGroups, err := oidcGroupMappingsEnabled(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to check OIDC group mappings: %w", err)
	}

	// Fall back to UserInfo endpoint when the ID token is missing optional claims.
	// The OIDC spec does not require providers to include email/name in the ID token.
	if claims.Email == "" || claims.Name == "" || !claims.EmailVerified ||
		(requireGroups && claims.Groups == nil) {
		userInfo, uiErr := config.Auth.OIDCProvider.UserInfo(ctx, oauth2.StaticTokenSource(tok))
		if uiErr == nil {
			uiClaims := &oidcClaims{}
			if err := userInfo.Claims(uiClaims); err == nil {
				if uiClaims.Sub == "" || claims.Sub == "" || uiClaims.Sub != claims.Sub {
					return nil, fmt.Errorf("OIDC UserInfo sub claim (%s) does not match ID token sub claim (%s)", uiClaims.Sub, claims.Sub)
				}
				if claims.Email == "" {
					claims.Email = uiClaims.Email
					claims.EmailVerified = uiClaims.EmailVerified
				} else if uiClaims.Email != "" && !strings.EqualFold(claims.Email, uiClaims.Email) {
					return nil, fmt.Errorf("OIDC UserInfo email claim does not match ID token email claim")
				}
				if claims.Name == "" {
					claims.Name = uiClaims.Name
				}
				if !claims.EmailVerified && uiClaims.EmailVerified && uiClaims.Email != "" {
					claims.EmailVerified = true
				}
				if claims.Groups == nil {
					claims.Groups = uiClaims.Groups
				}
			}
		}
	}

	if claims.Email == "" {
		return nil, fmt.Errorf("OIDC provider did not return an email claim")
	}

	if claims.Sub == "" {
		return nil, fmt.Errorf("OIDC provider did not return a sub claim")
	}
	if requireGroups && claims.Groups == nil {
		return nil, fmt.Errorf("OIDC provider did not return a groups claim")
	}

	return claims, nil
}

func oidcGroupMappingsEnabled(ctx context.Context, config *server.ServerConfig) (bool, error) {
	if len(config.Auth.OIDCGroupMappings) > 0 {
		return true, nil
	}
	if config.Layer == nil || config.Layer.Pool == nil {
		return false, nil
	}
	issuer, err := authn.OIDCIssuer(config)
	if err != nil {
		return false, err
	}
	return sqlcv1.New().HasOIDCGroupMappings(ctx, config.Layer.Pool, issuer)
}
