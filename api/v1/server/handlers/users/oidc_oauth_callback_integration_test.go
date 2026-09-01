//go:build integration

package users

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

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

func TestUpsertOIDCUserFromToken(t *testing.T) {
	// Database initialization validates the queue URL without connecting to it.
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, m := newOIDCTestConfig(t, dbConf)
		us := NewUserService(cfg)
		ctx := context.Background()

		email := uniqueEmail(t, "oidc")
		subject := uuid.NewString()
		emptyGroups := []string{}

		tok := m.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Alice Example", Groups: &emptyGroups,
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

		tok2 := m.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Alice Renamed", Groups: &emptyGroups,
		})
		user2, err := us.upsertOIDCUserFromToken(ctx, cfg, tok2)
		if err != nil {
			t.Fatalf("update via OIDC failed: %v", err)
		}
		if user2.ID != user.ID {
			t.Fatalf("second login created a new user (%s) instead of updating (%s)", user2.ID, user.ID)
		}

		_, err = us.upsertOIDCUserFromToken(ctx, cfg, m.token(t, idTokenClaims{
			Subject: uuid.NewString(), Email: email, EmailVerified: true, Name: "Alice Renamed", Groups: &emptyGroups,
		}))
		if err == nil {
			t.Fatal("expected a different OIDC subject for the same account to be rejected")
		}

		_, err = us.upsertOIDCUserFromToken(ctx, cfg, m.token(t, idTokenClaims{
			Subject: subject, Email: uniqueEmail(t, "oidc-duplicate"), EmailVerified: true, Name: "Alice Duplicate", Groups: &emptyGroups,
		}))
		if err == nil {
			t.Fatal("expected the same OIDC subject on a second account to be rejected")
		}

		return nil
	})
}

func TestOIDCExistingAccountLinking(t *testing.T) {
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, issuer := newOIDCTestConfig(t, dbConf)
		service := NewUserService(cfg)
		ctx := context.Background()
		queries := sqlcv1.New()
		emptyGroups := []string{}

		unverifiedEmail := uniqueEmail(t, "oidc-unverified-link")
		unverifiedUser, err := queries.CreateUser(ctx, dbConf.Pool, sqlcv1.CreateUserParams{
			ID: uuid.New(), Email: unverifiedEmail, EmailVerified: pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			return err
		}
		_, err = service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: uuid.NewString(), Email: unverifiedEmail, EmailVerified: false, Groups: &emptyGroups,
		}))
		if err == nil {
			t.Fatal("expected an unverified email not to link an existing account")
		}
		if _, err := cfg.V1.User().GetUserOAuth(ctx, unverifiedUser.ID, "oidc"); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("unexpected OIDC link after rejected login: %v", err)
		}

		verifiedEmail := uniqueEmail(t, "oidc-cross-provider")
		verifiedUser, err := queries.CreateUser(ctx, dbConf.Pool, sqlcv1.CreateUserParams{
			ID: uuid.New(), Email: verifiedEmail, EmailVerified: pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			return err
		}
		_, err = queries.CreateUserOAuth(ctx, dbConf.Pool, sqlcv1.CreateUserOAuthParams{
			Userid: verifiedUser.ID, Provider: "google", Provideruserid: uuid.NewString(), Accesstoken: []byte("access"),
		})
		if err != nil {
			return err
		}
		linked, err := service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: uuid.NewString(), Email: verifiedEmail, EmailVerified: true, Groups: &emptyGroups,
		}))
		if err != nil {
			t.Fatalf("link OIDC alongside existing provider: %v", err)
		}
		if linked.ID != verifiedUser.ID {
			t.Fatalf("linked user = %s, want %s", linked.ID, verifiedUser.ID)
		}
		if _, err := cfg.V1.User().GetUserOAuth(ctx, verifiedUser.ID, "google"); err != nil {
			t.Fatalf("existing provider link was lost: %v", err)
		}
		if _, err := cfg.V1.User().GetUserOAuth(ctx, verifiedUser.ID, "oidc"); err != nil {
			t.Fatalf("OIDC provider link was not created: %v", err)
		}

		return nil
	})
}

