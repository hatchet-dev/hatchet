//go:build !e2e && !load && !rampup && !integration

package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// Zero slots is the kill switch: no writer means the dispatcher sends every worker event over the
// task queue, and no second task controller gets built. The config has no repository or queue, so
// this also fails if anything is constructed before the check.
func TestNewDirectTaskWriterIsOffAtZeroSlots(t *testing.T) {
	w, err := newDirectTaskWriter(&server.ServerConfig{
		Runtime: server.ConfigFileRuntime{DirectTaskWriteSlots: 0},
	})

	require.NoError(t, err)
	assert.Nil(t, w)
}
