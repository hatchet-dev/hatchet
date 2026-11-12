package dispatcher

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

const orderedReleaseTestTimeout = 200 * time.Millisecond

type orderedReleaseTestSession struct {
	d         *DispatcherServiceImpl
	inv       *durableTaskInvocation
	tenantId  uuid.UUID
	taskId    uuid.UUID
	delivered []int64
}

// newOrderedReleaseTestSession registers a live invocation so completions go through
// DeliverDurableEventLogEntryCompletion. Node ids equal satisfied orders, so delivered
// shows the release order.
func newOrderedReleaseTestSession() *orderedReleaseTestSession {
	s := &orderedReleaseTestSession{d: newTestDispatcher(), tenantId: uuid.New(), taskId: uuid.New()}
	s.inv = &durableTaskInvocation{
		tenantId: s.tenantId,
		l:        s.d.l,
		sendFn: func(resp *contracts.DurableTaskResponse) error {
			s.delivered = append(s.delivered, resp.GetEntryCompleted().GetRef().GetNodeId())
			return nil
		},
	}
	s.d.durableInvocations.Store(durableInvocationsKey{tenantId: s.tenantId, taskId: s.taskId}, s.inv)

	return s
}

func (s *orderedReleaseTestSession) deliver(t *testing.T, order int64) {
	t.Helper()
	deliverCompletion(t, s.d, s.tenantId, s.taskId, order, order)
}

func (s *orderedReleaseTestSession) timedOut() []timedOutOrderedRelease {
	return s.inv.timedOutOrderedReleases(orderedReleaseTestTimeout)
}

// A replay keeps a slow child's completion held for most of the replay while it keeps releasing
// other completions; that must not count as a stalled gap.
func TestOrderedRelease_HoldDoesNotTimeOutWhileReleasing(t *testing.T) {
	s := newOrderedReleaseTestSession()

	s.deliver(t, 1)
	s.deliver(t, 50) // held until order 49 arrives

	for order := int64(2); order <= 10; order++ {
		time.Sleep(orderedReleaseTestTimeout / 4)
		s.deliver(t, order)
		require.Empty(t, s.timedOut(), "released order %d, the gap is not stalled", order)
	}

	require.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, s.delivered)
}

func TestOrderedRelease_TimesOutWhenNothingIsReleased(t *testing.T) {
	s := newOrderedReleaseTestSession()

	s.deliver(t, 1)
	s.deliver(t, 3)
	s.deliver(t, 4)

	time.Sleep(2 * orderedReleaseTestTimeout)

	timedOut := s.timedOut()
	require.Len(t, timedOut, 1)
	require.Equal(t, int64(2), timedOut[0].missingSatisfiedOrder)
	require.Equal(t, []int64{3, 4}, timedOut[0].heldSatisfiedOrders)
	require.Equal(t, []int64{1}, s.delivered)
}

// Completions that are only held are not progress: if the gap never fills, the invocation must
// still time out even though new completions keep arriving behind it.
func TestOrderedRelease_HeldArrivalsDoNotResetStall(t *testing.T) {
	s := newOrderedReleaseTestSession()

	s.deliver(t, 1)
	s.deliver(t, 3)

	for order := int64(4); order <= 9; order++ {
		time.Sleep(orderedReleaseTestTimeout / 4)
		s.deliver(t, order)
	}

	timedOut := s.timedOut()
	require.Len(t, timedOut, 1)
	require.Equal(t, int64(2), timedOut[0].missingSatisfiedOrder)
	require.Equal(t, []int64{1}, s.delivered)
}

// The stall clock starts when a completion is first held, not at an earlier release.
func TestOrderedRelease_StallMeasuredFromFirstHoldAfterIdleRelease(t *testing.T) {
	s := newOrderedReleaseTestSession()

	s.deliver(t, 1)
	time.Sleep(2 * orderedReleaseTestTimeout)
	s.deliver(t, 3)

	require.Empty(t, s.timedOut())

	time.Sleep(2 * orderedReleaseTestTimeout)

	require.Len(t, s.timedOut(), 1)
}

// When a reconnected worker reports a higher satisfied order, the cursor advance counts as
// progress even if the gap behind the held completions is not filled yet.
func TestOrderedRelease_ResumeCountsAsProgress(t *testing.T) {
	s := newOrderedReleaseTestSession()
	releaseKey := orderedReleaseKey{taskExternalId: s.taskId, invocationCount: 1}

	s.deliver(t, 1)
	s.deliver(t, 5)

	time.Sleep(orderedReleaseTestTimeout * 3 / 2)

	// check straight after the resume, as handleWorkerStatus runs the eviction check right after
	// resumeOrderedRelease
	require.NoError(t, s.inv.resumeOrderedRelease(releaseKey, 3))
	require.Empty(t, s.timedOut(), "the cursor moved to 3 after the hold timed out")

	time.Sleep(2 * orderedReleaseTestTimeout)

	timedOut := s.timedOut()
	require.Len(t, timedOut, 1)
	require.Equal(t, int64(4), timedOut[0].missingSatisfiedOrder)
	require.Equal(t, []int64{5}, timedOut[0].heldSatisfiedOrders)
}
