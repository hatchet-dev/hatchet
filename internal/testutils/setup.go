package testutils

import (
	"context"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func SetupEngine(ctx context.Context, t *testing.T) {
	t.Helper()

	_, b, _, _ := runtime.Caller(0)
	testPath := filepath.Dir(b)
	dir := path.Join(testPath, "../..")

	log.Printf("dir: %s", dir)

	_ = os.Setenv("DATABASE_URL", "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet")
	_ = os.Setenv("SERVER_TLS_CERT_FILE", path.Join(dir, "hack/dev/certs/cluster.pem"))
	_ = os.Setenv("SERVER_TLS_KEY_FILE", path.Join(dir, "hack/dev/certs/cluster.key"))
	_ = os.Setenv("SERVER_TLS_ROOT_CA_FILE", path.Join(dir, "hack/dev/certs/ca.cert"))
	_ = os.Setenv("SERVER_PORT", "8080")
	_ = os.Setenv("SERVER_URL", "http://localhost:8080")
	_ = os.Setenv("SERVER_AUTH_COOKIE_SECRETS", "something something")
	_ = os.Setenv("SERVER_AUTH_COOKIE_DOMAIN", "app.dev.hatchet-tools.com")
	_ = os.Setenv("SERVER_AUTH_COOKIE_INSECURE", "false")
	_ = os.Setenv("SERVER_AUTH_SET_EMAIL_VERIFIED", "true")

	_ = os.Setenv("SERVER_SECURITY_CHECK_ENABLED", "false")

	cf := loader.NewConfigLoader(path.Join(dir, "./generated/"))

	if err := engine.Run(ctx, cf, ""); err != nil {
		t.Fatalf("engine failure: %s", err.Error())
	}
}
