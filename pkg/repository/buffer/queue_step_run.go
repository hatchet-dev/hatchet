package buffer

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type BulkStepRunQueuer struct {
	*TenantBufferManager[BulkQueueStepRunOpts, pgtype.UUID]

	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewBulkStepRunQueuer(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) (*BulkStepRunQueuer, error) {
	queries := dbsqlc.New()
	flushPeriod := 20 * time.Millisecond

	w := &BulkStepRunQueuer{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}

	eventBufOpts := TenantBufManagerOpts[BulkQueueStepRunOpts, pgtype.UUID]{
		Name:                "step_run_queuer",
		OutputFunc:          w.BulkQueueStepRuns,
		SizeFunc:            sizeOfQueueData,
		L:                   w.l,
		V:                   w.v,
		FlushPeriod:         &flushPeriod,
		FlushItemsThreshold: 1000,
	}

	manager, err := NewTenantBufManager(eventBufOpts)

	if err != nil {
		l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	w.TenantBufferManager = manager

	return w, nil
}

func (w *BulkStepRunQueuer) Cleanup() error {
	return w.TenantBufferManager.Cleanup()
}

func sizeOfQueueData(item BulkQueueStepRunOpts) int {
	return len(item.GetStepRunForEngineRow.SRID.Bytes) + len(item.Input)
}

func sortForQueueStepRuns(opts []BulkQueueStepRunOpts) []BulkQueueStepRunOpts {
	sort.SliceStable(opts, func(i, j int) bool {
		return sqlchelpers.UUIDToStr(opts[i].GetStepRunForEngineRow.SRID) < sqlchelpers.UUIDToStr(opts[j].GetStepRunForEngineRow.SRID)
	})

	return opts
}

type BulkQueueStepRunOpts struct {
	*dbsqlc.GetStepRunForEngineRow

	Priority int
	IsRetry  bool
	Input    []byte
}

func (w *BulkStepRunQueuer) BulkQueueStepRuns(ctx context.Context, opts []BulkQueueStepRunOpts) ([]pgtype.UUID, error) {
	res := make([]pgtype.UUID, 0, len(opts))
	orderedOpts := sortForQueueStepRuns(opts)

	err := sqlchelpers.DeadlockRetry(w.l, func() (err error) {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 10000)

		if err != nil {
			return err
		}

		defer rollback()

		// we have to split into step runs with input versus without input, as the bulk insert doesn't
		// allow for null values
		inputs := make([][]byte, 0, len(orderedOpts))
		stepRunIdWithInputs := make([]pgtype.UUID, 0, len(orderedOpts))
		stepRunIdWithoutInputs := make([]pgtype.UUID, 0, len(orderedOpts))
		retryCountsWithInputs := make([]int32, 0, len(orderedOpts))
		retryCountsWithoutInputs := make([]int32, 0, len(orderedOpts))
		res = make([]pgtype.UUID, 0, len(orderedOpts))

		for _, o := range orderedOpts {
			res = append(res, o.GetStepRunForEngineRow.SRID)

			var retryCount int32

			if o.IsRetry {
				retryCount = o.GetStepRunForEngineRow.SRRetryCount + 1
			}

			if o.Input != nil {
				inputs = append(inputs, o.Input)
				stepRunIdWithInputs = append(stepRunIdWithInputs, o.GetStepRunForEngineRow.SRID)
				retryCountsWithInputs = append(retryCountsWithInputs, retryCount)
			} else {
				stepRunIdWithoutInputs = append(stepRunIdWithoutInputs, o.GetStepRunForEngineRow.SRID)
				retryCountsWithoutInputs = append(retryCountsWithoutInputs, retryCount)
			}
		}

		if len(stepRunIdWithInputs) > 0 {
			err = w.queries.QueueStepRunBulkWithInput(ctx, tx, dbsqlc.QueueStepRunBulkWithInputParams{
				Ids:         stepRunIdWithInputs,
				Inputs:      inputs,
				Retrycounts: retryCountsWithInputs,
			})

			if err != nil {
				return err
			}
		}

		if len(stepRunIdWithoutInputs) > 0 {
			err = w.queries.QueueStepRunBulkNoInput(ctx, tx, dbsqlc.QueueStepRunBulkNoInputParams{
				Ids:         stepRunIdWithoutInputs,
				Retrycounts: retryCountsWithoutInputs,
			})

			if err != nil {
				return err
			}
		}

		// next, insert the queue items
		params := make([]dbsqlc.CreateQueueItemsBulkParams, 0, len(orderedOpts))

		for _, o := range orderedOpts {
			innerStepRun := o.GetStepRunForEngineRow
			tenantId := o.GetStepRunForEngineRow.SRTenantId

			params = append(params, dbsqlc.CreateQueueItemsBulkParams{
				StepRunId:         innerStepRun.SRID,
				StepId:            innerStepRun.StepId,
				ActionId:          sqlchelpers.TextFromStr(innerStepRun.ActionId),
				StepTimeout:       innerStepRun.StepTimeout,
				TenantId:          tenantId,
				Queue:             innerStepRun.SRQueue,
				IsQueued:          true,
				Priority:          int32(o.Priority), // nolint: gosec
				Sticky:            innerStepRun.StickyStrategy,
				DesiredWorkerId:   innerStepRun.DesiredWorkerId,
				ScheduleTimeoutAt: getScheduleTimeout(innerStepRun),
			})
		}

		n, err := w.queries.CreateQueueItemsBulk(ctx, tx, params)

		if err != nil {
			return err
		}

		if int(n) != len(orderedOpts) {
			return fmt.Errorf("expected %d queue items to be inserted, but only %d were", len(orderedOpts), n)
		}

		return commit(ctx)
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func getScheduleTimeout(stepRun *dbsqlc.GetStepRunForEngineRow) pgtype.Timestamp {
	var timeoutDuration time.Duration

	// get the schedule timeout from the step
	stepScheduleTimeout := stepRun.StepScheduleTimeout

	if stepScheduleTimeout != "" {
		timeoutDuration, _ = time.ParseDuration(stepScheduleTimeout)
	} else {
		timeoutDuration = defaults.DefaultScheduleTimeout
	}

	timeout := time.Now().UTC().Add(timeoutDuration)

	return sqlchelpers.TimestampFromTime(timeout)
}