package exchangetoken_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"

	"github.com/hatchet-dev/hatchet/pkg/auth/exchangetoken"
)

const (
	testSupportIssuer   = "hatchet-control-plane"
	testSupportAudience = "hatchet-support"
)

type supportTokenOpts struct {
	tenantID  uuid.UUID
	userID    uuid.UUID
	grantID   uuid.UUID
	role      string
	email     string
	name      string
	issuer    string
	audience  string
	expiresAt time.Time
	omit      map[string]bool
}

func newSupportTokenClient(t *testing.T, public *keyset.Handle) exchangetoken.SupportTokenClient {
	t.Helper()

	client, err := exchangetoken.NewSupportTokenClient(public, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testSupportIssuer,
		Audience: testSupportAudience,
	})
	require.NoError(t, err)

	return client
}

func newSignedSupportToken(t *testing.T, private *keyset.Handle, o supportTokenOpts) string {
	t.Helper()

	signer, err := jwt.NewSigner(private)
	require.NoError(t, err)

	issuedAt := time.Now().Add(-1 * time.Minute)
	subject := o.userID.String()

	custom := map[string]interface{}{}

	if !o.omit[exchangetoken.SupportTenantClaim] {
		custom[exchangetoken.SupportTenantClaim] = o.tenantID.String()
	}

	if !o.omit[exchangetoken.SupportGrantIDClaim] {
		custom[exchangetoken.SupportGrantIDClaim] = o.grantID.String()
	}

	if !o.omit[exchangetoken.SupportRoleClaim] {
		custom[exchangetoken.SupportRoleClaim] = o.role
	}

	if !o.omit[exchangetoken.SupportEmailClaim] {
		custom[exchangetoken.SupportEmailClaim] = o.email
	}

	if o.name != "" {
		custom[exchangetoken.SupportNameClaim] = o.name
	}

	raw, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		Issuer:       &o.issuer,
		Subject:      &subject,
		Audience:     &o.audience,
		IssuedAt:     &issuedAt,
		ExpiresAt:    &o.expiresAt,
		CustomClaims: custom,
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(raw)
	require.NoError(t, err)

	return token
}

func validSupportTokenOpts() supportTokenOpts {
	return supportTokenOpts{
		tenantID:  uuid.New(),
		userID:    uuid.New(),
		grantID:   uuid.New(),
		role:      string(exchangetoken.SupportRoleViewer),
		email:     "jb@hatchet.run",
		name:      "JB",
		issuer:    testSupportIssuer,
		audience:  testSupportAudience,
		expiresAt: time.Now().Add(15 * time.Minute),
		omit:      map[string]bool{},
	}
}

func TestValidateSupportToken(t *testing.T) {
	private, public := newKeyPair(t)
	client := newSupportTokenClient(t, public)

	for _, role := range []exchangetoken.SupportRole{exchangetoken.SupportRoleViewer, exchangetoken.SupportRoleOperator} {
		o := validSupportTokenOpts()
		o.role = string(role)

		claims, err := client.ValidateSupportToken(context.Background(), newSignedSupportToken(t, private, o))
		require.NoError(t, err)

		assert.Equal(t, o.tenantID, claims.TenantID)
		assert.Equal(t, o.userID, claims.UserID)
		assert.Equal(t, o.grantID, claims.GrantID)
		assert.Equal(t, role, claims.Role)
		assert.Equal(t, "jb@hatchet.run", claims.Email)
		assert.Equal(t, "JB", claims.Name)
		assert.WithinDuration(t, o.expiresAt, claims.ExpiresAt, time.Second)
	}
}

