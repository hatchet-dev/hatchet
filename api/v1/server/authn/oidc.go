package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// OIDCProviderUserID length-prefixes the issuer so subjects cannot collide across providers.
func OIDCProviderUserID(issuer, subject string) string {
	return fmt.Sprintf("%d:%s%s", len(issuer), issuer, subject)
}

// OIDCSubjectFromClaims requires the configured identity claim to be a non-empty string.
func OIDCSubjectFromClaims(decode func(any) error, claim string) (string, error) {
	if claim == "" {
		claim = "sub"
	}
	claims := make(map[string]json.RawMessage)
	if err := decode(&claims); err != nil {
		return "", err
	}
	var subject string
	if err := json.Unmarshal(claims[claim], &subject); err != nil || subject == "" {
		return "", fmt.Errorf("OIDC provider did not return a string %s claim", claim)
	}
	return subject, nil
}

// IsCurrentOIDCGlobalAdmin checks UserInfo on every call so group revocations take effect without a new login.
func IsCurrentOIDCGlobalAdmin(ctx context.Context, config *server.ServerConfig, userID uuid.UUID) (bool, error) {
	if len(config.Auth.OIDCGroupMappings) == 0 || config.Auth.OIDCProvider == nil || config.Auth.OIDCOAuthConfig == nil {
		return false, nil
	}

	binding, err := config.V1.User().GetUserOAuth(ctx, userID, "oidc")
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get OIDC binding: %w", err)
	}

	accessToken, err := config.Encryption.Decrypt(binding.AccessToken, "oidc_access_token")
	if err != nil {
		return false, fmt.Errorf("decrypt OIDC access token: %w", err)
	}
	refreshToken, err := config.Encryption.Decrypt(binding.RefreshToken, "oidc_refresh_token")
	if err != nil {
		return false, fmt.Errorf("decrypt OIDC refresh token: %w", err)
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
		return false, fmt.Errorf("refresh OIDC access token: %w", err)
	}
	if currentToken.AccessToken != storedToken.AccessToken || currentToken.RefreshToken != storedToken.RefreshToken || !currentToken.Expiry.Equal(storedToken.Expiry) {
		encryptedAccessToken, err := config.Encryption.Encrypt([]byte(currentToken.AccessToken), "oidc_access_token")
		if err != nil {
			return false, fmt.Errorf("encrypt refreshed OIDC access token: %w", err)
		}
		newRefreshToken := currentToken.RefreshToken
		if newRefreshToken == "" {
			newRefreshToken = "none"
		}
		encryptedRefreshToken, err := config.Encryption.Encrypt([]byte(newRefreshToken), "oidc_refresh_token")
		if err != nil {
			return false, fmt.Errorf("encrypt refreshed OIDC refresh token: %w", err)
		}
		if _, err := config.V1.User().UpdateUser(ctx, userID, &v1.UpdateUserOpts{OAuth: &v1.OAuthOpts{
			Provider:       "oidc",
			ProviderUserId: binding.ProviderUserId,
			AccessToken:    encryptedAccessToken,
			RefreshToken:   encryptedRefreshToken,
			ExpiresAt:      &currentToken.Expiry,
		}}); err != nil {
			return false, fmt.Errorf("store refreshed OIDC token: %w", err)
		}
	}

	userInfo, err := config.Auth.OIDCProvider.UserInfo(ctx, oauth2.StaticTokenSource(currentToken))
	if err != nil {
		return false, fmt.Errorf("get current OIDC user info: %w", err)
	}

	var providerClaims struct {
		Issuer string `json:"issuer"`
	}
	if err := config.Auth.OIDCProvider.Claims(&providerClaims); err != nil {
		return false, fmt.Errorf("get OIDC provider issuer: %w", err)
	}
	subject, err := OIDCSubjectFromClaims(userInfo.Claims, config.Auth.ConfigFile.OIDC.SubjectClaim)
	if err != nil {
		return false, fmt.Errorf("get current OIDC subject: %w", err)
	}
	if OIDCProviderUserID(providerClaims.Issuer, subject) != binding.ProviderUserId {
		return false, fmt.Errorf("OIDC UserInfo subject does not match account binding")
	}

	var claims struct {
		Groups *[]string `json:"groups"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return false, fmt.Errorf("parse current OIDC groups: %w", err)
	}
	if claims.Groups == nil {
		return false, fmt.Errorf("OIDC provider did not return a groups claim")
	}

	for _, group := range *claims.Groups {
		role := config.Auth.OIDCGroupMappings[group]
		if role == "OWNER" || role == "ADMIN" {
			return true, nil
		}
	}

	return false, nil
}
