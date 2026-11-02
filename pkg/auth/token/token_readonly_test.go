//go:build !e2e && !load && !rampup && !integration

package token_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/stretchr/testify/require"
)

type capturingTokenRepo struct {
	stubAPITokenRepository
	opts *v1.CreateAPITokenOpts
}

func (r *capturingTokenRepo) CreateAPIToken(ctx context.Context, opts *v1.CreateAPITokenOpts) (*sqlcv1.APIToken, error) {
	r.opts = opts
	return r.stubAPITokenRepository.CreateAPIToken(ctx, opts)
}
func TestGenerateTenantTokenReadOnly(t *testing.T) {
	master, private, public, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)
	enc, err := encryption.NewLocalEncryption(master, private, public)
	require.NoError(t, err)
	repo := &capturingTokenRepo{}
	manager, err := token.NewJWTManager(enc, repo, &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"})
	require.NoError(t, err)
	for _, readOnly := range []bool{false, true} {
		generated, err := manager.GenerateTenantToken(context.Background(), uuid.New(), "test token", false, readOnly, nil)
		require.NoError(t, err)
		require.Equal(t, readOnly, repo.opts.ReadOnly)
		require.Equal(t, generated.TokenId, repo.opts.ID)
	}
}
