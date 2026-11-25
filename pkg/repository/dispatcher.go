package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateDispatcherOpts struct {
	ID uuid.UUID `validate:"required"`
}

type DispatcherRepository interface {
	// ListUnackedAssignments returns assignments to the given workers, made in [assignedAfter,
	// assignedBefore), that no worker has acknowledged. Batch members are left out. It relies on assigned_at, which is only
	// written when ReassignUnackedTasks is enabled.
	ListUnackedAssignments(ctx context.Context, workerIds []uuid.UUID, assignedAfter, assignedBefore time.Time, limitPerWorker int) ([]UnackedAssignment, error)

	// CreateNewDispatcher creates a new dispatcher for a given tenant.
	CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error)

	// Heartbeat records that the dispatcher is alive and can write to the database. A heartbeat
	// that follows a gap of more than 15 seconds restarts the dispatcher's unbroken run, which
	// pauses task reassignment (see TaskRepository.ProcessTaskReassignments).
	Heartbeat(ctx context.Context, dispatcherId uuid.UUID) error

	Delete(ctx context.Context, dispatcherId uuid.UUID) error

	// MarkInactive flags the dispatcher as shutting down. ClaimOperators then claims nothing
	// for it and lets other dispatchers take its operators over at once.
	MarkInactive(ctx context.Context, dispatcherId uuid.UUID) error
}

type dispatcherRepository struct {
	*sharedRepository
}

func newDispatcherRepository(shared *sharedRepository) DispatcherRepository {
	return &dispatcherRepository{
		sharedRepository: shared,
	}
}

func (d *dispatcherRepository) CreateNewDispatcher(ctx context.Context, opts *CreateDispatcherOpts) (*sqlcv1.Dispatcher, error) {
	if err := d.v.Validate(opts); err != nil {
		return nil, err
	}

	return d.queries.CreateDispatcher(ctx, d.pool.ForShared(), opts.ID)
}

func (d *dispatcherRepository) Heartbeat(ctx context.Context, dispatcherId uuid.UUID) error {
	return d.queries.HeartbeatDispatcher(ctx, d.pool.ForShared(), dispatcherId)
}

func (d *dispatcherRepository) Delete(ctx context.Context, dispatcherId uuid.UUID) error {
	_, err := d.queries.DeleteDispatcher(ctx, d.pool.ForShared(), dispatcherId)
	return err
}

func (d *dispatcherRepository) MarkInactive(ctx context.Context, dispatcherId uuid.UUID) error {
	_, err := d.queries.SetDispatchersInactive(ctx, d.pool.ForShared(), []uuid.UUID{dispatcherId})
	return err
}

type UnackedAssignment struct {
	TenantID   uuid.UUID
	WorkerID   uuid.UUID
	TaskID     int64
	RetryCount int32
	AssignedAt time.Time
}

// The lateral join resolves each worker's tenant first so the runtime lookup can use the
// (tenant_id, worker_id) index; there is no index on worker_id alone.
const listUnackedAssignmentsSQL = `
SELECT w."tenantId", w.id, rt.task_id, rt.retry_count, rt.assigned_at
FROM "Worker" w
JOIN LATERAL (
    SELECT rt.task_id, rt.retry_count, rt.assigned_at
    FROM v1_task_runtime rt
    WHERE rt.tenant_id = w."tenantId"
        AND rt.worker_id = w.id
        AND rt.evicted_at IS NULL
        AND rt.acked_at IS NULL
        -- the pull can only rebuild single-task starts, not a batch start with all its members
        AND rt.batch_id IS NULL
        AND rt.assigned_at >= $2
        AND rt.assigned_at < $3
    ORDER BY rt.assigned_at
    LIMIT $4
) rt ON true
WHERE w.id = ANY($1::uuid[])
`

func (d *dispatcherRepository) ListUnackedAssignments(ctx context.Context, workerIds []uuid.UUID, assignedAfter, assignedBefore time.Time, limitPerWorker int) ([]UnackedAssignment, error) {
	if len(workerIds) == 0 {
		return nil, nil
	}

	rows, err := d.pool.ForShared().Query(ctx, listUnackedAssignmentsSQL, workerIds, assignedAfter, assignedBefore, limitPerWorker)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	res := make([]UnackedAssignment, 0)

	for rows.Next() {
		var a UnackedAssignment

		if err := rows.Scan(&a.TenantID, &a.WorkerID, &a.TaskID, &a.RetryCount, &a.AssignedAt); err != nil {
			return nil, err
		}

		res = append(res, a)
	}

	return res, rows.Err()
}
