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
	// tokens created without capabilities are stored with the bundle they would otherwise resolve to
	for readOnly, want := range map[bool][]string{false: {"all"}, true: {"read-only"}} {
		generated, err := manager.GenerateTenantToken(context.Background(), uuid.New(), "test token", false, readOnly, nil)
		require.NoError(t, err)
		require.Equal(t, readOnly, repo.opts.ReadOnly)
		require.Equal(t, generated.TokenId, repo.opts.ID)
		require.Equal(t, want, repo.opts.Capabilities)
	}
}

func TestGenerateTenantTokenRejectsInvalidCapabilities(t *testing.T) {
	master, private, public, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)
	enc, err := encryption.NewLocalEncryption(master, private, public)
	require.NoError(t, err)
	repo := &capturingTokenRepo{}
	manager, err := token.NewJWTManager(enc, repo, &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"})
	require.NoError(t, err)

	for _, caps := range [][]string{nil, {}, {"workers:fly"}} {
		_, err := manager.GenerateTenantToken(context.Background(), uuid.New(), "test token", false, false, nil, token.WithCapabilities(caps))
		require.ErrorIs(t, err, token.ErrInvalidCapabilities, "%v", caps)
	}

	require.Nil(t, repo.opts, "nothing is written for a rejected list")

	_, err = manager.GenerateTenantToken(context.Background(), uuid.New(), "test token", false, false, nil, token.WithCapabilities([]string{"worker"}))
	require.NoError(t, err)
	require.Equal(t, []string{"worker"}, repo.opts.Capabilities)
}
