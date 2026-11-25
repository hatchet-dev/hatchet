package dispatcher

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func dispatchOf(assignedAt map[int64]pgtype.Timestamptz) *v1.TasksForDispatch {
	data := &v1.TasksForDispatch{Runtimes: map[int64]*sqlcv1.V1TaskRuntime{}}

	for taskId, at := range assignedAt {
		data.Tasks = append(data.Tasks, &sqlcv1.V1Task{ID: taskId})
		data.Runtimes[taskId] = &sqlcv1.V1TaskRuntime{TaskID: taskId, AssignedAt: at}
	}

	return data
}

func taskIds(data *v1.TasksForDispatch) []int64 {
	ids := make([]int64, 0, len(data.Tasks))

	for _, task := range data.Tasks {
		ids = append(ids, task.ID)
	}

	return ids
}

func TestPubSubDeliveryClaimDropsSecondDeliveryOfAnAssignment(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})
	assignedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	assert.Equal(t, []int64{1}, taskIds(b.claimDispatch(dispatchOf(map[int64]pgtype.Timestamptz{1: assignedAt}))))
	assert.Empty(t, taskIds(b.claimDispatch(dispatchOf(map[int64]pgtype.Timestamptz{1: assignedAt}))), "a late pub/sub message or a pull resending the same assignment")
}

// Restoring an evicted durable task assigns the same retry again; only assigned_at moves.
func TestPubSubDeliveryClaimSendsARestoredDurableTask(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})
	first := time.Now()
	key := assignmentKey{taskId: 1, retryCount: 0}

	assert.True(t, b.claim(key, first))
	assert.True(t, b.claim(key, first.Add(5*time.Second)), "restore")
	assert.False(t, b.claim(key, first.Add(5*time.Second)), "a second delivery of the restore")
	assert.True(t, b.seen(key, first), "a late delivery of the first assignment")
}

// A pull returns the same assigned_at the assignment's first delivery claimed, and a later
// assignment of the same retry has a later one.
func TestPubSubDeliveryClaimComparesAssignedAt(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})
	assignedAt := time.Now()
	key := assignmentKey{taskId: 3, retryCount: 0}

	assert.True(t, b.claim(key, assignedAt))
	assert.True(t, b.seen(key, assignedAt), "the pull finds the same assignment")
	assert.False(t, b.seen(key, assignedAt.Add(time.Microsecond)), "a later assignment of the same retry")
	assert.True(t, b.claim(assignmentKey{taskId: 3, retryCount: 1}, assignedAt), "a new retry is a new assignment")
}

func TestPubSubDeliveryClaimDispatchMixesTrackedDuplicateAndUntracked(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})
	assignedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	assert.True(t, b.claim(assignmentKey{taskId: 1}, assignedAt.Time))

	kept := taskIds(b.claimDispatch(dispatchOf(map[int64]pgtype.Timestamptz{1: assignedAt, 2: assignedAt, 3: {}})))
	assert.ElementsMatch(t, []int64{2, 3}, kept)
}

// Without a pull nothing resends, so nothing is tracked or dropped.
func TestPubSubDeliveryTracksNothingWithoutAPull(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{})
	key := assignmentKey{taskId: 1}
	at := time.Now()

	assert.True(t, b.claim(key, at))
	assert.True(t, b.claim(key, at))
	assert.Equal(t, 0, b.received.Len())
}

func TestPubSubDeliveryClaimAlwaysSendsWithoutAssignedAt(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})

	for range 2 {
		assert.Equal(t, []int64{7}, taskIds(b.claimDispatch(dispatchOf(map[int64]pgtype.Timestamptz{7: {}}))))
	}

	var unwired *pubSubDelivery
	assert.True(t, unwired.claim(assignmentKey{taskId: 7}, time.Now()))
}

// Until something sends over pub/sub every assignment comes over the durable queue, so the pull, which
// only recovers lost pub/sub messages, doesn't run.
func TestPubSubDeliveryPullWaitsForTheFirstPubSubMessage(t *testing.T) {
	b := newPubSubDelivery(PubSubDeliveryOpts{Pull: true})

	select {
	case <-b.activated:
		t.Fatal("pull released before any pub/sub message")
	default:
	}

	b.markActive()
	b.markActive()

	select {
	case <-b.activated:
	default:
		t.Fatal("pull not released by the first pub/sub message")
	}
}
