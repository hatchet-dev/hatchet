//go:build !e2e && !load && !rampup && !integration

package serverlessoperator

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
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

	tl.stop()

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
	tl.stop()
}
