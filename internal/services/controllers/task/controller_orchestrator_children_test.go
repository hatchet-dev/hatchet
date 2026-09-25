package task

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// recordingMQ keeps every message sent to it and fails the sends whose 1-based numbers are in
// failing.
type recordingMQ struct {
	fakeMQ
	failing map[int]bool
	msgs    []*msgqueue.Message
	mu      sync.Mutex
}

func (m *recordingMQ) SendMessage(_ context.Context, _ msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.msgs = append(m.msgs, msg)

	if m.failing[len(m.msgs)] {
		return fmt.Errorf("fake broker unavailable (send %d)", len(m.msgs))
	}

	return nil
}

func childrenFixture(n int) []v1.TaskIdInsertedAtRetryCount {
	insertedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	children := make([]v1.TaskIdInsertedAtRetryCount, n)

	for i := range children {
		children[i] = v1.TaskIdInsertedAtRetryCount{Id: int64(i + 1), InsertedAt: insertedAt}
	}

	return children
}

func withWorker(row *sqlcv1.ReleaseTasksRow) *sqlcv1.ReleaseTasksRow {
	row.WorkerID = uuid.New()

	return row
}

func TestOrchestratorIdsWithoutWorker(t *testing.T) {
	evicted := orchestratorRow(1, true)
	alsoEvicted := orchestratorRow(2, true)

	tests := []struct {
		name     string
		released []*sqlcv1.ReleaseTasksRow
		want     []uuid.UUID
	}{
		{
			name:     "orchestrator without a worker is included",
			released: []*sqlcv1.ReleaseTasksRow{evicted},
			want:     []uuid.UUID{evicted.ExternalID},
		},
		{
			name:     "orchestrator on a live operator worker is left to the operator",
			released: []*sqlcv1.ReleaseTasksRow{withWorker(orchestratorRow(1, true))},
		},
		{
			name:     "non-orchestrator without a worker is skipped",
			released: []*sqlcv1.ReleaseTasksRow{childRow(1)},
		},
		{
			name:     "stale (non-current-retry) orchestrator is skipped",
			released: []*sqlcv1.ReleaseTasksRow{orchestratorRow(1, false)},
		},
		{
			name:     "nil row is skipped",
			released: []*sqlcv1.ReleaseTasksRow{nil, evicted},
			want:     []uuid.UUID{evicted.ExternalID},
		},
		{
			name: "only the qualifying orchestrators are returned, in order",
			released: []*sqlcv1.ReleaseTasksRow{
				childRow(3),
				evicted,
				withWorker(orchestratorRow(4, true)),
				alsoEvicted,
			},
			want: []uuid.UUID{evicted.ExternalID, alsoEvicted.ExternalID},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, orchestratorIdsWithoutWorker(tc.released))
		})
	}
}

func TestPublishChildCancellations_BoundsEachPayload(t *testing.T) {
	tests := []struct {
		name         string
		children     int
		wantMessages int
	}{
		{name: "no children sends nothing", children: 0, wantMessages: 0},
		{name: "one child", children: 1, wantMessages: 1},
		{name: "exactly one batch", children: BULK_MSG_BATCH_SIZE, wantMessages: 1},
		{name: "one over a batch", children: BULK_MSG_BATCH_SIZE + 1, wantMessages: 2},
		{name: "many batches", children: BULK_MSG_BATCH_SIZE*5 + 7, wantMessages: 6},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mq := &recordingMQ{}
			c := newTestController(mq)
			children := childrenFixture(tc.children)

			require.NoError(t, c.publishChildCancellations(context.Background(), uuid.New(), children))
			require.Len(t, mq.msgs, tc.wantMessages)

			var got []int64

			for _, msg := range mq.msgs {
				assert.Equal(t, msgqueue.MsgIDCancelTasks, msg.ID)
				require.Len(t, msg.Payloads, 1, "each message carries a single payload")

				payloads := msgqueue.JSONConvert[tasktypes.CancelTasksPayload](msg.Payloads)
				require.Len(t, payloads, 1)
				assert.LessOrEqual(t, len(payloads[0].Tasks), BULK_MSG_BATCH_SIZE)

				for _, task := range payloads[0].Tasks {
					got = append(got, task.Id)
				}
			}

			var want []int64

			for _, child := range children {
				want = append(want, child.Id)
			}

			assert.Equal(t, want, got, "every child is published exactly once, in order")
		})
	}
}

func TestPublishChildCancellations_FailedBatchDoesNotStopTheRest(t *testing.T) {
	mq := &recordingMQ{failing: map[int]bool{1: true}}
	c := newTestController(mq)

	err := c.publishChildCancellations(context.Background(), uuid.New(), childrenFixture(BULK_MSG_BATCH_SIZE*2+1))

	require.Error(t, err)
	assert.Len(t, mq.msgs, 3, "the batches after a failed one are still published")
}
