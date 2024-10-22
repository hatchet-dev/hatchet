package buffer

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type BulkSemaphoreReleaser struct {
	*TenantBufferManager[SemaphoreReleaseOpts, pgtype.UUID]

	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewBulkSemaphoreReleaser(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, conf ConfigFileBuffer) (*BulkSemaphoreReleaser, error) {
	queries := dbsqlc.New()

	w := &BulkSemaphoreReleaser{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}

	eventBufOpts := TenantBufManagerOpts[SemaphoreReleaseOpts, pgtype.UUID]{
		Name:       "semaphore_releaser",
		OutputFunc: w.BulkReleaseSemaphores,
		SizeFunc:   sizeOfData,
		L:          w.l,
		V:          w.v,
		Config:     conf,
	}

	manager, err := NewTenantBufManager(eventBufOpts)

	if err != nil {
		l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	w.TenantBufferManager = manager

	return w, nil
}

func (w *BulkSemaphoreReleaser) Cleanup() error {
	return w.TenantBufferManager.Cleanup()
}

func sizeOfData(item SemaphoreReleaseOpts) int {
	return len(item.StepRunId.Bytes) + len(item.TenantId.Bytes)
}

func sortForSemaphoreRelease(opts []SemaphoreReleaseOpts) []SemaphoreReleaseOpts {
	sort.SliceStable(opts, func(i, j int) bool {
		return sqlchelpers.UUIDToStr(opts[i].StepRunId) < sqlchelpers.UUIDToStr(opts[j].StepRunId)
	})

	return opts
}

type SemaphoreReleaseOpts struct {
	StepRunId pgtype.UUID
	TenantId  pgtype.UUID
}

func (w *BulkSemaphoreReleaser) BulkReleaseSemaphores(ctx context.Context, opts []SemaphoreReleaseOpts) ([]pgtype.UUID, error) {
	res := make([]pgtype.UUID, 0, len(opts))

	for _, o := range opts {
		res = append(res, o.StepRunId)
	}

	orderedOpts := sortForSemaphoreRelease(opts)

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
