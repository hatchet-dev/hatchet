package hatchet

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkerLogger_ExplicitLoggerWins(t *testing.T) {
	configured := zerolog.New(&bytes.Buffer{}).Level(zerolog.WarnLevel)
	clientLogger := zerolog.New(&bytes.Buffer{}).Level(zerolog.DebugLevel)

	resolved := resolveWorkerLogger(&configured, &clientLogger)

	require.NotNil(t, resolved)
	assert.Same(t, &configured, resolved, "an explicit worker logger must be returned unchanged")
}

func TestResolveWorkerLogger_InheritsClientLogger(t *testing.T) {
	var buf bytes.Buffer
	clientLogger := zerolog.New(&buf).Level(zerolog.WarnLevel)

	resolved := resolveWorkerLogger(nil, &clientLogger)

	require.NotNil(t, resolved)

	// Inherits the client's level: debug output is suppressed.
	resolved.Debug().Msg("should be filtered")
	assert.Empty(t, buf.String(), "inherited logger must respect the client's log level")

	// Warn output is emitted and tagged with the worker service field.
	resolved.Warn().Msg("visible")
	out := buf.String()
	assert.Contains(t, out, `"message":"visible"`)
	assert.Contains(t, out, `"service":"worker"`)
}

func TestResolveWorkerLogger_NilClientLogger(t *testing.T) {
	assert.Nil(t, resolveWorkerLogger(nil, nil), "with no logger available, pkg/worker's standalone default applies")
}
