package nats

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests need no NATS server: the tlsRootCAFile/tlsEnabled consistency
// check runs before options are built, and nats.go's RootCAs option loads and
// parses the CA bundle while options are applied, so misconfiguration fails
// Connect with a descriptive error before any dial.

func TestNewPubSubTLSRootCAFileRequiresTLSEnabled(t *testing.T) {
	cleanup, ps, err := NewPubSub(
		WithPubSubURL("nats://127.0.0.1:4222"),
		WithPubSubTLSRootCAFile("/etc/hatchet/nats-ca.pem"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "tlsEnabled")
	assert.Nil(t, cleanup)
	assert.Nil(t, ps)
}

func TestNewPubSubTLSRootCAFileMissing(t *testing.T) {
	cleanup, ps, err := NewPubSub(
		WithPubSubURL("nats://127.0.0.1:4222"),
		WithPubSubTLSEnabled(true),
		WithPubSubTLSRootCAFile("/nonexistent/ca.pem"),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rootCA file")
	assert.Nil(t, cleanup)
	assert.Nil(t, ps)
}

func TestNewPubSubTLSRootCAFileNotPEM(t *testing.T) {
	caPath := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(caPath, []byte("not a pem certificate"), 0o600))

	cleanup, ps, err := NewPubSub(
		WithPubSubURL("nats://127.0.0.1:4222"),
		WithPubSubTLSEnabled(true),
		WithPubSubTLSRootCAFile(caPath),
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse root certificate")
	assert.Nil(t, cleanup)
	assert.Nil(t, ps)
}
