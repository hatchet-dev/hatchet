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
	ID     uuid.UUID
	Name   string
	Labels []*sqlcv1.ListManyWorkerLabelsRow

	// TotalSlotsByType is the worker's total slot capacity keyed by slot type.
	TotalSlotsByType map[string]int
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

	// the query returns one row per (worker, slot type)
	activeWorkerRows, err := d.queries.ListActiveWorkers(ctx, d.pool, tenantId)

	if err != nil {
		return nil, err
	}

	workerIdsToResults := make(map[uuid.UUID]*ListActiveWorkersResult, len(activeWorkerRows))
	res := make([]*ListActiveWorkersResult, 0, len(activeWorkerRows))
	workerIds := make([]uuid.UUID, 0, len(activeWorkerRows))

	for _, row := range activeWorkerRows {
		worker, ok := workerIdsToResults[row.ID]
		if !ok {
			worker = &ListActiveWorkersResult{
				ID:               row.ID,
				Name:             row.Name,
				TotalSlotsByType: make(map[string]int),
			}

			workerIdsToResults[row.ID] = worker
			res = append(res, worker)
			workerIds = append(workerIds, row.ID)
		}

		worker.TotalSlotsByType[row.SlotType] += int(row.MaxUnits)
	}

	labels, err := d.queries.ListManyWorkerLabels(ctx, d.pool, workerIds)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	for _, label := range labels {
		if worker, ok := workerIdsToResults[label.WorkerId]; ok {
			worker.Labels = append(worker.Labels, label)
		}
	}

	return res, nil
}

// listTotalSlotsForWorkers returns each worker's total slot capacity keyed by slot type.
func (d *leaseRepository) listTotalSlotsForWorkers(ctx context.Context, tenantId uuid.UUID, workerIds []uuid.UUID) (map[uuid.UUID]map[string]int, error) {
	slotConfigs, err := d.queries.ListWorkerSlotConfigs(ctx, d.pool, sqlcv1.ListWorkerSlotConfigsParams{
		Tenantid:  tenantId,
		Workerids: workerIds,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	workerIdsToTotalSlots := make(map[uuid.UUID]map[string]int, len(workerIds))

	for _, config := range slotConfigs {
		if _, ok := workerIdsToTotalSlots[config.WorkerID]; !ok {
			workerIdsToTotalSlots[config.WorkerID] = make(map[string]int)
		}

		workerIdsToTotalSlots[config.WorkerID][config.SlotType] += int(config.MaxUnits)
	}

	return workerIdsToTotalSlots, nil
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

	workerIdsToTotalSlots, err := d.listTotalSlotsForWorkers(ctx, tenantId, []uuid.UUID{workerId})

	if err != nil {
		return nil, err
	}

	return &ListActiveWorkersResult{
		ID:               worker.Worker.ID,
		Labels:           workerIdsToLabels[worker.Worker.ID],
		Name:             worker.Worker.Name,
		TotalSlotsByType: workerIdsToTotalSlots[worker.Worker.ID],
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
