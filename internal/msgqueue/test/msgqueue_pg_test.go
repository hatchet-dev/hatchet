//go:build pgqueue

package test

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestPgQueueEnsure(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	testPath := filepath.Dir(b)
	dir := path.Join(testPath, "../../..")

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
	_ = os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET_FILE", path.Join(dir, "hack/dev/encryption-keys/master.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE", path.Join(dir, "hack/dev/encryption-keys/private_ec256.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE", path.Join(dir, "hack/dev/encryption-keys/public_ec256.key"))

	_ = os.Setenv("SERVER_SECURITY_CHECK_ENABLED", "false")

	cf := loader.NewConfigLoader(path.Join(dir, "./generated/"))

	wg := sync.WaitGroup{}
	wg.Add(1)

	cleanup, sc, err := cf.LoadServerConfig("", func(scf *server.ServerConfigFile) {
		assert.Equal(t, "postgres", scf.MessageQueue.Kind)
		wg.Done()
	})
	if err != nil {
		t.Fatalf("error loading server config: %v", err)
	}
	defer cleanup()       // nolint:errcheck
	defer sc.Disconnect() // nolint:errcheck
	wg.Wait()
}
