package scheduler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

func TestTaskBulkAssignedTaskCarriesRetryCounts(t *testing.T) {
	tenantId := uuid.New()
	worker1, worker2 := uuid.New(), uuid.New()

	assignments := newWorkerAssignments()
	assignments.add(worker1, 10, 0)
	assignments.add(worker1, 11, 2)
	assignments.add(worker2, 12, 1)

	msg, err := taskBulkAssignedTask(tenantId, assignments)
	require.NoError(t, err)
	assert.Equal(t, msgqueue.MsgIDTaskAssignedBulk, msg.ID)

	payloads := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)
	require.Len(t, payloads, 1)

	payload := payloads[0]
	assert.ElementsMatch(t, []int64{10, 11}, payload.WorkerIdToTaskIds[worker1])
	assert.ElementsMatch(t, []int64{12}, payload.WorkerIdToTaskIds[worker2])

	assert.Equal(t, map[uuid.UUID]map[int64]int32{
		worker1: {10: 0, 11: 2},
		worker2: {12: 1},
	}, payload.WorkerIdToTaskRetryCounts)
}

func TestTaskBulkAssignedTaskRetryCountsCoverEveryTask(t *testing.T) {
	assignments := newWorkerAssignments()
	workerId := uuid.New()

	for taskId := int64(0); taskId < 5; taskId++ {
		assignments.add(workerId, taskId, int32(taskId%2))
	}

	msg, err := taskBulkAssignedTask(uuid.New(), assignments)
	require.NoError(t, err)

	payload := msgqueue.JSONConvert[tasktypes.TaskAssignedBulkTaskPayload](msg.Payloads)[0]

	for _, taskId := range payload.WorkerIdToTaskIds[workerId] {
		retryCount, ok := payload.WorkerIdToTaskRetryCounts[workerId][taskId]
		assert.True(t, ok, "task %d has no retry count in the message", taskId)
		assert.Equal(t, int32(taskId%2), retryCount)
	}
}
