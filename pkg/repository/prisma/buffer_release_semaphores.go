package prisma

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func NewBulkSemaphoreReleaser(shared *sharedRepository, conf buffer.ConfigFileBuffer) (*buffer.TenantBufferManager[semaphoreReleaseOpts, pgtype.UUID], error) {
	eventBufOpts := buffer.TenantBufManagerOpts[semaphoreReleaseOpts, pgtype.UUID]{
		Name:       "semaphore_releaser",
		OutputFunc: shared.bulkReleaseSemaphores,
		SizeFunc:   sizeOfSemaphoreReleaseData,
		L:          shared.l,
		V:          shared.v,
		Config:     conf,
	}

	manager, err := buffer.NewTenantBufManager(eventBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func sizeOfSemaphoreReleaseData(item semaphoreReleaseOpts) int {
	return len(item.StepRunId.Bytes) + len(item.TenantId.Bytes)
}

func sortForSemaphoreRelease(opts []semaphoreReleaseOpts) []semaphoreReleaseOpts {
	sort.SliceStable(opts, func(i, j int) bool {
		return sqlchelpers.UUIDToStr(opts[i].StepRunId) < sqlchelpers.UUIDToStr(opts[j].StepRunId)
	})

	return opts
}

type semaphoreReleaseOpts struct {
	StepRunId pgtype.UUID
	TenantId  pgtype.UUID
}

func (w *sharedRepository) bulkReleaseSemaphores(ctx context.Context, opts []semaphoreReleaseOpts) ([]*pgtype.UUID, error) {
	orderedOpts := sortForSemaphoreRelease(opts)

	res := make([]*pgtype.UUID, 0, len(orderedOpts))

	for _, o := range orderedOpts {
		srId := o.StepRunId
		res = append(res, &srId)
	}

	stepRunIds := make([]pgtype.UUID, 0, len(orderedOpts))
	tenantIds := make([]pgtype.UUID, 0, len(orderedOpts))

	for _, o := range orderedOpts {
		stepRunIds = append(stepRunIds, o.StepRunId)
		tenantIds = append(tenantIds, o.TenantId)
	}

	verifiedStepRunIds, err := w.queries.VerifiedStepRunTenantIds(ctx, w.pool, dbsqlc.VerifiedStepRunTenantIdsParams{
		Steprunids: stepRunIds,
		Tenantids:  tenantIds,
	})

	if err != nil {
		return nil, err
	}

	err = sqlchelpers.DeadlockRetry(w.l, func() (err error) {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 5000)

		if err != nil {
			return err
		}

		defer rollback()

		err = w.queries.UpdateStepRunUnsetWorkerIdBulk(ctx, tx, verifiedStepRunIds)

		if err != nil {
			return err
		}

		return commit(ctx)
	})

	if err != nil {
		return nil, err
	}

	err = sqlchelpers.DeadlockRetry(w.l, func() (err error) {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 5000)

		if err != nil {
			return err
		}

		defer rollback()

		err = w.queries.RemoveTimeoutQueueItems(ctx, tx, verifiedStepRunIds)

		if err != nil {
			return err
		}

		return commit(ctx)
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}
