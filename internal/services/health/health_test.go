package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type unhealthyRepository struct {
	v1.HealthRepository
}

func (unhealthyRepository) IsHealthy(context.Context) bool { return false }

type disconnectedQueue struct {
	msgqueue.MessageQueue
}

func (disconnectedQueue) IsReady() bool { return false }

func TestLivenessIgnoresDependencies(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	logger := zerolog.Nop()
	h := New(unhealthyRepository{}, disconnectedQueue{}, "test", &logger)

	cleanup, err := h.Start(port)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cleanup() })

	get := func(path string) int {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, path))
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.StatusCode
	}

	assert.Equal(t, http.StatusOK, get("/live"))
	assert.Equal(t, http.StatusServiceUnavailable, get("/ready"))
}
