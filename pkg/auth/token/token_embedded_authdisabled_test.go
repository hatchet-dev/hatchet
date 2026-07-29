//go:build authdisabled && !e2e && !load && !rampup && !integration

package token_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/authmode"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

// TestEmbeddedTokenValidates checks that the committed embedded token validates against a JWT
// manager built from the embedded keyset with the loader's pinned opts.
func TestEmbeddedTokenValidates(t *testing.T) {
	enc, err := encryption.NewInsecureJWTEncryption(authmode.EmbeddedPrivateKeyset(), authmode.EmbeddedPublicKeyset())
	require.NoError(t, err)

	mgr, err := token.NewJWTManager(enc, &stubAPITokenRepository{}, &token.TokenOpts{
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