func TestOIDCGroupMembershipReconciliationPreservesManualMembership(t *testing.T) {
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, issuer := newOIDCTestConfig(t, dbConf)
		service := NewUserService(cfg)
		ctx := context.Background()
		queries := sqlcv1.New()

		managedTenantID := uuid.New()
		manualTenantID := uuid.New()
		syncedTenantID := uuid.New()
		for tenantID, name := range map[uuid.UUID]string{
			managedTenantID: "OIDC managed tenant",
			manualTenantID:  "Manually managed tenant",
			syncedTenantID:  "Synchronized tenant",
		} {
			if _, err := dbConf.Pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, name, tenantID.String()); err != nil {
				return err
			}
		}

		email := uniqueEmail(t, "oidc-groups")
		subject := uuid.NewString()
		emptyGroups := []string{}
		user, err := service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Group User", Groups: &emptyGroups,
		}))
		if err != nil {
			return err
		}

		if _, err := queries.CreateTenantMember(ctx, dbConf.Pool, sqlcv1.CreateTenantMemberParams{
			Tenantid: manualTenantID, Userid: user.ID, Role: sqlcv1.TenantMemberRoleVIEWER,
		}); err != nil {
			return err
		}

		cfg.Auth.ConfigFile.OIDC.IssuerURL = issuer.server.URL
		cfg.Auth.OIDCGroupMappings = map[string]string{"platform-admins": "ADMIN"}
		adminGroups := []string{"platform-admins"}
		_, err = service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Group User", Groups: &adminGroups,
		}))
		if err != nil {
			return err
		}

		managed, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: managedTenantID, Userid: user.ID,
		})
		if err != nil {
			return err
		}
		if managed.Role != sqlcv1.TenantMemberRoleADMIN || !managed.OidcIssuer.Valid {
			t.Fatalf("managed membership = %+v", managed)
		}

		manual, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: manualTenantID, Userid: user.ID,
		})
		if err != nil {
			return err
		}
		if manual.Role != sqlcv1.TenantMemberRoleVIEWER || manual.OidcIssuer.Valid {
			t.Fatalf("manual membership was overwritten: %+v", manual)
		}

		synced, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: syncedTenantID, Userid: user.ID,
		})
		if err != nil {
			return err
		}
		synced, err = queries.SyncUpdateTenantMember(ctx, dbConf.Pool, sqlcv1.SyncUpdateTenantMemberParams{
			ID: synced.ID, Updatedat: synced.UpdatedAt, Role: sqlcv1.TenantMemberRoleVIEWER, CanViewPayloads: pgtype.Bool{},
		})
		if err != nil {
			return err
		}
		if synced.Role != sqlcv1.TenantMemberRoleVIEWER || synced.OidcIssuer.Valid {
			t.Fatalf("synchronized membership retained OIDC provenance: %+v", synced)
		}

		_, err = service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Group User", Groups: &emptyGroups,
		}))
		if err != nil {
			return err
		}
		if _, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: managedTenantID, Userid: user.ID,
		}); err != pgx.ErrNoRows {
			t.Fatalf("managed membership should be removed, got %v", err)
		}
		if _, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: manualTenantID, Userid: user.ID,
		}); err != nil {
			t.Fatalf("manual membership should remain: %v", err)
		}
		if _, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: syncedTenantID, Userid: user.ID,
		}); err != nil {
			t.Fatalf("synchronized membership should remain: %v", err)
		}

		return nil
	})
}

func TestDatabaseOIDCGroupMembershipReconciliation(t *testing.T) {
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, issuer := newOIDCTestConfig(t, dbConf)
		service := NewUserService(cfg)
		ctx := context.Background()
		queries := sqlcv1.New()
		tenantID := uuid.New()
		if _, err := dbConf.Pool.Exec(ctx, `INSERT INTO "Tenant" ("id", "name", "slug") VALUES ($1, $2, $3)`, tenantID, "Database mapped tenant", tenantID.String()); err != nil {
			return err
		}
		defer dbConf.Pool.Exec(ctx, `DELETE FROM "Tenant" WHERE "id" = $1`, tenantID) //nolint:errcheck
		if _, err := queries.UpsertTenantOIDCGroupMapping(ctx, dbConf.Pool, sqlcv1.UpsertTenantOIDCGroupMappingParams{
			Tenantid: tenantID, Groupname: "database-members", Role: sqlcv1.TenantMemberRoleMEMBER,
		}); err != nil {
			return err
		}

		cfg.Auth.ConfigFile.OIDC.IssuerURL = issuer.server.URL
		groups := []string{"database-members"}
		email := uniqueEmail(t, "oidc-database-group")
		subject := uuid.NewString()
		user, err := service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Database Group User", Groups: &groups,
		}))
		if err != nil {
			return err
		}
		member, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: tenantID, Userid: user.ID,
		})
		if err != nil {
			return err
		}
		if member.Role != sqlcv1.TenantMemberRoleMEMBER || !member.OidcIssuer.Valid {
			t.Fatalf("database mapping membership = %+v", member)
		}

		if _, err := service.upsertOIDCUserFromToken(ctx, cfg, issuer.token(t, idTokenClaims{
			Subject: subject, Email: email, EmailVerified: true, Name: "Database Group User",
		})); err == nil {
			t.Fatal("expected login without an authoritative groups claim to fail")
		}
		if _, err := queries.GetTenantMemberByUserID(ctx, dbConf.Pool, sqlcv1.GetTenantMemberByUserIDParams{
			Tenantid: tenantID, Userid: user.ID,
		}); err != nil {
			t.Fatalf("membership should survive unavailable groups: %v", err)
		}
		return nil
	})
}

// New accounts may remain unverified; only linking to an existing account requires verified ownership.
func TestUpsertOIDCUserFromToken_UnverifiedEmail(t *testing.T) {
	_ = os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(dbConf *database.Layer) error {
		cfg, m := newOIDCTestConfig(t, dbConf)
		us := NewUserService(cfg)
		ctx := context.Background()

		gatedEmail := uniqueEmail(t, "entra-gated")
		emptyGroups := []string{}
		gatedTok := m.token(t, idTokenClaims{
			Subject: uuid.NewString(), Email: gatedEmail, EmailVerified: false, Name: "Bob Example", Groups: &emptyGroups,
		})
		gated, err := us.upsertOIDCUserFromToken(ctx, cfg, gatedTok)
		if err != nil {
			t.Fatalf("unverified email should be created, not rejected: %v", err)
		}
		if gated.EmailVerified {
			t.Fatal("with SetEmailVerified=false an unverified email should stay unverified")
		}

		cfg.Auth.ConfigFile.SetEmailVerified = true
		autoEmail := uniqueEmail(t, "entra-auto")
		autoTok := m.token(t, idTokenClaims{
			Subject: uuid.NewString(), Email: autoEmail, EmailVerified: false, Name: "Carol Example", Groups: &emptyGroups,
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
