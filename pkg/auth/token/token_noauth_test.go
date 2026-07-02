//go:build !e2e && !load && !rampup && !integration

package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type stubAPITokenRepository struct{}

func (s *stubAPITokenRepository) CreateAPIToken(ctx context.Context, opts *v1.CreateAPITokenOpts) (*sqlcv1.APIToken, error) {
	return &sqlcv1.APIToken{ID: opts.ID, ExpiresAt: pgtype.Timestamp{Time: opts.ExpiresAt, Valid: true}}, nil
}

func (s *stubAPITokenRepository) GetAPITokenById(ctx context.Context, id uuid.UUID) (*sqlcv1.APIToken, error) {
	return &sqlcv1.APIToken{ID: id, Revoked: false, ExpiresAt: pgtype.Timestamp{Time: time.Now().Add(24 * time.Hour), Valid: true}}, nil
}

func (s *stubAPITokenRepository) ListAPITokensByTenant(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.APIToken, error) {
	return nil, nil
}

func (s *stubAPITokenRepository) RevokeAPIToken(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (s *stubAPITokenRepository) DeleteAPIToken(ctx context.Context, tenantId, id uuid.UUID) error {
	return nil
}

func TestNoAuthTokenTrustedOnlyWithVerifier(t *testing.T) {
	master, mainPriv, mainPub, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	noAuthPriv, noAuthPub, _, err := encryption.GenerateJWTKeysets(master)
	require.NoError(t, err)

	mainEnc, err := encryption.NewLocalEncryption(master, mainPriv, mainPub)
	require.NoError(t, err)

	noAuthEnc, err := encryption.NewLocalEncryption(master, noAuthPriv, noAuthPub)
	require.NoError(t, err)

	repo := &stubAPITokenRepository{}
	opts := &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"}

	mgrWithNoAuth, err := token.NewJWTManager(mainEnc, repo, opts, token.WithNoAuthVerifier(noAuthEnc))
	require.NoError(t, err)

	mgrPlain, err := token.NewJWTManager(mainEnc, repo, opts)
	require.NoError(t, err)

	noAuthMinter, err := token.NewJWTManager(noAuthEnc, repo, opts)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	noAuthTok, err := noAuthMinter.GenerateTenantToken(ctx, tenantID, "noauth", false, nil)
	require.NoError(t, err)

	gotTenant, _, err := mgrWithNoAuth.ValidateTenantToken(ctx, noAuthTok.Token)
	assert.NoError(t, err, "no-auth token should validate when the no-auth verifier is loaded")
	assert.Equal(t, tenantID, gotTenant)

	_, _, err = mgrPlain.ValidateTenantToken(ctx, noAuthTok.Token)
	assert.Error(t, err, "no-auth token must be rejected without the no-auth verifier")

	mainTok, err := mgrPlain.GenerateTenantToken(ctx, tenantID, "main", false, nil)
	require.NoError(t, err)

	_, _, err = mgrPlain.ValidateTenantToken(ctx, mainTok.Token)
	assert.NoError(t, err, "main token should validate on a plain manager")

	_, _, err = mgrWithNoAuth.ValidateTenantToken(ctx, mainTok.Token)
	assert.NoError(t, err, "main token should validate on a no-auth manager")
}
