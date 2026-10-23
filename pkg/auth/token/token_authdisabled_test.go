//go:build !e2e && !load && !rampup && !integration

package token_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

type stubAPITokenRepository struct {
	tokens map[uuid.UUID]*sqlcv1.APIToken
}

func (s *stubAPITokenRepository) CreateAPIToken(ctx context.Context, opts *v1.CreateAPITokenOpts) (*sqlcv1.APIToken, error) {
	tok := &sqlcv1.APIToken{
		ID:        opts.ID,
		TenantId:  opts.TenantId,
		ExpiresAt: pgtype.Timestamp{Time: opts.ExpiresAt, Valid: true},
	}

	if s.tokens == nil {
		s.tokens = make(map[uuid.UUID]*sqlcv1.APIToken)
	}

	s.tokens[opts.ID] = tok

	return tok, nil
}

func (s *stubAPITokenRepository) GetAPITokenById(ctx context.Context, id uuid.UUID) (*sqlcv1.APIToken, error) {
	if tok, ok := s.tokens[id]; ok {
		return tok, nil
	}

	return nil, pgx.ErrNoRows
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
// is only wired up (in the gRPC middleware) behind the compile-time authmode.IsDisabled constant.
func TestAuthDisabledKeysetIsolation(t *testing.T) {
	mainEnc := newLocalEncryption(t)

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

func TestRejectsTokenWhenSubjectDoesNotMatchStoredTenant(t *testing.T) {
	ctx := context.Background()
	opts := &token.TokenOpts{Issuer: "hatchet", Audience: "hatchet", ServerURL: "http://localhost:8080"}

	enc := newLocalEncryption(t)
	mgr, err := token.NewJWTManager(enc, &stubAPITokenRepository{}, opts)
	require.NoError(t, err)

	tenantA := uuid.New()
	tenantB := uuid.New()

	tokenA, err := mgr.GenerateTenantToken(ctx, tenantA, "tenant-a", false, nil)
	require.NoError(t, err)

	gotTenant, _, err := mgr.ValidateTenantToken(ctx, tokenA.Token)
	require.NoError(t, err)
	assert.Equal(t, tenantA, gotTenant)

	resignedWithTenantB := resignTokenWithSubject(t, enc, opts, tokenA, tenantB)

	_, _, err = mgr.ValidateTenantToken(ctx, resignedWithTenantB)
	assert.Error(t, err)
}

func newLocalEncryption(t *testing.T) encryption.EncryptionService {
	t.Helper()

	master, priv, pub, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	enc, err := encryption.NewLocalEncryption(master, priv, pub)
	require.NoError(t, err)

	return enc
}

func resignTokenWithSubject(
	t *testing.T,
	enc encryption.EncryptionService,
	opts *token.TokenOpts,
	tok *token.Token,
	subject uuid.UUID,
) string {
	t.Helper()

	signer, err := jwt.NewSigner(enc.GetPrivateJWTHandle())
	require.NoError(t, err)

	issuedAt := time.Now().Add(-time.Minute)
	subjectStr := subject.String()

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &issuedAt,
		Audience:  &opts.Audience,
		Subject:   &subjectStr,
		ExpiresAt: &tok.ExpiresAt,
		Issuer:    &opts.Issuer,
		CustomClaims: map[string]interface{}{
			"token_id":   tok.TokenId.String(),
			"server_url": opts.ServerURL,
		},
	})
	require.NoError(t, err)

	signed, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	return signed
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
