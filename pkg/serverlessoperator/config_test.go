//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestConfigDurableInvocationTimeout: a durable invocation is bounded by a setting of its
// own, and a value that is not positive falls back to the default.
func TestConfigDurableInvocationTimeout(t *testing.T) {
	assert.Equal(t, 15*time.Minute, DefaultDurableInvocationTimeout)
	assert.Equal(t, DefaultDurableInvocationTimeout, DefaultConfig().DurableInvocationTimeout)
	assert.Equal(t, DefaultDurableInvocationTimeout, Config{}.withDefaults().DurableInvocationTimeout)
	assert.Equal(t, DefaultDurableInvocationTimeout, Config{DurableInvocationTimeout: -time.Second}.withDefaults().DurableInvocationTimeout)
	assert.Equal(t, time.Hour, Config{DurableInvocationTimeout: time.Hour}.withDefaults().DurableInvocationTimeout)
}
