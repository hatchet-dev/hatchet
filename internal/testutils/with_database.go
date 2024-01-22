package testutils

import (
	"testing"

	"github.com/hatchet-dev/hatchet/internal/config/database"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
)

func RunTestWithDatabase(t *testing.T, test func(config *database.Config) error) {
	t.Helper()
	Prepare(t)

	confLoader := &loader.ConfigLoader{}

	conf, err := confLoader.LoadDatabaseConfig()
	if err != nil {
		t.Fatalf("failed to load database config: %v\n", err)
	}
	defer conf.Disconnect() // nolint: errcheck

	err = test(conf)

	if err != nil {
		t.Fatalf("%v\n", err)
	}
}
