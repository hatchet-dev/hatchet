//go:build pgqueue

package test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestPgQueueEnsure(t *testing.T) {
	// read in the local config
	configLoader := loader.NewConfigLoader("../../../generated/")

	wg := sync.WaitGroup{}
	wg.Add(1)

	cleanup, _, err := configLoader.LoadServerConfig("", func(scf *server.ServerConfigFile) {
		assert.Equal(t, "postgres", scf.MessageQueue.Kind)
		wg.Done()
	})
	if err != nil {
		// ignore config error
		return
	}
	defer cleanup() // nolint:errcheck
	wg.Wait()
}
