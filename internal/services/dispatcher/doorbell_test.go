package dispatcher

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

func TestDoorbellClaimDropsAssignmentsAlreadyHandled(t *testing.T) {
	b := newDoorbell(DoorbellOpts{PullInterval: time.Second, PullGrace: time.Second})
	workerId := uuid.New()

	first := &tasktypesv1.TaskAssignedBulkTaskPayload{}
	first.Add(workerId, 1, 0)
	b.claim(first)
	assert.Equal(t, []int64{1}, first.WorkerIdToTaskIds[workerId])

	second := &tasktypesv1.TaskAssignedBulkTaskPayload{}
	second.Add(workerId, 1, 0)
	second.Add(workerId, 2, 0)
	b.claim(second)
	assert.Equal(t, []int64{2}, second.WorkerIdToTaskIds[workerId])

	retried := &tasktypesv1.TaskAssignedBulkTaskPayload{}
	retried.Add(workerId, 1, 1)
	b.claim(retried)
	assert.Equal(t, []int64{1}, retried.WorkerIdToTaskIds[workerId], "a new retry is a new assignment")

	duplicate := &tasktypesv1.TaskAssignedBulkTaskPayload{}
	duplicate.Add(workerId, 2, 0)
	b.claim(duplicate)
	assert.Empty(t, duplicate.WorkerIdToTaskIds)
}

func TestDoorbellClaimKeepsAssignmentsWithoutRetryCounts(t *testing.T) {
	b := newDoorbell(DoorbellOpts{})
	workerId := uuid.New()

	for range 2 {
		legacy := &tasktypesv1.TaskAssignedBulkTaskPayload{WorkerIdToTaskIds: map[uuid.UUID][]int64{workerId: {7}}}
		b.claim(legacy)
		assert.Equal(t, []int64{7}, legacy.WorkerIdToTaskIds[workerId])
	}
}
