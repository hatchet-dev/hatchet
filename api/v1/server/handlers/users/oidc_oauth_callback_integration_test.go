//go:build integration

package users

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

// newOIDCTestConfig builds a ServerConfig wired to the in-process mock issuer +
// the real database layer. SetEmailVerified defaults to false (the production
// default), so unverified emails are created but left unverified.
func newOIDCTestConfig(t *testing.T, dbConf *database.Layer) (*server.ServerConfig, *mockOIDC) {
	t.Helper()

	masterKey, privJWT, pubJWT, _, err := encryption.GenerateLocalKeys()
	if err != nil {
		t.Fatalf("generate local keys: %v", err)
	}
	enc, err := encryption.NewLocalEncryption(masterKey, privJWT, pubJWT)
	if err != nil {
		t.Fatalf("new local encryption: %v", err)
	}

	m := newMockOIDC(t)
	logger := zerolog.Nop()
	cfg := &server.ServerConfig{
		Layer:      dbConf,
		Encryption: enc,
		Logger:     &logger,
		Runtime:    server.ConfigFileRuntime{AllowSignup: true, ServerURL: "http://localhost:8080"},
		Auth: server.AuthConfig{
			OIDCProvider:    m.provider,
			OIDCOAuthConfig: m.oauthCfg,
		},
	}
	return cfg, m
}

func uniqueEmail(t *testing.T, prefix string) string {
	t.Helper()
	suffix, err := random.Generate(8)
	if err != nil {
		t.Fatalf("random suffix: %v", err)
	}
	// CreateUser stores emails lowercased, so use a lowercase address.
	return strings.ToLower(prefix + "-" + suffix + "@example.com")
}

// TestUpsertOIDCUserFromToken exercises the full OIDC upsert against a real
// database: verify the ID token, then create (and on a second login, update) the
// Hatchet user + OAuth link. This is the path where OAuthOpts.Provider="oidc"
// must pass repository validation — a regression here is exactly the bug that
// shipped in the original PR (provider "oidc" rejected by the oneof validator).
func TestUpsertOIDCUserFromToken(t *testing.T) {
	// InitDataLayer requires a message-queue URL to be present (it is not
	// connected to in this DB-only test). Mirrors the other integration tests.
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, m := newOIDCTestConfig(t, dbConf)
		us := NewUserService(cfg)
		ctx := context.Background()

		email := uniqueEmail(t, "oidc")

		// First login: user does not exist yet -> CreateUser with provider="oidc".
		tok := m.token(t, idTokenClaims{
			Subject: "kc-sub-abc", Email: email, EmailVerified: true, Name: "Alice Example",
		})
		user, err := us.upsertOIDCUserFromToken(ctx, cfg, tok)
		if err != nil {
			t.Fatalf("create via OIDC failed: %v", err)
		}
		if user.Email != email {
			t.Fatalf("created user email = %q, want %q", user.Email, email)
		}
		if !user.EmailVerified {
			t.Fatal("created user should be email-verified")
		}

		// Second login for the same email: should update the existing user, not
		// create a duplicate.
		tok2 := m.token(t, idTokenClaims{
			Subject: "kc-sub-abc", Email: email, EmailVerified: true, Name: "Alice Renamed",
		})
		user2, err := us.upsertOIDCUserFromToken(ctx, cfg, tok2)
		if err != nil {
			t.Fatalf("update via OIDC failed: %v", err)
		}
		if user2.ID != user.ID {
			t.Fatalf("second login created a new user (%s) instead of updating (%s)", user2.ID, user.ID)
		}

		return nil
	})
}

// TestUpsertOIDCUserFromToken_UnverifiedEmail covers a provider that does not
// assert a verified email (e.g. Microsoft Entra ID). The login is never rejected
// (matching Google/GitHub); SetEmailVerified controls whether the stored user is
// verified or left for the application's verify-email gate.
func TestUpsertOIDCUserFromToken_UnverifiedEmail(t *testing.T) {
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, m := newOIDCTestConfig(t, dbConf)
		us := NewUserService(cfg)
		ctx := context.Background()

		// SetEmailVerified=false (default): the user is created but left
		// unverified (the verify-email gate then applies) — not rejected.
		gatedEmail := uniqueEmail(t, "entra-gated")
		gatedTok := m.token(t, idTokenClaims{
			Subject: "entra-sub-1", Email: gatedEmail, EmailVerified: false, Name: "Bob Example",
		})
		gated, err := us.upsertOIDCUserFromToken(ctx, cfg, gatedTok)
		if err != nil {
			t.Fatalf("unverified email should be created, not rejected: %v", err)
		}
		if gated.EmailVerified {
			t.Fatal("with SetEmailVerified=false an unverified email should stay unverified")
		}

		// SetEmailVerified=true: the same flow auto-verifies the user.
		cfg.Auth.ConfigFile.SetEmailVerified = true
		autoEmail := uniqueEmail(t, "entra-auto")
		autoTok := m.token(t, idTokenClaims{
			Subject: "entra-sub-2", Email: autoEmail, EmailVerified: false, Name: "Carol Example",
		})
		auto, err := us.upsertOIDCUserFromToken(ctx, cfg, autoTok)
		if err != nil {
			t.Fatalf("upsert with SetEmailVerified=true failed: %v", err)
		}
		if !auto.EmailVerified {
			t.Fatal("with SetEmailVerified=true the user should be stored as verified")
		}

		return nil
	})
}
