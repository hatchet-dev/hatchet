package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type tickerRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewTickerRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.TickerEngineRepository {
	queries := dbsqlc.New()

	return &tickerRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (t *tickerRepository) CreateNewTicker(opts *repository.CreateTickerOpts) (*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.CreateTicker(context.Background(), t.pool, sqlchelpers.UUIDFromStr(opts.ID))
}

func (t *tickerRepository) UpdateTicker(tickerId string, opts *repository.UpdateTickerOpts) (*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.UpdateTicker(
		context.Background(),
		t.pool,
		dbsqlc.UpdateTickerParams{
			ID:              sqlchelpers.UUIDFromStr(tickerId),
			LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
		},
	)
}

func (t *tickerRepository) ListTickers(opts *repository.ListTickerOpts) ([]*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.ListTickersParams{}

	if opts.LatestHeartbeatAfter != nil {
		params.LastHeartbeatAfter = sqlchelpers.TimestampFromTime(opts.LatestHeartbeatAfter.UTC())
	}

	if opts.Active != nil {
		params.IsActive = *opts.Active
	}

	return t.queries.ListTickers(
		context.Background(),
		t.pool,
		params,
	)
}

func (t *tickerRepository) Delete(tickerId string) error {
	_, err := t.queries.DeleteTicker(
		context.Background(),
		t.pool,
		sqlchelpers.UUIDFromStr(tickerId),
	)

	return err
}

func (t *tickerRepository) PollStepRuns(tickerId string) ([]*dbsqlc.StepRun, error) {
	return t.queries.PollStepRuns(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollGetGroupKeyRuns(tickerId string) ([]*dbsqlc.GetGroupKeyRun, error) {
	return t.queries.PollGetGroupKeyRuns(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollCronSchedules(tickerId string) ([]*dbsqlc.PollCronSchedulesRow, error) {
	return t.queries.PollCronSchedules(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollScheduledWorkflows(tickerId string) ([]*dbsqlc.PollScheduledWorkflowsRow, error) {
	return t.queries.PollScheduledWorkflows(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

// func (t *tickerRepository) AddJobRun(tickerId string, jobRun *db.JobRunModel) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.JobRuns.Link(
// 			db.JobRun.ID.Equals(jobRun.ID),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) AddStepRun(tickerId, stepRunId string) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.StepRuns.Link(
// 			db.StepRun.ID.Equals(stepRunId),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) AddGetGroupKeyRun(tickerId, getGroupKeyRunId string) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.GroupKeyRuns.Link(
// 			db.GetGroupKeyRun.ID.Equals(getGroupKeyRunId),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) AddCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.Crons.Link(
// 			db.WorkflowTriggerCronRef.ParentIDCron(
// 				db.WorkflowTriggerCronRef.ParentID.Equals(cron.ParentID),
// 				db.WorkflowTriggerCronRef.Cron.Equals(cron.Cron),
// 			),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) RemoveCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.Crons.Unlink(
// 			db.WorkflowTriggerCronRef.ParentIDCron(
// 				db.WorkflowTriggerCronRef.ParentID.Equals(cron.ParentID),
// 				db.WorkflowTriggerCronRef.Cron.Equals(cron.Cron),
// 			),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) AddScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.Scheduled.Link(
// 			db.WorkflowTriggerScheduledRef.ID.Equals(schedule.ID),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) RemoveScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).Update(
// 		db.Ticker.Scheduled.Unlink(
// 			db.WorkflowTriggerScheduledRef.ID.Equals(schedule.ID),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) GetTickerById(tickerId string) (*db.TickerModel, error) {
// 	return t.client.Ticker.FindUnique(
// 		db.Ticker.ID.Equals(tickerId),
// 	).With(
// 		db.Ticker.Crons.Fetch().With(
// 			db.WorkflowTriggerCronRef.Parent.Fetch().With(
// 				db.WorkflowTriggers.Workflow.Fetch().With(
// 					db.WorkflowVersion.Workflow.Fetch(),
// 				),
// 			),
// 		),
// 		db.Ticker.Scheduled.Fetch().With(
// 			db.WorkflowTriggerScheduledRef.Parent.Fetch().With(
// 				db.WorkflowVersion.Workflow.Fetch(),
// 			),
// 		),
// 	).Exec(context.Background())
// }

// func (t *tickerRepository) UpdateStaleTickers(onStale func(tickerId string, getValidTickerId func() string) error) error {
// 	tx, err := t.pool.Begin(context.Background())

// 	if err != nil {
// 		return err
// 	}

// 	defer deferRollback(context.Background(), t.l, tx.Rollback)

// 	staleTickers, err := t.queries.ListNewlyStaleTickers(context.Background(), tx)

// 	if err != nil {
// 		return err
// 	}

// 	activeTickers, err := t.queries.ListActiveTickers(context.Background(), tx)

// 	if err != nil {
// 		return err
// 	}

// 	// if there are no active tickers, we can't reassign the stale tickers
// 	if len(activeTickers) == 0 {
// 		return nil
// 	}

// 	tickersToDelete := make([]pgtype.UUID, 0)

// 	for i, ticker := range staleTickers {
// 		err := onStale(sqlchelpers.UUIDToStr(ticker.Ticker.ID), func() string {
// 			// assign tickers in round-robin fashion
// 			return sqlchelpers.UUIDToStr(activeTickers[i%len(activeTickers)].Ticker.ID)
// 		})

// 		if err != nil {
// 			return err
// 		}

// 		tickersToDelete = append(tickersToDelete, ticker.Ticker.ID)
// 	}

// 	_, err = t.queries.SetTickersInactive(context.Background(), tx, tickersToDelete)

// 	if err != nil {
// 		return err
// 	}

// 	return tx.Commit(context.Background())
// }
