package dispatcher

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

func TestPubSubDeliveryClaimDropsAssignmentsAlreadyHandled(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{PullInterval: time.Second, PullGrace: time.Second})
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

func TestPubSubDeliveryClaimKeepsAssignmentsWithoutRetryCounts(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{})
	workerId := uuid.New()

	for range 2 {
		legacy := &tasktypesv1.TaskAssignedBulkTaskPayload{WorkerIdToTaskIds: map[uuid.UUID][]int64{workerId: {7}}}
		b.claim(legacy)
		assert.Equal(t, []int64{7}, legacy.WorkerIdToTaskIds[workerId])
	}
}

func TestPubSubDeliveryClaimOneBlocksLaterPulls(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{})
	key := assignmentKey{taskId: 3, retryCount: 0}

	assert.True(t, b.claimOne(key), "a local assignment is claimed the first time")
	assert.True(t, b.seen(key), "the pull skips an assignment delivered locally")
	assert.False(t, b.claimOne(key))

	workerId := uuid.New()
	fromPubSub := &tasktypesv1.TaskAssignedBulkTaskPayload{}
	fromPubSub.Add(workerId, 3, 0)
	b.claim(fromPubSub)
	assert.Empty(t, fromPubSub.WorkerIdToTaskIds, "a late pub/sub message doesn't resend a local assignment")
}
