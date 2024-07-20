//go:build pgqueue

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestPgQueueEnsure(t *testing.T) {
	// read in the local config
	configLoader := loader.NewConfigLoader("")

	cleanup, _, err := configLoader.LoadServerConfig("", func(scf *server.ServerConfigFile) {
		assert.Equal(t, scf.MessageQueue.Kind, "postgres")
	})
	if err != nil {
		t.Fatalf("could not load server config: %v", err)
	}

	defer cleanup() // nolint:errcheck
}
