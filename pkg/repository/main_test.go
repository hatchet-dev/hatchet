//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}

	if testContainer != nil {
		_ = testContainer.Terminate(context.Background())
	}

	if originalDatabaseURL != "" {
		_ = os.Setenv("DATABASE_URL", originalDatabaseURL)
	} else {
		_ = os.Unsetenv("DATABASE_URL")
	}

	os.Exit(code)
}
