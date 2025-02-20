package testutils

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func RunTestWithDatabase(t *testing.T, test func(config *database.Layer) error) {
	t.Helper()
	Prepare(t)

	confLoader := &loader.ConfigLoader{}

	conf, err := confLoader.InitDataLayer()
	if err != nil {
		t.Fatalf("failed to load database config: %v\n", err)
	}
	defer conf.Disconnect() // nolint: errcheck

	err = test(conf)

	if err != nil {
		t.Fatalf("%v\n", err)
	}
}
