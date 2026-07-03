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

func TestAuthDisabledTokenTrustedOnlyWithVerifier(t *testing.T) {
	master, mainPriv, mainPub, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	adPriv, adPub, err := encryption.GenerateInsecureJWTKeyset()
	require.NoError(t, err)

	mainEnc, err := encryption.NewLocalEncryption(master, mainPriv, mainPub)
	require.NoError(t, err)

	repo := &stubAPITokenRepository{}
	opts := &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"}

	mgrWithAuthDisabled, err := token.NewJWTManager(mainEnc, repo, opts, token.WithAuthDisabledVerifier(adPub))
	require.NoError(t, err)

	mgrPlain, err := token.NewJWTManager(mainEnc, repo, opts)
	require.NoError(t, err)

	minter, err := token.NewJWTManagerFromKeysets(adPriv, adPub, repo, opts)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	adTok, err := minter.GenerateTenantToken(ctx, tenantID, "authdisabled", false, nil)
	require.NoError(t, err)

	gotTenant, _, err := mgrWithAuthDisabled.ValidateTenantToken(ctx, adTok.Token)
	assert.NoError(t, err, "embedded-keyset token should validate in an authdisabled build")
	assert.Equal(t, tenantID, gotTenant)

	_, _, err = mgrPlain.ValidateTenantToken(ctx, adTok.Token)
	assert.Error(t, err, "embedded-keyset token must be rejected without the authdisabled verifier")

	mainTok, err := mgrPlain.GenerateTenantToken(ctx, tenantID, "main", false, nil)
	require.NoError(t, err)

	_, _, err = mgrPlain.ValidateTenantToken(ctx, mainTok.Token)
	assert.NoError(t, err, "main token should validate on a plain manager")

	_, _, err = mgrWithAuthDisabled.ValidateTenantToken(ctx, mainTok.Token)
	assert.NoError(t, err, "main token should validate on an authdisabled manager")
}
