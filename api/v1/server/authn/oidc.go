package authn

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func OIDCProviderUserID(issuer, subject string) string {
	return fmt.Sprintf("%d:%s%s", len(issuer), issuer, subject)
}

func IsCurrentOIDCGlobalAdmin(ctx context.Context, config *server.ServerConfig, userID uuid.UUID) (bool, error) {
	if len(config.Auth.OIDCGroupMappings) == 0 || config.Auth.OIDCProvider == nil {
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

	userInfo, err := config.Auth.OIDCProvider.UserInfo(ctx, oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: string(accessToken),
		TokenType:   "Bearer",
		Expiry:      binding.ExpiresAt.Time,
	}))
	if err != nil {
		return false, fmt.Errorf("get current OIDC user info: %w", err)
	}

	var providerClaims struct {
		Issuer string `json:"issuer"`
	}
	if err := config.Auth.OIDCProvider.Claims(&providerClaims); err != nil {
		return false, fmt.Errorf("get OIDC provider issuer: %w", err)
	}
	if OIDCProviderUserID(providerClaims.Issuer, userInfo.Subject) != binding.ProviderUserId {
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
