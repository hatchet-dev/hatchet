package testutils

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func Prepare(t *testing.T) {
	t.Helper()

	_, b, _, _ := runtime.Caller(0)
	testPath := filepath.Dir(b)
	baseDir := "../.."

	tenantId := "707d0855-80ab-4e1f-a156-f1c4546cbf52"

	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", tenantId)
	_ = os.Setenv("DATABASE_URL", "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet")
	_ = os.Setenv("HATCHET_CLIENT_TLS_ROOT_CA_FILE", path.Join(testPath, baseDir, "hack/dev/certs/ca.cert"))
	_ = os.Setenv("HATCHET_CLIENT_TLS_SERVER_NAME", "cluster")
	_ = os.Setenv("SERVER_TLS_CERT_FILE", path.Join(testPath, baseDir, "hack/dev/certs/cluster.pem"))
	_ = os.Setenv("SERVER_TLS_KEY_FILE", path.Join(testPath, baseDir, "hack/dev/certs/cluster.key"))
	_ = os.Setenv("SERVER_TLS_ROOT_CA_FILE", path.Join(testPath, baseDir, "hack/dev/certs/ca.cert"))

	_ = os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/master.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/private_ec256.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/public_ec256.key"))

	_ = os.Setenv("SERVER_PORT", "8080")
	_ = os.Setenv("SERVER_URL", "http://localhost:8080")

	_ = os.Setenv("SERVER_AUTH_COOKIE_SECRETS", "something something")
	_ = os.Setenv("SERVER_AUTH_COOKIE_DOMAIN", "app.dev.hatchet-tools.com")
	_ = os.Setenv("SERVER_AUTH_COOKIE_INSECURE", "false")
	_ = os.Setenv("SERVER_AUTH_SET_EMAIL_VERIFIED", "true")

	_ = os.Setenv("SERVER_SECURITY_CHECK_ENABLED", "false")

	_ = os.Setenv("SERVER_LOGGER_LEVEL", "error")
	_ = os.Setenv("SERVER_LOGGER_FORMAT", "console")
	_ = os.Setenv("DATABASE_LOGGER_LEVEL", "error")
	_ = os.Setenv("DATABASE_LOGGER_FORMAT", "console")
	_ = os.Setenv("SERVER_ADDITIONAL_LOGGERS_QUEUE_LEVEL", "error")
	_ = os.Setenv("SERVER_ADDITIONAL_LOGGERS_QUEUE_FORMAT", "console")
	_ = os.Setenv("SERVER_ADDITIONAL_LOGGERS_PGXSTATS_LEVEL", "error")
	_ = os.Setenv("SERVER_ADDITIONAL_LOGGERS_PGXSTATS_FORMAT", "console")

	// read in the local config
	configLoader := loader.NewConfigLoader(path.Join(testPath, baseDir, "generated"))

	cleanup, server, err := configLoader.CreateServerFromConfig("", func(scf *server.ServerConfigFile) {
		// disable security checks since we're not running the server
		scf.SecurityCheck.Enabled = false
	})
	if err != nil {
		t.Fatalf("could not load server config: %v", err)
	}

	// check if tenant exists
	_, err = server.APIRepository.Tenant().GetTenantByID(tenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			_, err = server.APIRepository.Tenant().CreateTenant(&repository.CreateTenantOpts{
				ID:   &tenantId,
				Name: "test-tenant",
				Slug: "test-tenant",
			})
			if err != nil {
				t.Fatalf("could not create tenant: %v", err)
			}
		} else {
			t.Fatalf("could not get tenant: %v", err)
		}
	}

	defaultTok, err := server.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, "default", false, nil)
	if err != nil {
		t.Fatalf("could not generate default token: %v", err)
	}

	_ = os.Setenv("HATCHET_CLIENT_TOKEN", defaultTok.Token)

	if err := server.Disconnect(); err != nil {
		t.Fatalf("could not disconnect from server: %v", err)
	}

	if err := cleanup(); err != nil {
		t.Fatalf("could not cleanup server config: %v", err)
	}
}
