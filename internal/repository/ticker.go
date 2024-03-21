package repository

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CreateTickerOpts struct {
	ID string `validate:"required,uuid"`
}

type UpdateTickerOpts struct {
	LastHeartbeatAt *time.Time
}

type ListTickerOpts struct {
	// Set this to only return tickers whose heartbeat is more recent than this time
	LatestHeartbeatAfter *time.Time

	Active *bool
}

type TickerEngineRepository interface {
	// CreateNewTicker creates a new ticker.
	CreateNewTicker(opts *CreateTickerOpts) (*dbsqlc.Ticker, error)

	// UpdateTicker updates a ticker.
	UpdateTicker(tickerId string, opts *UpdateTickerOpts) (*dbsqlc.Ticker, error)

	// ListTickers lists tickers.
	ListTickers(opts *ListTickerOpts) ([]*dbsqlc.Ticker, error)

	// Delete deletes a ticker.
	Delete(tickerId string) error

	// PollStepRuns looks for step runs who are close to past their timeoutAt value and are in a running state
	PollStepRuns(tickerId string) ([]*dbsqlc.StepRun, error)

	// PollJobRuns looks for get group key runs who are close to past their timeoutAt value and are in a running state
	PollGetGroupKeyRuns(tickerId string) ([]*dbsqlc.GetGroupKeyRun, error)

	// PollCronSchedules returns all cron schedules which should be managed by the ticker
	PollCronSchedules(tickerId string) ([]*dbsqlc.PollCronSchedulesRow, error)

	PollScheduledWorkflows(tickerId string) ([]*dbsqlc.PollScheduledWorkflowsRow, error)

	// // AddJobRun assigns a job run to a ticker.
	// AddJobRun(tickerId string, jobRun *db.JobRunModel) (*db.TickerModel, error)

	// // AddStepRun assigns a step run to a ticker.
	// AddStepRun(tickerId, stepRunId string) (*db.TickerModel, error)

	// // AddGetGroupKeyRun assigns a get group key run to a ticker.
	// AddGetGroupKeyRun(tickerId, getGroupKeyRunId string) (*db.TickerModel, error)

	// // AddCron assigns a cron to a ticker.
	// AddCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error)

	// // RemoveCron removes a cron from a ticker.
	// RemoveCron(tickerId string, cron *db.WorkflowTriggerCronRefModel) (*db.TickerModel, error)

	// // AddScheduledWorkflow assigns a scheduled workflow to a ticker.
	// AddScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error)

	// // RemoveScheduledWorkflow removes a scheduled workflow from a ticker.
	// RemoveScheduledWorkflow(tickerId string, schedule *db.WorkflowTriggerScheduledRefModel) (*db.TickerModel, error)

	// UpdateStaleTickers(onStale func(tickerId string, getValidTickerId func() string) error) error
}
