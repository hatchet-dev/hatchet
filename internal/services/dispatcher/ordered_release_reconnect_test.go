package dispatcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func deliverCompletion(t *testing.T, d *DispatcherServiceImpl, tenantId, taskId uuid.UUID, nodeId, satisfiedOrder int64) {
	t.Helper()
	require.NoError(t, d.DeliverDurableEventLogEntryCompletion(tenantId, taskId, 1, 0, nodeId, []byte("{}"), &satisfiedOrder, false, nil))
}

func TestOrderedReleaseResumesAfterStreamReconnect(t *testing.T) {
	d := newTestDispatcher()
	tenantId, taskId := uuid.New(), uuid.New()
	key := durableInvocationsKey{tenantId: tenantId, taskId: taskId}
	releaseKey := orderedReleaseKey{taskExternalId: taskId, invocationCount: 1}

	var delivered []int64
	newSession := func() *durableTaskInvocation {
		return &durableTaskInvocation{
			tenantId: tenantId,
			l:        d.l,
			sendFn: func(resp *contracts.DurableTaskResponse) error {
				delivered = append(delivered, resp.GetEntryCompleted().GetRef().GetNodeId())
				return nil
			},
		}
	}

	d.durableInvocations.Store(key, newSession())
	deliverCompletion(t, d, tenantId, taskId, 1, 1)
	require.Equal(t, []int64{1}, delivered)

	reconnected := newSession()
	d.durableInvocations.Store(key, reconnected)

	deliverCompletion(t, d, tenantId, taskId, 3, 3)
	deliverCompletion(t, d, tenantId, taskId, 2, 2)
	require.Equal(t, []int64{1}, delivered)

	require.NoError(t, reconnected.resumeOrderedRelease(releaseKey, 1))
	require.Equal(t, []int64{1, 2, 3}, delivered)
	require.Empty(t, reconnected.timedOutOrderedReleases(0))

	require.NoError(t, reconnected.resumeOrderedRelease(releaseKey, 2))
	deliverCompletion(t, d, tenantId, taskId, 4, 4)
	require.Equal(t, []int64{1, 2, 3, 4}, delivered)
}
