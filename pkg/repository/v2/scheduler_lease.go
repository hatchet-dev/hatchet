package v2

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type ListActiveWorkersResult struct {
	ID      pgtype.UUID
	MaxRuns int
	Labels  []*sqlcv2.ListManyWorkerLabelsRow
}

type leaseRepository struct {
	*sharedRepository
}

func newLeaseRepository(shared *sharedRepository) *leaseRepository {
	return &leaseRepository{
		sharedRepository: shared,
	}
}

func (d *leaseRepository) AcquireOrExtendLeases(ctx context.Context, tenantId pgtype.UUID, kind sqlcv2.LeaseKind, resourceIds []string, existingLeases []*sqlcv2.Lease) ([]*sqlcv2.Lease, error) {
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

	err = d.queries.GetLeasesToAcquire(ctx, tx, sqlcv2.GetLeasesToAcquireParams{
		Kind:        kind,
		Resourceids: resourceIds,
		Tenantid:    tenantId,
	})

	if err != nil {
		return nil, err
	}

	leases, err := d.queries.AcquireOrExtendLeases(ctx, tx, sqlcv2.AcquireOrExtendLeasesParams{
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

func (d *leaseRepository) ReleaseLeases(ctx context.Context, tenantId pgtype.UUID, leases []*sqlcv2.Lease) error {
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

func (d *leaseRepository) ListQueues(ctx context.Context, tenantId pgtype.UUID) ([]*sqlcv2.V2Queue, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-queues")
	defer span.End()

	return d.queries.ListQueues(ctx, d.pool, tenantId)
}

func (d *leaseRepository) ListActiveWorkers(ctx context.Context, tenantId pgtype.UUID) ([]*ListActiveWorkersResult, error) {
	ctx, span := telemetry.NewSpan(ctx, "list-active-workers")
	defer span.End()

	activeWorkers, err := d.queries.ListActiveWorkers(ctx, d.pool, tenantId)

	if err != nil {
		return nil, err
	}

	workerIds := make([]pgtype.UUID, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		workerIds = append(workerIds, worker.ID)
	}

	labels, err := d.queries.ListManyWorkerLabels(ctx, d.pool, workerIds)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	workerIdsToLabels := make(map[string][]*sqlcv2.ListManyWorkerLabelsRow, len(labels))

	for _, label := range labels {
		wId := sqlchelpers.UUIDToStr(label.WorkerId)

		if _, ok := workerIdsToLabels[wId]; !ok {
			workerIdsToLabels[wId] = make([]*sqlcv2.ListManyWorkerLabelsRow, 0)
		}

		workerIdsToLabels[wId] = append(workerIdsToLabels[wId], label)
	}

	res := make([]*ListActiveWorkersResult, 0, len(activeWorkers))

	for _, worker := range activeWorkers {
		wId := sqlchelpers.UUIDToStr(worker.ID)
		res = append(res, &ListActiveWorkersResult{
			ID:      worker.ID,
			MaxRuns: int(worker.MaxRuns),
			Labels:  workerIdsToLabels[wId],
		})
	}

	return res, nil
}
