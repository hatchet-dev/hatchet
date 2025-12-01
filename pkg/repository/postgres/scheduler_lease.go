package postgres

import (
	"github.com/google/uuid"

	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type leaseRepository struct {
	*sharedRepository
}

func newLeaseRepository(shared *sharedRepository) *leaseRepository {
	return &leaseRepository{
		sharedRepository: shared,
	}
}

func (d *leaseRepository) AcquireOrExtendLeases(ctx context.Context, tenantId uuid.UUID, kind dbsqlc.LeaseKind, resourceIds []string, existingLeases []*dbsqlc.Lease) ([]*dbsqlc.Lease, error) {
	ctx, span := telemetry.NewSpan(ctx, "acquire-leases")
	defer span.End()

	leaseIds := make([]int64, len(existingLeases))

	for i, lease := range existingLeases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	err = d.queries.GetLeasesToAcquire(ctx, tx, dbsqlc.GetLeasesToAcquireParams{
		Kind:        kind,
		Resourceids: resourceIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, err
	}

	leases, err := d.queries.AcquireOrExtendLeases(ctx, tx, dbsqlc.AcquireOrExtendLeasesParams{
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

func (d *leaseRepository) ReleaseLeases(ctx context.Context, tenantId uuid.UUID, leases []*dbsqlc.Lease) error {
	ctx, span := telemetry.NewSpan(ctx, "release-leases")
	defer span.End()

	leaseIds := make([]int64, len(leases))

	for i, lease := range leases {
		leaseIds[i] = lease.ID
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, d.pool, d.l, 5000)

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

func (d *leaseRepository) ListQueues(ctx context.Context, tenantId uuid.UUID) ([]*dbsqlc.Queue, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queues")
	defer span.End()

	return d.queries.ListQueues(ctx, d.pool, tenantId)
}

func (d *leaseRepository) ListActiveWorkers(ctx context.Context, tenantId uuid.UUID) ([]*repository.ListActiveWorkersResult, error) {
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

	workerIdsToLabels := make(map[string][]*dbsqlc.ListManyWorkerLabelsRow, len(labels))

	for _, label := range labels {
		wId := label.WorkerId.String()

		if _, ok := workerIdsToLabels[wId]; !ok {
			workerIdsToLabels[wId] = make([]*dbsqlc.ListManyWorkerLabelsRow, 0)
		}

		workerIdsToLabels[wId] = append(workerIdsToLabels[wId], label)
	}

	res := make([]*repository.ListActiveWorkersResult, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		wId := worker.ID.String()
		res = append(res, &repository.ListActiveWorkersResult{
			ID:      wId,
			MaxRuns: int(worker.MaxRuns),
			Labels:  workerIdsToLabels[wId],
		})
	}

	return res, nil
}
