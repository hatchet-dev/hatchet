package authn

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/api/v1/server/oidcrbac"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// OIDCProviderUserID length-prefixes the issuer so subjects cannot collide across providers.
func OIDCProviderUserID(issuer, subject string) string {
	return fmt.Sprintf("%d:%s%s", len(issuer), issuer, subject)
}

// OIDCIssuer reads the verified issuer from the provider's discovery document.
func OIDCIssuer(config *server.ServerConfig) (string, error) {
	if config.Auth.OIDCProvider == nil {
		return "", fmt.Errorf("OIDC provider is not configured")
	}
	var claims struct {
		Issuer string `json:"issuer"`
	}
	if err := config.Auth.OIDCProvider.Claims(&claims); err != nil {
		return "", fmt.Errorf("get OIDC provider issuer: %w", err)
	}
	if claims.Issuer == "" {
		return "", fmt.Errorf("OIDC provider discovery document has no issuer")
	}
	return claims.Issuer, nil
}

// IsCurrentOIDCGlobalAdmin checks UserInfo on every call so group revocations take effect without a new login.
func IsCurrentOIDCGlobalAdmin(ctx context.Context, config *server.ServerConfig, userID uuid.UUID) (bool, error) {
	if len(config.Auth.OIDCGroupMappings) == 0 || config.Auth.OIDCProvider == nil || config.Auth.OIDCOAuthConfig == nil {
		return false, nil
	}
	groups, found, err := currentOIDCGroups(ctx, config, userID)
	if err != nil || !found {
		return false, err
	}
	for _, group := range groups {
		role := config.Auth.OIDCGroupMappings[group]
		if role == "OWNER" || role == "ADMIN" {
			return true, nil
		}
	}
	return false, nil
}

// ReconcileCurrentOIDCMemberships applies current provider groups before authorization.
func ReconcileCurrentOIDCMemberships(ctx context.Context, config *server.ServerConfig, userID uuid.UUID) error {
	if config.Auth.OIDCProvider == nil {
		return nil
	}
	if config.Layer == nil || config.Layer.Pool == nil {
		return fmt.Errorf("OIDC group mapping requires a database")
	}
	issuer, err := OIDCIssuer(config)
	if err != nil {
		return err
	}
	hasMappings := len(config.Auth.OIDCGroupMappings) > 0
	if !hasMappings {
		hasMappings, err = sqlcv1.New().HasOIDCGroupMappings(ctx, config.Layer.Pool, issuer)
		if err != nil {
			return err
		}
	}
	groups := []string(nil)
	if hasMappings {
		var found bool
		groups, found, err = currentOIDCGroups(ctx, config, userID)
		if err != nil {
			return err
		}
		if !found {
			groups = nil
		}
	}
	return oidcrbac.ReconcileTenantMemberships(ctx, config, userID, issuer, groups)
}

func currentOIDCGroups(ctx context.Context, config *server.ServerConfig, userID uuid.UUID) ([]string, bool, error) {
	if config.Auth.OIDCProvider == nil || config.Auth.OIDCOAuthConfig == nil {
		return nil, false, nil
	}
	binding, err := config.V1.User().GetUserOAuth(ctx, userID, "oidc")
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get OIDC binding: %w", err)
	}

	accessToken, err := config.Encryption.Decrypt(binding.AccessToken, "oidc_access_token")
	if err != nil {
		return nil, false, fmt.Errorf("decrypt OIDC access token: %w", err)
	}
	refreshToken, err := config.Encryption.Decrypt(binding.RefreshToken, "oidc_refresh_token")
	if err != nil {
		return nil, false, fmt.Errorf("decrypt OIDC refresh token: %w", err)
	}
	if string(refreshToken) == "none" {
		refreshToken = nil
	}

	storedToken := &oauth2.Token{
		AccessToken:  string(accessToken),
		RefreshToken: string(refreshToken),
		TokenType:    "Bearer",
		Expiry:       binding.ExpiresAt.Time,
	}
	currentToken, err := config.Auth.OIDCOAuthConfig.TokenSource(ctx, storedToken).Token()
	if err != nil {
		return nil, false, fmt.Errorf("refresh OIDC access token: %w", err)
	}
	if currentToken.AccessToken != storedToken.AccessToken || currentToken.RefreshToken != storedToken.RefreshToken || !currentToken.Expiry.Equal(storedToken.Expiry) {
		encryptedAccessToken, err := config.Encryption.Encrypt([]byte(currentToken.AccessToken), "oidc_access_token")
		if err != nil {
			return nil, false, fmt.Errorf("encrypt refreshed OIDC access token: %w", err)
		}
		newRefreshToken := currentToken.RefreshToken
		if newRefreshToken == "" {
			newRefreshToken = "none"
		}
		encryptedRefreshToken, err := config.Encryption.Encrypt([]byte(newRefreshToken), "oidc_refresh_token")
		if err != nil {
			return nil, false, fmt.Errorf("encrypt refreshed OIDC refresh token: %w", err)
		}
		if _, err := config.V1.User().UpdateUser(ctx, userID, &v1.UpdateUserOpts{OAuth: &v1.OAuthOpts{
			Provider:       "oidc",
			ProviderUserId: binding.ProviderUserId,
			AccessToken:    encryptedAccessToken,
			RefreshToken:   encryptedRefreshToken,
			ExpiresAt:      &currentToken.Expiry,
		}}); err != nil {
			return nil, false, fmt.Errorf("store refreshed OIDC token: %w", err)
		}
	}

	userInfo, err := config.Auth.OIDCProvider.UserInfo(ctx, oauth2.StaticTokenSource(currentToken))
	if err != nil {
		return nil, false, fmt.Errorf("get current OIDC user info: %w", err)
	}

	issuer, err := OIDCIssuer(config)
	if err != nil {
		return nil, false, err
	}
	if userInfo.Subject == "" {
		return nil, false, fmt.Errorf("OIDC provider did not return a sub claim")
	}
	if OIDCProviderUserID(issuer, userInfo.Subject) != binding.ProviderUserId {
		return nil, false, fmt.Errorf("OIDC UserInfo subject does not match account binding")
	}

	var claims struct {
		Groups *[]string `json:"groups"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return nil, false, fmt.Errorf("parse current OIDC groups: %w", err)
	}
	if claims.Groups == nil {
		return nil, false, fmt.Errorf("OIDC provider did not return a groups claim")
	}
	return *claims.Groups, true, nil
}
