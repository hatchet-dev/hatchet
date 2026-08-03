package users

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"golang.org/x/oauth2"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

const testOIDCClientID = "hatchet-test"

// mockOIDC is an in-process OIDC issuer for tests: it serves the discovery
// document + JWKS and mints signed ID tokens, so the OIDC verification and user
// upsert paths can be exercised without a real IdP (Keycloak/Dex). This keeps
// the test deterministic and container-free, matching the project's preference
// for in-process integration tests.
type mockOIDC struct {
	server   *httptest.Server
	signer   jose.Signer
	provider *oidc.Provider
	oauthCfg *oauth2.Config
}

func newMockOIDC(t *testing.T) *mockOIDC {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	const keyID = "test-key-1"
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: jose.JSONWebKey{Key: priv, KeyID: keyID}},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key: priv.Public(), KeyID: keyID, Algorithm: "RS256", Use: "sig",
	}}}

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                srv.URL,
			"authorization_endpoint":                srv.URL + "/auth",
			"token_endpoint":                        srv.URL + "/token",
			"jwks_uri":                              srv.URL + "/keys",
			"userinfo_endpoint":                     srv.URL + "/userinfo",
			"id_token_signing_alg_values_supported": []string{"RS256"},
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	provider, err := oidc.NewProvider(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("oidc discovery: %v", err)
	}

	return &mockOIDC{
		server:   srv,
		signer:   signer,
		provider: provider,
		oauthCfg: &oauth2.Config{ClientID: testOIDCClientID},
	}
}

// idTokenClaims are the claims the mock signs into an ID token. Zero values for
// iss/aud/exp/iat are filled with sensible defaults in token().
type idTokenClaims struct {
	Issuer        string `json:"iss"`
	Subject       string `json:"sub"`
	Audience      string `json:"aud"`
	Expiry        int64  `json:"exp"`
	IssuedAt      int64  `json:"iat"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
}

// token mints an *oauth2.Token whose "id_token" extra is a signed JWT carrying
// the given claims (audience defaults to the test client id).
func (m *mockOIDC) token(t *testing.T, c idTokenClaims) *oauth2.Token {
	t.Helper()

	if c.Issuer == "" {
		c.Issuer = m.server.URL
	}
	if c.Audience == "" {
		c.Audience = testOIDCClientID
	}
	if c.Expiry == 0 {
		c.Expiry = time.Now().Add(time.Hour).Unix()
	}
	if c.IssuedAt == 0 {
		c.IssuedAt = time.Now().Unix()
	}

	payload, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	jws, err := m.signer.Sign(payload)
	if err != nil {
		t.Fatalf("sign id token: %v", err)
	}
	raw, err := jws.CompactSerialize()
	if err != nil {
		t.Fatalf("serialize id token: %v", err)
	}

	return (&oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}).
		WithExtra(map[string]any{"id_token": raw})
}

// TestGetOIDCClaimsFromToken covers the security-critical verification path:
// the ID token is checked against the issuer's JWKS and audience before any
// claims are trusted.
func TestGetOIDCClaimsFromToken(t *testing.T) {
	m := newMockOIDC(t)
	cfg := &server.ServerConfig{
		Auth: server.AuthConfig{
			OIDCProvider:    m.provider,
			OIDCOAuthConfig: m.oauthCfg,
		},
	}
	ctx := context.Background()

	t.Run("valid token returns verified claims", func(t *testing.T) {
		tok := m.token(t, idTokenClaims{
			Subject: "kc-sub-123", Email: "alice@example.com", EmailVerified: true, Name: "Alice",
		})

		claims, err := getOIDCClaimsFromToken(ctx, cfg, tok)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.Email != "alice@example.com" || claims.Sub != "kc-sub-123" ||
			!claims.EmailVerified || claims.Name != "Alice" {
			t.Fatalf("unexpected claims: %+v", claims)
		}
	})

	t.Run("token for a different audience is rejected", func(t *testing.T) {
		tok := m.token(t, idTokenClaims{
			Subject: "x", Audience: "some-other-client", Email: "a@b.c", EmailVerified: true,
		})
		if _, err := getOIDCClaimsFromToken(ctx, cfg, tok); err == nil {
			t.Fatal("expected verification error for wrong audience, got nil")
		}
	})

	t.Run("response without an id_token is rejected", func(t *testing.T) {
		tok := &oauth2.Token{AccessToken: "no-id-token"}
		if _, err := getOIDCClaimsFromToken(ctx, cfg, tok); err == nil {
			t.Fatal("expected error for missing id_token, got nil")
		}
	})
}
