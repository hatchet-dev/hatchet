package dispatcher

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func newOrderedReleaseTestInvocation(sent *[]int64) *durableTaskInvocation {
	l := zerolog.Nop()

	return &durableTaskInvocation{
		l: &l,
		sendFn: func(resp *contracts.DurableTaskResponse) error {
			*sent = append(*sent, resp.GetEntryCompleted().GetRef().GetNodeId())
			return nil
		},
	}
}

func deliverOrderedCompletion(t *testing.T, inv *durableTaskInvocation, taskId uuid.UUID, order int64) {
	t.Helper()

	resp := &contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &contracts.DurableTaskEventLogEntryCompletedResponse{
				// node id = satisfied order, so the sent sequence shows the release order
				Ref: &contracts.DurableEventLogEntryRef{DurableTaskExternalId: taskId.String(), InvocationCount: 2, NodeId: order},
			},
		},
	}

	require.NoError(t, inv.deliverOrdered(taskId, 2, &order, resp))
}

// A replay keeps a slow child's completion held for most of the replay while it keeps releasing
// other completions; that must not count as a stalled gap.
func TestOrderedRelease_HoldDoesNotTimeOutWhileReleasing(t *testing.T) {
	const timeout = 100 * time.Millisecond

	var sent []int64
	inv := newOrderedReleaseTestInvocation(&sent)
	taskId := uuid.New()

	deliverOrderedCompletion(t, inv, taskId, 1)
	deliverOrderedCompletion(t, inv, taskId, 50) // held until order 49 arrives

	for order := int64(2); order <= 10; order++ {
		time.Sleep(timeout / 4)
		deliverOrderedCompletion(t, inv, taskId, order)
		require.Empty(t, inv.timedOutOrderedReleases(timeout), "released order %d, the gap is not stalled", order)
	}

	require.Equal(t, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, sent)
}

func TestOrderedRelease_TimesOutWhenNothingIsReleased(t *testing.T) {
	const timeout = 50 * time.Millisecond

	var sent []int64
	inv := newOrderedReleaseTestInvocation(&sent)
	taskId := uuid.New()

	deliverOrderedCompletion(t, inv, taskId, 1)
	deliverOrderedCompletion(t, inv, taskId, 3)
	deliverOrderedCompletion(t, inv, taskId, 4)

	time.Sleep(2 * timeout)

	timedOut := inv.timedOutOrderedReleases(timeout)
	require.Len(t, timedOut, 1)
	require.Equal(t, int64(2), timedOut[0].missingSatisfiedOrder)
	require.Equal(t, []int64{3, 4}, timedOut[0].heldSatisfiedOrders)
	require.Equal(t, []int64{1}, sent)
}

// The stall clock starts when a completion is first held, not at an earlier release.
func TestOrderedRelease_StallMeasuredFromFirstHoldAfterIdleRelease(t *testing.T) {
	const timeout = 100 * time.Millisecond

	var sent []int64
	inv := newOrderedReleaseTestInvocation(&sent)
	taskId := uuid.New()

	deliverOrderedCompletion(t, inv, taskId, 1)
	time.Sleep(2 * timeout)
	deliverOrderedCompletion(t, inv, taskId, 3)

	require.Empty(t, inv.timedOutOrderedReleases(timeout))
}
