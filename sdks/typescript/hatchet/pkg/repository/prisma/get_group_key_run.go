package prisma

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type getGroupKeyRunRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewGetGroupKeyRunRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.GetGroupKeyRunEngineRepository {
	queries := dbsqlc.New()

	return &getGroupKeyRunRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (s *getGroupKeyRunRepository) ListGetGroupKeyRunsToRequeue(ctx context.Context, tenantId string) ([]*dbsqlc.GetGroupKeyRun, error) {
	return s.queries.ListGetGroupKeyRunsToRequeue(ctx, s.pool, sqlchelpers.UUIDFromStr(tenantId))
}

func (s *getGroupKeyRunRepository) ListGetGroupKeyRunsToReassign(ctx context.Context, tenantId string) ([]*dbsqlc.GetGroupKeyRun, error) {
	return s.queries.ListGetGroupKeyRunsToReassign(ctx, s.pool, sqlchelpers.UUIDFromStr(tenantId))
}

func (s *getGroupKeyRunRepository) AssignGetGroupKeyRunToWorker(ctx context.Context, tenantId, getGroupKeyRunId string) (workerId string, dispatcherId string, err error) {
	// var assigned
	var assigned *dbsqlc.AssignGetGroupKeyRunToWorkerRow

	err = sqlchelpers.DeadlockRetry(s.l, func() (err error) {
		assigned, err = s.queries.AssignGetGroupKeyRunToWorker(ctx, s.pool, dbsqlc.AssignGetGroupKeyRunToWorkerParams{
			Getgroupkeyrunid: sqlchelpers.UUIDFromStr(getGroupKeyRunId),
			Tenantid:         sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return repository.ErrNoWorkerAvailable
			}

			return err
		}

		return nil
	})

	if err != nil {
		return "", "", err
	}

	return sqlchelpers.UUIDToStr(assigned.WorkerId), sqlchelpers.UUIDToStr(assigned.DispatcherId), nil
}

func (s *getGroupKeyRunRepository) AssignGetGroupKeyRunToTicker(ctx context.Context, tenantId, getGroupKeyRunId string) (tickerId string, err error) {
	// var assigned
	var assigned *dbsqlc.AssignGetGroupKeyRunToTickerRow

	err = sqlchelpers.DeadlockRetry(s.l, func() (err error) {
		assigned, err = s.queries.AssignGetGroupKeyRunToTicker(ctx, s.pool, dbsqlc.AssignGetGroupKeyRunToTickerParams{
			Getgroupkeyrunid: sqlchelpers.UUIDFromStr(getGroupKeyRunId),
			Tenantid:         sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return repository.ErrNoWorkerAvailable
			}

			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return sqlchelpers.UUIDToStr(assigned.TickerId), nil
}

func (s *getGroupKeyRunRepository) UpdateGetGroupKeyRun(ctx context.Context, tenantId, getGroupKeyRunId string, opts *repository.UpdateGetGroupKeyRunOpts) (*dbsqlc.GetGroupKeyRunForEngineRow, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgGetGroupKeyRunId := sqlchelpers.UUIDFromStr(getGroupKeyRunId)

	updateParams := dbsqlc.UpdateGetGroupKeyRunParams{
		ID:       pgGetGroupKeyRunId,
		Tenantid: pgTenantId,
	}

	updateWorkflowRunParams := dbsqlc.UpdateWorkflowRunGroupKeyFromRunParams{
		Tenantid:      pgTenantId,
		Groupkeyrunid: sqlchelpers.UUIDFromStr(getGroupKeyRunId),
	}

	if opts.RequeueAfter != nil {
		updateParams.RequeueAfter = sqlchelpers.TimestampFromTime(*opts.RequeueAfter)
	}

	if opts.StartedAt != nil {
		updateParams.StartedAt = sqlchelpers.TimestampFromTime(*opts.StartedAt)
	}

	if opts.FinishedAt != nil {
		updateParams.FinishedAt = sqlchelpers.TimestampFromTime(*opts.FinishedAt)
	}

	if opts.Status != nil {
		runStatus := dbsqlc.NullStepRunStatus{}

		if err := runStatus.Scan(string(*opts.Status)); err != nil {
			return nil, err
		}

		updateParams.Status = runStatus
	}

	if opts.Output != nil {
		updateParams.Output = sqlchelpers.TextFromStr(*opts.Output)
	}

	if opts.Error != nil {
		updateParams.Error = sqlchelpers.TextFromStr(*opts.Error)
	}

	if opts.CancelledAt != nil {
		updateParams.CancelledAt = sqlchelpers.TimestampFromTime(*opts.CancelledAt)
	}

	if opts.CancelledReason != nil {
		updateParams.CancelledReason = sqlchelpers.TextFromStr(*opts.CancelledReason)
	}

	if opts.ScheduleTimeoutAt != nil {
		updateParams.ScheduleTimeoutAt = sqlchelpers.TimestampFromTime(*opts.ScheduleTimeoutAt)
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	res1, err := s.queries.UpdateGetGroupKeyRun(ctx, tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update get group key run: %w", err)
	}

	// only update workflow run if status or output has changed
	if opts.Status != nil || opts.Output != nil {
		_, err = s.queries.UpdateWorkflowRunGroupKeyFromRun(ctx, tx, updateWorkflowRunParams)

		if err != nil {
			return nil, fmt.Errorf("could not resolve workflow run status from get group key run: %w", err)
		}
	}

	getGroupKeyRuns, err := s.queries.GetGroupKeyRunForEngine(ctx, tx, dbsqlc.GetGroupKeyRunForEngineParams{
		Ids:      []pgtype.UUID{res1.ID},
		Tenantid: pgTenantId,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	// in this case, we've committed the update (so we can update timeouts), but we're hitting a case where the Workflow or
	// WorkflowRun has been deleted, so we return an error.
	if len(getGroupKeyRuns) == 0 {
		return nil, fmt.Errorf("could not find get group key run for engine")
	}

	return getGroupKeyRuns[0], nil
}

func (s *getGroupKeyRunRepository) GetGroupKeyRunForEngine(ctx context.Context, tenantId, getGroupKeyRunId string) (*dbsqlc.GetGroupKeyRunForEngineRow, error) {
	res, err := s.queries.GetGroupKeyRunForEngine(ctx, s.pool, dbsqlc.GetGroupKeyRunForEngineParams{
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(getGroupKeyRunId)},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("could not find group key run %s", getGroupKeyRunId)
	}

	return res[0], nil
}
