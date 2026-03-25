package exchangetoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"

	"github.com/hatchet-dev/hatchet/pkg/auth/exchangetoken"
)

const (
	testIssuer   = "hatchet"
	testAudience = "hatchet"
)

func TestValidateExchangeToken(t *testing.T) {
	privateHandle, publicHandle := newKeyPair(t)

	tenantId := uuid.New()
	userId := uuid.New()

	token := newSignedToken(t, privateHandle, tenantId, userId, testIssuer, testAudience, time.Now().Add(1*time.Hour))

	client, err := exchangetoken.NewExchangeTokenClient(publicHandle, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testIssuer,
		Audience: testAudience,
	})
	assert.NoError(t, err)

	gotTenantId, gotUserId, err := client.ValidateExchangeToken(context.Background(), token)

	assert.NoError(t, err)
	assert.Equal(t, tenantId, gotTenantId)
	assert.Equal(t, userId, gotUserId)
}

func TestValidateExchangeToken_Expired(t *testing.T) {
	privateHandle, publicHandle := newKeyPair(t)

	tenantId := uuid.New()
	userId := uuid.New()

	token := newSignedToken(t, privateHandle, tenantId, userId, testIssuer, testAudience, time.Now().Add(-1*time.Hour))

	client, err := exchangetoken.NewExchangeTokenClient(publicHandle, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testIssuer,
		Audience: testAudience,
	})
	assert.NoError(t, err)

	_, _, err = client.ValidateExchangeToken(context.Background(), token)
	assert.Error(t, err)
}

func TestValidateExchangeToken_WrongAudience(t *testing.T) {
	privateHandle, publicHandle := newKeyPair(t)

	tenantId := uuid.New()
	userId := uuid.New()

	token := newSignedToken(t, privateHandle, tenantId, userId, testIssuer, "wrong-audience", time.Now().Add(1*time.Hour))

	client, err := exchangetoken.NewExchangeTokenClient(publicHandle, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testIssuer,
		Audience: testAudience,
	})
	assert.NoError(t, err)

	_, _, err = client.ValidateExchangeToken(context.Background(), token)
	assert.Error(t, err)
}

func TestValidateExchangeToken_WrongKey(t *testing.T) {
	_, publicHandle := newKeyPair(t)
	otherPrivateHandle, _ := newKeyPair(t)

	tenantId := uuid.New()
	userId := uuid.New()

	token := newSignedToken(t, otherPrivateHandle, tenantId, userId, testIssuer, testAudience, time.Now().Add(1*time.Hour))

	client, err := exchangetoken.NewExchangeTokenClient(publicHandle, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testIssuer,
		Audience: testAudience,
	})
	assert.NoError(t, err)

	_, _, err = client.ValidateExchangeToken(context.Background(), token)
	assert.Error(t, err)
}

func newKeyPair(t *testing.T) (private, public *keyset.Handle) {
	t.Helper()

	private, err := keyset.NewHandle(jwt.ES256Template())
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	public, err = private.Public()
	if err != nil {
		t.Fatalf("failed to get public handle: %v", err)
	}

	return private, public
}

func newSignedToken(t *testing.T, privateHandle *keyset.Handle, tenantId, userId uuid.UUID, issuer, audience string, expiresAt time.Time) string {
	t.Helper()

	signer, err := jwt.NewSigner(privateHandle)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	issuedAt := time.Now().Add(-1 * time.Minute)
	subjectStr := userId.String()

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		Issuer:    &issuer,
		Subject:   &subjectStr,
		Audience:  &audience,
		IssuedAt:  &issuedAt,
		ExpiresAt: &expiresAt,
		CustomClaims: map[string]interface{}{
			"tenant_id": tenantId.String(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create raw JWT: %v", err)
	}

	token, err := signer.SignAndEncode(rawJWT)
	if err != nil {
		t.Fatalf("failed to sign JWT: %v", err)
	}

	return token
}
