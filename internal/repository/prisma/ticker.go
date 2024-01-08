package prisma

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type tickerRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewTickerRepository(client *db.PrismaClient, v validator.Validator) repository.TickerRepository {
	return &tickerRepository{
		client: client,
		v:      v,
	}
}

func (t *tickerRepository) CreateNewTicker(opts *repository.CreateTickerOpts) (*db.TickerModel, error) {
	return t.client.Ticker.CreateOne(
		db.Ticker.ID.Set(opts.ID),
		db.Ticker.LastHeartbeatAt.Set(time.Now().UTC()),
	).Exec(context.Background())
}

func (t *tickerRepository) UpdateTicker(tickerId string, opts *repository.UpdateTickerOpts) (*db.TickerModel, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.LastHeartbeatAt.SetIfPresent(opts.LastHeartbeatAt),
	).Exec(context.Background())
}

func (t *tickerRepository) ListTickers(opts *repository.ListTickerOpts) ([]db.TickerModel, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.TickerWhereParam{}

	if opts.LatestHeartbeatAt != nil {
		params = append(params, db.Ticker.LastHeartbeatAt.Gt(*opts.LatestHeartbeatAt))
	}

	return t.client.Ticker.FindMany(
		params...,
	).Exec(context.Background())
}

func (t *tickerRepository) Delete(tickerId string) error {
	_, err := t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Delete().Exec(context.Background())

	return err
}

func (t *tickerRepository) AddJobRun(tickerId string, jobRun *db.JobRunModel) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.JobRuns.Link(
			db.JobRun.ID.Equals(jobRun.ID),
		),
	).Exec(context.Background())
}

func (t *tickerRepository) AddStepRun(tickerId, stepRunId string) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.StepRuns.Link(
			db.StepRun.ID.Equals(stepRunId),
		),
	).Exec(context.Background())
}

func (t *tickerRepository) AddCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.Crons.Link(
			db.WorkflowTriggerCronRef.ParentIDCron(
				db.WorkflowTriggerCronRef.ParentID.Equals(cron.ParentID),
				db.WorkflowTriggerCronRef.Cron.Equals(cron.Cron),
			),
		),
	).Exec(context.Background())
}

func (t *tickerRepository) RemoveCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.Crons.Unlink(
			db.WorkflowTriggerCronRef.ParentIDCron(
				db.WorkflowTriggerCronRef.ParentID.Equals(cron.ParentID),
				db.WorkflowTriggerCronRef.Cron.Equals(cron.Cron),
			),
		),
	).Exec(context.Background())
}

func (t *tickerRepository) AddScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.Scheduled.Link(
			db.WorkflowTriggerScheduledRef.ID.Equals(schedule.ID),
		),
	).Exec(context.Background())
}

func (t *tickerRepository) RemoveScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error) {
	return t.client.Ticker.FindUnique(
		db.Ticker.ID.Equals(tickerId),
	).Update(
		db.Ticker.Scheduled.Unlink(
			db.WorkflowTriggerScheduledRef.ID.Equals(schedule.ID),
		),
	).Exec(context.Background())
}