func TestValidateSupportToken_Rejections(t *testing.T) {
	private, public := newKeyPair(t)
	otherPrivate, _ := newKeyPair(t)
	client := newSupportTokenClient(t, public)

	cases := map[string]func(o *supportTokenOpts) *keyset.Handle{
		"expired":        func(o *supportTokenOpts) *keyset.Handle { o.expiresAt = time.Now().Add(-time.Minute); return private },
		"wrong audience": func(o *supportTokenOpts) *keyset.Handle { o.audience = "hatchet"; return private },
		"wrong issuer":   func(o *supportTokenOpts) *keyset.Handle { o.issuer = "someone-else"; return private },
		"wrong key":      func(o *supportTokenOpts) *keyset.Handle { return otherPrivate },
		"unknown role":   func(o *supportTokenOpts) *keyset.Handle { o.role = "OWNER"; return private },
		"empty role":     func(o *supportTokenOpts) *keyset.Handle { o.role = ""; return private },
		"missing role": func(o *supportTokenOpts) *keyset.Handle {
			o.omit[exchangetoken.SupportRoleClaim] = true
			return private
		},
		"missing grant id": func(o *supportTokenOpts) *keyset.Handle {
			o.omit[exchangetoken.SupportGrantIDClaim] = true
			return private
		},
		"missing tenant id": func(o *supportTokenOpts) *keyset.Handle {
			o.omit[exchangetoken.SupportTenantClaim] = true
			return private
		},
		"malformed tenant id": func(o *supportTokenOpts) *keyset.Handle { o.tenantID = uuid.Nil; return private },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			o := validSupportTokenOpts()
			key := mutate(&o)

			token := newSignedSupportToken(t, private, o)
			if key != private {
				token = newSignedSupportToken(t, key, o)
			}

			if name == "malformed tenant id" {
				// uuid.Nil parses; authn rejects it against the queried tenant
				claims, err := client.ValidateSupportToken(context.Background(), token)
				require.NoError(t, err)
				assert.Equal(t, uuid.Nil, claims.TenantID)
				return
			}

			_, err := client.ValidateSupportToken(context.Background(), token)
			assert.Error(t, err)
		})
	}
}

// neither token kind validates as the other
func TestSupportAndExchangeTokensDoNotCross(t *testing.T) {
	supportPrivate, supportPublic := newKeyPair(t)
	exchangePrivate, exchangePublic := newKeyPair(t)

	supportClient := newSupportTokenClient(t, supportPublic)

	exchangeClient, err := exchangetoken.NewExchangeTokenClient(exchangePublic, &exchangetoken.ExchangeTokenOpts{
		Issuer:   testIssuer,
		Audience: testAudience,
	})
	require.NoError(t, err)

	tenantID := uuid.New()
	userID := uuid.New()

	exchangeToken := newSignedToken(t, exchangePrivate, tenantID, userID, testIssuer, testAudience, time.Now().Add(time.Hour))
	_, err = supportClient.ValidateSupportToken(context.Background(), exchangeToken)
	assert.Error(t, err, "an exchange token must not be accepted as a support token")

	exchangeShapedToken := newSignedToken(t, supportPrivate, tenantID, userID, testSupportIssuer, testSupportAudience, time.Now().Add(time.Hour))
	_, err = supportClient.ValidateSupportToken(context.Background(), exchangeShapedToken)
	assert.Error(t, err)

	o := validSupportTokenOpts()
	supportToken := newSignedSupportToken(t, supportPrivate, o)
	_, _, err = exchangeClient.ValidateExchangeToken(context.Background(), supportToken)
	assert.Error(t, err, "a support token must not be accepted as an exchange token")
}

func TestIsSupportRole(t *testing.T) {
	assert.True(t, exchangetoken.IsSupportRole("SUPPORT_VIEWER"))
	assert.True(t, exchangetoken.IsSupportRole("SUPPORT_OPERATOR"))
	assert.False(t, exchangetoken.IsSupportRole("OWNER"))
	assert.False(t, exchangetoken.IsSupportRole("ADMIN"))
	assert.False(t, exchangetoken.IsSupportRole("MEMBER"))
	assert.False(t, exchangetoken.IsSupportRole("VIEWER"))
	assert.False(t, exchangetoken.IsSupportRole(""))
	assert.False(t, exchangetoken.IsSupportRole("support_viewer"))
}
