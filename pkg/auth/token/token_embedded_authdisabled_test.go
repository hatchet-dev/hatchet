//go:build authdisabled && !e2e && !load && !rampup && !integration

package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/authmode"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// TestEmbeddedTokenValidates checks that the committed embedded token validates against a JWT
// manager built from the embedded keyset with the loader's pinned opts.
func TestEmbeddedTokenValidates(t *testing.T) {
	enc, err := encryption.NewInsecureJWTEncryption(authmode.EmbeddedPrivateKeyset(), authmode.EmbeddedPublicKeyset())
	require.NoError(t, err)

	embeddedTenantID := uuid.MustParse(authmode.EmbeddedTokenTenantID)
	repo := &stubAPITokenRepository{}
	_, err = repo.CreateAPIToken(context.Background(), &v1.CreateAPITokenOpts{
		ID:        uuid.MustParse(authmode.EmbeddedTokenID),
		TenantId:  &embeddedTenantID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)

	mgr, err := token.NewJWTManager(enc, repo, &token.TokenOpts{
		Issuer:               authmode.EmbeddedTokenIssuer,
		Audience:             authmode.EmbeddedTokenAudience,
		ServerURL:            authmode.EmbeddedTokenServerURL,
		GRPCBroadcastAddress: authmode.EmbeddedTokenGRPCAddress,
	})
	require.NoError(t, err)

	tenantID, _, err := mgr.ValidateTenantToken(context.Background(), authmode.EmbeddedToken())
	require.NoError(t, err)
	assert.Equal(t, authmode.EmbeddedTokenTenantID, tenantID.String())
}
