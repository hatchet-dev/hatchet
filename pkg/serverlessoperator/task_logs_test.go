//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/internal/memrepo"
)

func TestTaskLoggerNamesTheEndpoint(t *testing.T) {
	repo := memrepo.New()
	l := zerolog.Nop()
	tl := newTaskLogger(repo.TaskLogs(), &l)

	tenantA, tenantB := uuid.New(), uuid.New()
	epA := &cachedEndpoint{id: uuid.New(), tenantId: tenantA}
	epB := &cachedEndpoint{id: uuid.New(), tenantId: tenantB}
	taskA, taskB := uuid.New(), uuid.New()

	tl.delivered(epA, &endpointConfig{name: "orders"}, &contracts.AssignedAction{TaskRunExternalId: taskA.String(), RetryCount: 2})
	tl.delivered(epB, &endpointConfig{name: "billing"}, &contracts.AssignedAction{TaskRunExternalId: taskB.String()})
	// an id that is not a uuid has no task to log on
	tl.delivered(epA, &endpointConfig{name: "orders"}, &contracts.AssignedAction{TaskRunExternalId: "not-a-uuid"})

	tl.stop(context.Background())

	rows := repo.TaskLogRows()
	require.Len(t, rows, 2)

	byTask := map[uuid.UUID]memrepo.TaskLog{}

	for _, row := range rows {
		byTask[row.TaskExternalId] = row
	}

	a := byTask[taskA]
	assert.Equal(t, tenantA, a.TenantId)
	assert.Equal(t, int32(2), a.RetryCount)
	assert.Equal(t, "Delivered to serverless endpoint orders", a.Message)

	var metadata map[string]string
	require.NoError(t, json.Unmarshal(a.Metadata, &metadata))
	assert.Equal(t, epA.id.String(), metadata[taskLogEndpointIdKey])
	assert.Equal(t, "orders", metadata[taskLogEndpointNameKey])

	assert.Equal(t, tenantB, byTask[taskB].TenantId)
}

func TestTaskLoggerDisabled(t *testing.T) {
	l := zerolog.Nop()
	tl := newTaskLogger(nil, &l)

	assert.Nil(t, tl)

	// a disabled logger accepts every call
	tl.delivered(&cachedEndpoint{}, &endpointConfig{}, &contracts.AssignedAction{TaskRunExternalId: uuid.NewString()})
	tl.stop(context.Background())
}

// blockingTaskLogRepo holds every Insert until released, and counts the lines it saw.
type blockingTaskLogRepo struct {
	started chan struct{}
	release chan struct{}
	lines   atomic.Int64
}

func (b *blockingTaskLogRepo) Insert(ctx context.Context, _ uuid.UUID, lines []repository.ServerlessTaskLogOpts) error {
	b.lines.Add(int64(len(lines)))
	b.started <- struct{}{}

	select {
	case <-b.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Lines still queued when the shutdown deadline has passed are dropped rather than written
// one slow statement at a time.
func TestTaskLoggerDropsQueuedLinesPastShutdownDeadline(t *testing.T) {
	repo := &blockingTaskLogRepo{started: make(chan struct{}, 1), release: make(chan struct{})}
	l := zerolog.Nop()
	tl := newTaskLogger(repo, &l)

	ep := &cachedEndpoint{id: uuid.New(), tenantId: uuid.New()}
	cfg := &endpointConfig{name: "orders"}

	tl.delivered(ep, cfg, &contracts.AssignedAction{TaskRunExternalId: uuid.NewString()})
	<-repo.started // the writer is inside Insert with the first line

	tl.delivered(ep, cfg, &contracts.AssignedAction{TaskRunExternalId: uuid.NewString()})
	tl.delivered(ep, cfg, &contracts.AssignedAction{TaskRunExternalId: uuid.NewString()})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stopped := make(chan struct{})

	go func() {
		tl.stop(ctx)
		close(stopped)
	}()

	<-tl.stopCh // stop has asked the writer to finish
	close(repo.release)

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("stop did not return once the shutdown deadline had passed")
	}

	assert.Equal(t, int64(1), repo.lines.Load())
}
