package testutils

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func RunTestWithDatabase(t *testing.T, test func(config *database.Config) error) {
	t.Helper()
	Prepare(t)

	confLoader := &loader.ConfigLoader{}

	cleanup, conf, err := confLoader.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("failed to load database config: %v\n", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer cleanup() // nolint:errcheck

	err = test(conf)

	if err != nil {
		t.Fatalf("%v\n", err)
	}
}
