//go:build !e2e && !load && !rampup && !integration

package token_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tink-crypto/tink-go/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"

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

// TestAuthDisabledKeysetIsolation is the security invariant behind authdisabled mode: the main
// (production) JWT manager can never validate a token signed by the embedded authdisabled keyset,
// and vice versa. The embedded token is only accepted via the separate authdisabled manager, which
// is only wired up (in the gRPC middleware) behind the compile-time authmode.Disabled constant.
func TestAuthDisabledKeysetIsolation(t *testing.T) {
	master, mainPriv, mainPub, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	mainEnc, err := encryption.NewLocalEncryption(master, mainPriv, mainPub)
	require.NoError(t, err)

	adPriv, adPub := generateInsecureJWTKeyset(t)
	adEnc, err := encryption.NewInsecureJWTEncryption(adPriv, adPub)
	require.NoError(t, err)

	repo := &stubAPITokenRepository{}
	opts := &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"}

	mainMgr, err := token.NewJWTManager(mainEnc, repo, opts)
	require.NoError(t, err)

	adMgr, err := token.NewJWTManager(adEnc, repo, opts)
	require.NoError(t, err)

	ctx := context.Background()
	tenantID := uuid.New()

	adTok, err := adMgr.GenerateTenantToken(ctx, tenantID, "authdisabled", false, nil)
	require.NoError(t, err)

	gotTenant, _, err := adMgr.ValidateTenantToken(ctx, adTok.Token)
	assert.NoError(t, err, "embedded-keyset token should validate on the authdisabled manager")
	assert.Equal(t, tenantID, gotTenant)

	_, _, err = mainMgr.ValidateTenantToken(ctx, adTok.Token)
	assert.Error(t, err, "the main JWT manager must reject the embedded-keyset token")

	mainTok, err := mainMgr.GenerateTenantToken(ctx, tenantID, "main", false, nil)
	require.NoError(t, err)

	_, _, err = adMgr.ValidateTenantToken(ctx, mainTok.Token)
	assert.Error(t, err, "the authdisabled manager must reject a main-keyset token")
}

func generateInsecureJWTKeyset(t *testing.T) (private, public []byte) {
	t.Helper()

	privateHandle, err := keyset.NewHandle(jwt.ES256Template())
	require.NoError(t, err)

	publicHandle, err := privateHandle.Public()
	require.NoError(t, err)

	return insecureKeysetBytes(t, privateHandle), insecureKeysetBytes(t, publicHandle)
}

func insecureKeysetBytes(t *testing.T, kh *keyset.Handle) []byte {
	t.Helper()

	buf := new(bytes.Buffer)
	require.NoError(t, insecurecleartextkeyset.Write(kh, keyset.NewJSONWriter(buf)))

	out := make([]byte, base64.RawStdEncoding.EncodedLen(buf.Len()))
	base64.RawStdEncoding.Encode(out, buf.Bytes())

	return out
}
