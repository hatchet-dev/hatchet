package task

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeMQ struct {
	sends     atomic.Int64
	failN     int32 // number of leading SendMessage calls that return an error
	lastMsg   atomic.Pointer[msgqueue.Message]
	lastQueue atomic.Pointer[string]
}

func (m *fakeMQ) SendMessage(_ context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	n := m.sends.Add(1)
	name := q.Name()
	m.lastQueue.Store(&name)
	m.lastMsg.Store(msg)

	if int32(n) <= m.failN {
		return fmt.Errorf("fake broker unavailable (call %d)", n)
	}

	return nil
}

func (m *fakeMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *fakeMQ) SetQOS(_ int)                                        {}
func (m *fakeMQ) Subscribe(_ msgqueue.Queue, _, _ msgqueue.MsgHandler) (func() error, error) {
	return nil, nil
}
func (m *fakeMQ) IsReady() bool { return true }

func newTestController(mq msgqueue.MessageQueue) *TasksControllerImpl {
	l := zerolog.Nop()
	return &TasksControllerImpl{
		mq: mq,
		l:  &l,
	}
}

func orchestratorRow(id int64, currentRetry bool) *sqlcv1.ReleaseTasksRow {
	return &sqlcv1.ReleaseTasksRow{
		ID:                id,
		ExternalID:        uuid.New(),
		RetryCount:        0,
		IsCurrentRetry:    currentRetry,
		IsDagOrchestrator: true,
	}
}

func childRow(id int64) *sqlcv1.ReleaseTasksRow {
	return &sqlcv1.ReleaseTasksRow{
		ID:                id,
		ExternalID:        uuid.New(),
		RetryCount:        0,
		IsCurrentRetry:    true,
		IsDagOrchestrator: false,
	}
}

func TestEmitOrchestratorTerminalEvents_Filtering(t *testing.T) {
	tenantId := uuid.New()

	tests := []struct {
		name        string
		released    []*sqlcv1.ReleaseTasksRow
		skipRetried map[int64]struct{}
		wantSends   int64
	}{
		{
			name:      "non-orchestrator rows are skipped",
			released:  []*sqlcv1.ReleaseTasksRow{childRow(1), childRow(2)},
			wantSends: 0,
		},
		{
			name:      "stale (non-current-retry) orchestrator is skipped",
			released:  []*sqlcv1.ReleaseTasksRow{orchestratorRow(1, false)},
			wantSends: 0,
		},
		{
			name:        "retried orchestrator is skipped",
			released:    []*sqlcv1.ReleaseTasksRow{orchestratorRow(1, true)},
			skipRetried: map[int64]struct{}{1: {}},
			wantSends:   0,
		},
		{
			name:      "nil row is skipped",
			released:  []*sqlcv1.ReleaseTasksRow{nil, orchestratorRow(2, true)},
			wantSends: 1,
		},
		{
			name:      "one orchestrator among children: one send",
			released:  []*sqlcv1.ReleaseTasksRow{childRow(1), orchestratorRow(2, true), childRow(3)},
			wantSends: 1,
		},
		{
			name:      "multiple orchestrators: one send each",
			released:  []*sqlcv1.ReleaseTasksRow{orchestratorRow(1, true), orchestratorRow(2, true)},
			wantSends: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mq := &fakeMQ{}
			c := newTestController(mq)

			err := c.emitOrchestratorTerminalEvents(
				context.Background(), tenantId, tc.released,
				map[int64]string{1: "d1", 2: "d2", 3: "d3"},
				tc.skipRetried, sqlcv1.V1EventTypeOlapFINISHED,
			)

			require.NoError(t, err)
			assert.Equal(t, tc.wantSends, mq.sends.Load())
		})
	}
}

func TestEmitOrchestratorTerminalEvents_PayloadRouting(t *testing.T) {
	tenantId := uuid.New()

	t.Run("FINISHED puts detail in EventPayload", func(t *testing.T) {
		mq := &fakeMQ{}
		c := newTestController(mq)

		err := c.emitOrchestratorTerminalEvent(
			context.Background(), tenantId, 42, 3, sqlcv1.V1EventTypeOlapFINISHED, `{"ok":true}`,
		)
		require.NoError(t, err)
		require.Equal(t, int64(1), mq.sends.Load())
		require.Equal(t, msgqueue.OLAP_QUEUE.Name(), *mq.lastQueue.Load())
	})

	t.Run("CANCELLED puts detail in EventMessage", func(t *testing.T) {
		mq := &fakeMQ{}
		c := newTestController(mq)

		err := c.emitOrchestratorTerminalEvent(
			context.Background(), tenantId, 42, 0, sqlcv1.V1EventTypeOlapCANCELLED, "cancelled by user",
		)
		require.NoError(t, err)
		require.Equal(t, int64(1), mq.sends.Load())
	})
}

func TestEmitOrchestratorTerminalEvent_RetriesThenSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backoff-timing test in -short mode")
	}

	mq := &fakeMQ{failN: 2} // fail the first two attempts, succeed on the third
	c := newTestController(mq)

	err := c.emitOrchestratorTerminalEvent(
		context.Background(), uuid.New(), 1, 0, sqlcv1.V1EventTypeOlapFINISHED, "",
	)

	require.NoError(t, err)
	assert.Equal(t, int64(3), mq.sends.Load())
}

func TestEmitOrchestratorTerminalEvent_ExhaustsRetries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping backoff-timing test in -short mode")
	}

	mq := &fakeMQ{failN: 1 << 30} // always fail
	c := newTestController(mq)

	err := c.emitOrchestratorTerminalEvent(
		context.Background(), uuid.New(), 1, 0, sqlcv1.V1EventTypeOlapFAILED, "boom",
	)

	require.Error(t, err)
	assert.Equal(t, int64(4), mq.sends.Load()) // maxAttempts
}

func TestEmitOrchestratorTerminalEvent_ContextCancelledStopsRetry(t *testing.T) {
	mq := &fakeMQ{failN: 1 << 30} // always fail
	c := newTestController(mq)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: attempt 0 runs, then the retry wait bails immediately

	err := c.emitOrchestratorTerminalEvent(
		ctx, uuid.New(), 1, 0, sqlcv1.V1EventTypeOlapFINISHED, "",
	)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int64(1), mq.sends.Load())
}
