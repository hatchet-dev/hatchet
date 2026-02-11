package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type ListActiveWorkersResult struct {
	ID      uuid.UUID
	MaxRuns int
	Name    string
	Labels  []*sqlcv1.ListManyWorkerLabelsRow
}

type leaseRepository struct {
	*sharedRepository
}

func newLeaseRepository(shared *sharedRepository) *leaseRepository {
	return &leaseRepository{
		sharedRepository: shared,
	}
}

func (d *leaseRepository) AcquireOrExtendLeases(ctx context.Context, tenantId uuid.UUID, kind sqlcv1.LeaseKind, resourceIds []string, existingLeases []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	ctx, span := telemetry.NewSpan(ctx, "acquire-leases")
	defer span.End()

	leaseIds := make([]int64, len(existingLeases))

	for i, lease := range existingLeases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = d.queries.GetLeasesToAcquire(ctx, tx, sqlcv1.GetLeasesToAcquireParams{
		Kind:        kind,
		Resourceids: resourceIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, err
	}

	leases, err := d.queries.AcquireOrExtendLeases(ctx, tx, sqlcv1.AcquireOrExtendLeasesParams{
		Kind:             kind,
		Resourceids:      resourceIds,
		Tenantid:         tenantId,
		Existingleaseids: leaseIds,
	})

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return leases, nil
}

func (d *leaseRepository) ReleaseLeases(ctx context.Context, tenantId uuid.UUID, leases []*sqlcv1.Lease) error {
	ctx, span := telemetry.NewSpan(ctx, "release-leases")
	defer span.End()

	leaseIds := make([]int64, len(leases))

	for i, lease := range leases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l)

	if err != nil {
		return err
	}

	defer rollback()

	_, err = d.queries.ReleaseLeases(ctx, tx, leaseIds)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (d *leaseRepository) ListQueues(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1Queue, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queues")
	defer span.End()

	return d.queries.ListQueues(ctx, d.pool, tenantId)
}

func (d *leaseRepository) ListActiveWorkers(ctx context.Context, tenantId uuid.UUID) ([]*ListActiveWorkersResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-active-workers")
	defer span.End()

	activeWorkers, err := d.queries.ListActiveWorkers(ctx, d.pool, tenantId)

	if err != nil {
		return nil, err
	}

	workerIds := make([]uuid.UUID, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		workerIds = append(workerIds, worker.ID)
	}

	labels, err := d.queries.ListManyWorkerLabels(ctx, d.pool, workerIds)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	workerIdsToLabels := make(map[uuid.UUID][]*sqlcv1.ListManyWorkerLabelsRow, len(labels))

	for _, label := range labels {
		if _, ok := workerIdsToLabels[label.WorkerId]; !ok {
			workerIdsToLabels[label.WorkerId] = make([]*sqlcv1.ListManyWorkerLabelsRow, 0)
		}

		workerIdsToLabels[label.WorkerId] = append(workerIdsToLabels[label.WorkerId], label)
	}

	res := make([]*ListActiveWorkersResult, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		res = append(res, &ListActiveWorkersResult{
			ID:      worker.ID,
			MaxRuns: int(worker.MaxRuns),
			Labels:  workerIdsToLabels[worker.ID],
			Name:    worker.Name,
		})
	}

	return res, nil
}

func (d *leaseRepository) GetActiveWorker(ctx context.Context, tenantId, workerId uuid.UUID) (*ListActiveWorkersResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-active-worker")
	defer span.End()

	worker, err := d.queries.GetActiveWorkerById(ctx, d.pool, sqlcv1.GetActiveWorkerByIdParams{
		Tenantid: tenantId,
		ID:       workerId,
	})

	if err != nil {
		return nil, err
	}

	labels, err := d.queries.ListManyWorkerLabels(ctx, d.pool, []uuid.UUID{workerId})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	workerIdsToLabels := make(map[uuid.UUID][]*sqlcv1.ListManyWorkerLabelsRow, len(labels))

	for _, label := range labels {
		if _, ok := workerIdsToLabels[label.WorkerId]; !ok {
			workerIdsToLabels[label.WorkerId] = make([]*sqlcv1.ListManyWorkerLabelsRow, 0)
		}

		workerIdsToLabels[label.WorkerId] = append(workerIdsToLabels[label.WorkerId], label)
	}

	return &ListActiveWorkersResult{
		ID:      worker.Worker.ID,
		MaxRuns: int(worker.Worker.MaxRuns),
		Labels:  workerIdsToLabels[worker.Worker.ID],
		Name:    worker.Worker.Name,
	}, nil
}

func (d *leaseRepository) ListConcurrencyStrategies(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-concurrency-strategies")
	defer span.End()

	return d.queries.ListActiveConcurrencyStrategies(ctx, d.pool, tenantId)
}

func (d *leaseRepository) GetConcurrencyStrategy(ctx context.Context, tenantId uuid.UUID, id int64) (*sqlcv1.V1StepConcurrency, error) {
	ctx, span := telemetry.NewSpan(ctx, "get-concurrency-strategy")
	defer span.End()

	return d.queries.GetConcurrencyStrategyById(ctx, d.pool, sqlcv1.GetConcurrencyStrategyByIdParams{
		ID:       id,
		Tenantid: tenantId,
	})
}
