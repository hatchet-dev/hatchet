package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypesv1 "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// PubSubDeliveryOpts configures how a dispatcher receives the messages other engines send it over pub/sub.
// Pull recovers the assignments pub/sub lost. Without it nothing does: the pull needs assigned_at, which
// only ReassignUnackedTasks writes, so it is off exactly when that is.
type PubSubDeliveryOpts struct {
	Pull         bool
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

// pubSubDelivery tracks which assignments this dispatcher already sent, whether they came over the
// queue, pub/sub, a pull or a local assignment, so that a pull doesn't resend a task the worker already
// has and a late pub/sub message doesn't resend a task the pull already sent.
//
// An assignment is its (task, retry count) plus when it was made: restoring an evicted durable task
// assigns the same retry again, and only its new assigned_at tells it from a second delivery of the
// first assignment.
type pubSubDelivery struct {
	opts PubSubDeliveryOpts

	// startedAt bounds the pull to assignments made while this dispatcher was up. Older ones were
	// addressed to whichever dispatcher the worker used before, which may have sent them, so they are
	// left to ReassignUnackedTasks rather than risk a duplicate.
	startedAt time.Time

	// activated closes on the first message that arrives over pub/sub. Until then every sender uses the
	// durable queue, so the pull, which would only cost a query every interval, doesn't run.
	activated chan struct{}
	activate  sync.Once

	mu       sync.Mutex
	received *expirable.LRU[assignmentKey, time.Time]
}

func newPubSubDeliveryFromOpts(opts *PubSubDeliveryOpts) *pubSubDelivery {
	if opts == nil {
		return nil
	}

	return newPubSubDelivery(*opts)
}

func newPubSubDelivery(opts PubSubDeliveryOpts) *pubSubDelivery {
	return &pubSubDelivery{
		opts:      opts,
		startedAt: time.Now(),
		activated: make(chan struct{}),
		received:  expirable.NewLRU[assignmentKey, time.Time](1_000_000, nil, receivedTTL),
	}
}

func (b *pubSubDelivery) markActive() {
	b.activate.Do(func() { close(b.activated) })
}

// claim tracks nothing without a pull: only the pull resends, so there would be nothing to dedupe
// and the set would only take memory.
func (b *pubSubDelivery) claim(key assignmentKey, assignedAt time.Time) bool {
	if !b.tracking() {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.sentLocked(key, assignedAt) {
		return false
	}

	b.received.Add(key, assignedAt)

	return true
}

func (b *pubSubDelivery) tracking() bool {
	return b != nil && b.opts.Pull
}

func (b *pubSubDelivery) seen(key assignmentKey, assignedAt time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.sentLocked(key, assignedAt)
}

func (b *pubSubDelivery) sentLocked(key assignmentKey, assignedAt time.Time) bool {
	sentAt, ok := b.received.Get(key)

	return ok && !assignedAt.After(sentAt)
}

// claimDispatch leaves out the tasks already sent for the same assignment. A runtime without
// assigned_at is always sent: the pull never finds it, so there is nothing to tell it apart from.
func (b *pubSubDelivery) claimDispatch(data *v1.TasksForDispatch) *v1.TasksForDispatch {
	if !b.tracking() {
		return data
	}

	return &v1.TasksForDispatch{
		Tasks: enumerables.Filter(data.Tasks, func(task *sqlcv1.V1Task) bool {
			rt := data.Runtimes[task.ID]

			if rt == nil || !rt.AssignedAt.Valid {
				return true
			}

			return b.claim(assignmentKey{taskId: task.ID, retryCount: rt.RetryCount}, rt.AssignedAt.Time)
		}),
		Runtimes:    data.Runtimes,
		PayloadRows: data.PayloadRows,
	}
}

// startPubSubDelivery subscribes to the dispatcher's pub/sub topic alongside its durable queue, so it
// receives from engines that send over either.
func (d *DispatcherImpl) startPubSubDelivery(ctx context.Context, dispatcherId uuid.UUID, handler msgqueue.MsgHandler) (func() error, error) {
	cleanup, err := d.pubsub.Sub(msgqueue.DispatcherTopic(dispatcherId), func(msg *msgqueue.Message) error {
		d.pubSubDelivery.markActive()
		return handler(msg)
	})

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to dispatcher topic: %w", err)
	}

	if d.pubSubDelivery.opts.Pull {
		go d.runPull(ctx)
	}

	return cleanup, nil
}

func (d *DispatcherImpl) runPull(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-d.pubSubDelivery.activated:
	}

	ticker := time.NewTicker(d.pubSubDelivery.opts.PullInterval)
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
		d.pubSubDelivery.startedAt,
		time.Now().Add(-d.pubSubDelivery.opts.PullGrace),
		pullLimitPerWorker,
	)

	if err != nil {
		return err
	}

	byTenant := make(map[uuid.UUID]*tasktypesv1.TaskAssignedBulkTaskPayload)

	for _, row := range rows {
		if d.pubSubDelivery.seen(assignmentKey{taskId: row.TaskID, retryCount: row.RetryCount}, row.AssignedAt) {
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
		d.l.Warn().Ctx(ctx).Int("tasks", count).Str("tenant", tenantId.String()).Msg("pulled assignments that this dispatcher never received")

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
