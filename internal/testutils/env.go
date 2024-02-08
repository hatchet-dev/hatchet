package testutils

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func Prepare(t *testing.T) {
	t.Helper()

	_, b, _, _ := runtime.Caller(0)
	testPath := filepath.Dir(b)

	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", "707d0855-80ab-4e1f-a156-f1c4546cbf52")
	_ = os.Setenv("DATABASE_URL", "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet")
	_ = os.Setenv("HATCHET_CLIENT_TLS_ROOT_CA_FILE", path.Join(testPath, "../..", "hack/dev/certs/ca.cert"))
	_ = os.Setenv("HATCHET_CLIENT_TLS_SERVER_NAME", "cluster")
}
