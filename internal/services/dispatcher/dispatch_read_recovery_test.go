package dispatcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func runtimeFor(workerId uuid.UUID, retryCount int32) *sqlcv1.V1TaskRuntime {
	return &sqlcv1.V1TaskRuntime{WorkerID: &workerId, RetryCount: retryCount}
}

func TestAssignedAttemptsFromPayload(t *testing.T) {
	worker1, worker2 := uuid.New(), uuid.New()

	attempts := assignedAttemptsFromPayload(&tasktypesv1.TaskAssignedBulkTaskPayload{
		WorkerIdToTaskIds: map[uuid.UUID][]int64{
			worker1: {1, 2},
			worker2: {3},
		},
		WorkerIdToTaskRetryCounts: map[uuid.UUID]map[int64]int32{
			worker1: {1: 0, 2: 3},
		},
	})

	require.Len(t, attempts, 3)
	assert.Equal(t, assignedAttempt{WorkerId: worker1, RetryCount: 0, HasRetryCount: true}, attempts[1])
	assert.Equal(t, assignedAttempt{WorkerId: worker1, RetryCount: 3, HasRetryCount: true}, attempts[2])
	assert.Equal(t, assignedAttempt{WorkerId: worker2}, attempts[3], "a task without a retry count in the message has none")
}

func TestAssignedAttemptsFromPayloadWithoutRetryCounts(t *testing.T) {
	workerId := uuid.New()

	attempts := assignedAttemptsFromPayload(&tasktypesv1.TaskAssignedBulkTaskPayload{
		WorkerIdToTaskIds: map[uuid.UUID][]int64{workerId: {7}},
	})

	require.Len(t, attempts, 1)
	assert.Equal(t, assignedAttempt{WorkerId: workerId}, attempts[7])
	assert.False(t, attempts[7].matches(runtimeFor(workerId, 0)), "a message from a scheduler without retry counts never requeues")
}

func TestAssignedAttemptMatches(t *testing.T) {
	workerId, otherWorker := uuid.New(), uuid.New()
	attempt := assignedAttempt{WorkerId: workerId, RetryCount: 1, HasRetryCount: true}

	assert.True(t, attempt.matches(runtimeFor(workerId, 1)), "same worker and retry count")
	assert.False(t, attempt.matches(runtimeFor(workerId, 2)), "same worker, newer retry count")
	assert.False(t, attempt.matches(runtimeFor(workerId, 0)), "same worker, older retry count")
	assert.False(t, attempt.matches(runtimeFor(otherWorker, 1)), "same retry count on another worker")
	assert.False(t, attempt.matches(&sqlcv1.V1TaskRuntime{RetryCount: 1}), "runtime row without a worker")
	assert.False(t, attempt.matches(nil), "no runtime row")
}
