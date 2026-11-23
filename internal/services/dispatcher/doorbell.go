package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

type DoorbellOpts struct {
	PullInterval time.Duration
	PullGrace    time.Duration
}

// pullLimitPerWorker bounds one pull's read per worker; anything beyond it is picked up by the next pull.
const pullLimitPerWorker = 200

// receivedTTL outlasts ReassignUnackedTasks' one-minute cutoff, after which an unacknowledged
// assignment is reassigned with a new retry count and so a new key.
const receivedTTL = 3 * time.Minute

type assignmentKey struct {
	taskId     int64
	retryCount int32
}

// doorbell tracks which assignments this dispatcher already handled, from pub/sub or from a pull, so
// that a pull doesn't resend a task the worker already has and a late pub/sub message doesn't resend a
// task the pull already sent.
type doorbell struct {
	opts DoorbellOpts

	// startedAt bounds the pull to assignments made while this dispatcher was up. Older ones were
	// addressed to whichever dispatcher the worker used before, which may have sent them, so they are
	// left to ReassignUnackedTasks rather than risk a duplicate.
	startedAt time.Time

	mu       sync.Mutex
	received *expirable.LRU[assignmentKey, struct{}]
}

func newDoorbellFromOpts(opts *DoorbellOpts) *doorbell {
	if opts == nil {
		return nil
	}

	return newDoorbell(*opts)
}

func newDoorbell(opts DoorbellOpts) *doorbell {
	return &doorbell{
		opts:      opts,
		startedAt: time.Now(),
		received:  expirable.NewLRU[assignmentKey, struct{}](1_000_000, nil, receivedTTL),
	}
}

// claim drops the assignments already handled and marks the rest as handled. Without retry counts
// (a message from an older scheduler) every assignment is kept, as before.
func (b *doorbell) claim(payload *tasktypesv1.TaskAssignedBulkTaskPayload) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for workerId, taskIds := range payload.WorkerIdToTaskIds {
		retryCounts, ok := payload.WorkerIdToTaskRetryCounts[workerId]

		if !ok {
			continue
		}

		kept := taskIds[:0]

		for _, taskId := range taskIds {
			key := assignmentKey{taskId: taskId, retryCount: retryCounts[taskId]}

			if b.received.Contains(key) {
				continue
			}

			b.received.Add(key, struct{}{})
			kept = append(kept, taskId)
		}

		if len(kept) == 0 {
			delete(payload.WorkerIdToTaskIds, workerId)
			continue
		}

		payload.WorkerIdToTaskIds[workerId] = kept
	}
}

func (b *doorbell) seen(key assignmentKey) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.received.Contains(key)
}

// startDoorbell subscribes to the dispatcher's pub/sub topic alongside its durable queue, so engines
// that still send over the queue keep working during a rollout.
func (d *DispatcherImpl) startDoorbell(ctx context.Context, dispatcherId uuid.UUID, handler msgqueue.MsgHandler) (func() error, error) {
	cleanup, err := d.pubsub.Sub(msgqueue.DispatcherTopic(dispatcherId), handler)

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to dispatcher topic: %w", err)
	}

	go d.runPull(ctx)

	return cleanup, nil
}

func (d *DispatcherImpl) runPull(ctx context.Context) {
	ticker := time.NewTicker(d.doorbell.opts.PullInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		if err := d.pullOnce(ctx); err != nil && ctx.Err() == nil {
			d.l.Error().Ctx(ctx).Err(err).Msg("could not pull unacknowledged assignments")
		}
	}
}

func (d *DispatcherImpl) pullOnce(ctx context.Context) error {
	workerIds := connectedWorkerIds(d.workers)

	if len(workerIds) == 0 {
		return nil
	}

	rows, err := d.repov1.Dispatcher().ListUnackedAssignments(
		ctx,
		workerIds,
		d.doorbell.startedAt,
		time.Now().Add(-d.doorbell.opts.PullGrace),
		pullLimitPerWorker,
	)

	if err != nil {
		return err
	}

	byTenant := make(map[uuid.UUID]*tasktypesv1.TaskAssignedBulkTaskPayload)

	for _, row := range rows {
		if d.doorbell.seen(assignmentKey{taskId: row.TaskID, retryCount: row.RetryCount}) {
			continue
		}

		if byTenant[row.TenantID] == nil {
			byTenant[row.TenantID] = &tasktypesv1.TaskAssignedBulkTaskPayload{}
		}

		byTenant[row.TenantID].Add(row.WorkerID, row.TaskID, row.RetryCount)
	}

	for tenantId, payload := range byTenant {
		count := 0

		for _, taskIds := range payload.WorkerIdToTaskIds {
			count += len(taskIds)
		}

		msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDTaskAssignedBulk, false, true, payload)

		if err != nil {
			return err
		}

		prometheus.DispatcherPulledAssignments.Add(float64(count))
		d.l.Warn().Ctx(ctx).Int("tasks", count).Str("tenant", tenantId.String()).Msg("pulled assignments that never arrived over pub/sub")

		if err := d.handleTaskBulkAssignedTask(ctx, msg); err != nil {
			return err
		}
	}

	return nil
}

func connectedWorkerIds(w *workers) []uuid.UUID {
	ids := make([]uuid.UUID, 0)

	w.Range(func(workerId uuid.UUID, _ *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		ids = append(ids, workerId)
		return true
	})

	return ids
}
