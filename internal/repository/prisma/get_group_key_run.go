package prisma

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type getGroupKeyRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewGetGroupKeyRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.GetGroupKeyRunRepository {
	queries := dbsqlc.New()

	return &getGroupKeyRunRepository{
		client:  client,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (s *getGroupKeyRunRepository) ListGetGroupKeyRuns(tenantId string, opts *repository.ListGetGroupKeyRunsOpts) ([]db.GetGroupKeyRunModel, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.GetGroupKeyRunWhereParam{
		db.GetGroupKeyRun.TenantID.Equals(tenantId),
	}

	if opts.Status != nil {
		params = append(params, db.GetGroupKeyRun.Status.Equals(*opts.Status))
	}

	return s.client.GetGroupKeyRun.FindMany(
		params...,
	).With(
		db.GetGroupKeyRun.Ticker.Fetch(),
		db.GetGroupKeyRun.WorkflowRun.Fetch().With(
			db.WorkflowRun.WorkflowVersion.Fetch().With(
				db.WorkflowVersion.Concurrency.Fetch().With(
					db.WorkflowConcurrency.GetConcurrencyGroup.Fetch(),
				),
			),
		),
	).Exec(context.Background())
}

func (s *getGroupKeyRunRepository) ListGetGroupKeyRunsToRequeue(tenantId string) ([]*dbsqlc.GetGroupKeyRun, error) {
	return s.queries.ListGetGroupKeyRunsToRequeue(context.Background(), s.pool, sqlchelpers.UUIDFromStr(tenantId))
}

func (s *getGroupKeyRunRepository) UpdateGetGroupKeyRun(tenantId, getGroupKeyRunId string, opts *repository.UpdateGetGroupKeyRunOpts) (*db.GetGroupKeyRunModel, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgGetGroupKeyRunId := sqlchelpers.UUIDFromStr(getGroupKeyRunId)

	updateParams := dbsqlc.UpdateGetGroupKeyRunParams{
		ID:       pgGetGroupKeyRunId,
		Tenantid: pgTenantId,
	}

	updateWorkflowRunParams := dbsqlc.UpdateWorkflowRunGroupKeyParams{
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

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	_, err = s.queries.UpdateGetGroupKeyRun(context.Background(), tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update get group key run: %w", err)
	}

	// only update workflow run if status or output has changed
	if opts.Status != nil || opts.Output != nil {
		_, err = s.queries.UpdateWorkflowRunGroupKey(context.Background(), tx, updateWorkflowRunParams)

		if err != nil {
			return nil, fmt.Errorf("could not resolve workflow run status from get group key run: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return s.client.GetGroupKeyRun.FindUnique(
		db.GetGroupKeyRun.ID.Equals(getGroupKeyRunId),
	).With(
		db.GetGroupKeyRun.Ticker.Fetch(),
		db.GetGroupKeyRun.WorkflowRun.Fetch().With(
			db.WorkflowRun.WorkflowVersion.Fetch().With(
				db.WorkflowVersion.Concurrency.Fetch().With(
					db.WorkflowConcurrency.GetConcurrencyGroup.Fetch(),
				),
			),
		),
	).Exec(context.Background())
}

func (s *getGroupKeyRunRepository) GetGroupKeyRunById(tenantId, getGroupKeyRunId string) (*db.GetGroupKeyRunModel, error) {
	return s.client.GetGroupKeyRun.FindUnique(
		db.GetGroupKeyRun.ID.Equals(getGroupKeyRunId),
	).With(
		db.GetGroupKeyRun.Ticker.Fetch(),
		db.GetGroupKeyRun.WorkflowRun.Fetch().With(
			db.WorkflowRun.WorkflowVersion.Fetch().With(
				db.WorkflowVersion.Concurrency.Fetch().With(
					db.WorkflowConcurrency.GetConcurrencyGroup.Fetch(),
				),
			),
		),
	).Exec(context.Background())
}
