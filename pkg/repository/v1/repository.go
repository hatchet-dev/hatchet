package v1

import (
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type TaskOperationLimits struct {
	TimeoutLimit      int
	ReassignLimit     int
	RetryQueueLimit   int
	DurableSleepLimit int
}

type Repository interface {
	Triggers() TriggerRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
	Matches() MatchRepository
	OLAP() OLAPRepository
	OverwriteOLAPRepository(o OLAPRepository)
	Logs() LogLineRepository
	OverwriteLogsRepository(l LogLineRepository)
	Payloads() PayloadStoreRepository
	OverwriteExternalPayloadStore(o ExternalStore, nativeStoreTTL time.Duration)
	Workers() WorkerRepository
	Workflows() WorkflowRepository
	Ticker() TickerRepository
	Filters() FilterRepository
	Webhooks() WebhookRepository
	Idempotency() IdempotencyRepository
	IntervalSettings() IntervalSettingsRepository
	PGHealth() PGHealthRepository
}

type repositoryImpl struct {
	triggers     TriggerRepository
	tasks        TaskRepository
	scheduler    SchedulerRepository
	matches      MatchRepository
	olap         OLAPRepository
	logs         LogLineRepository
	workers      WorkerRepository
	workflows    WorkflowRepository
	ticker       TickerRepository
	filters      FilterRepository
	webhooks     WebhookRepository
	payloadStore PayloadStoreRepository
	idempotency  IdempotencyRepository
	intervals    IntervalSettingsRepository
	pgHealth     PGHealthRepository
}

func NewRepository(pool *pgxpool.Pool, l *zerolog.Logger, taskRetentionPeriod, olapRetentionPeriod time.Duration, maxInternalRetryCount int32, entitlements repository.EntitlementsRepository, taskLimits TaskOperationLimits, payloadStoreOpts PayloadStoreRepositoryOpts) (Repository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l, entitlements, payloadStoreOpts)

	impl := &repositoryImpl{
		triggers:     newTriggerRepository(shared),
		tasks:        newTaskRepository(shared, taskRetentionPeriod, maxInternalRetryCount, taskLimits.TimeoutLimit, taskLimits.ReassignLimit, taskLimits.RetryQueueLimit, taskLimits.DurableSleepLimit),
		scheduler:    newSchedulerRepository(shared),
		matches:      newMatchRepository(shared),
		olap:         newOLAPRepository(shared, olapRetentionPeriod, true),
		logs:         newLogLineRepository(shared),
		workers:      newWorkerRepository(shared),
		workflows:    newWorkflowRepository(shared),
		ticker:       newTickerRepository(shared),
		filters:      newFilterRepository(shared),
		webhooks:     newWebhookRepository(shared),
		payloadStore: shared.payloadStore,
		idempotency:  newIdempotencyRepository(shared),
		intervals:    newIntervalSettingsRepository(shared),
		pgHealth:     newPGHealthRepository(shared),
	}

	return impl, func() error {
		return cleanupShared()
	}
}

func (r *repositoryImpl) Triggers() TriggerRepository {
	return r.triggers
}

func (r *repositoryImpl) Tasks() TaskRepository {
	return r.tasks
}

func (r *repositoryImpl) Scheduler() SchedulerRepository {
	return r.scheduler
}

func (r *repositoryImpl) Matches() MatchRepository {
	return r.matches
}

func (r *repositoryImpl) OLAP() OLAPRepository {
	return r.olap
}

func (r *repositoryImpl) OverwriteOLAPRepository(o OLAPRepository) {
	r.olap = o
}

func (r *repositoryImpl) Logs() LogLineRepository {
	return r.logs
}

func (r *repositoryImpl) OverwriteLogsRepository(l LogLineRepository) {
	r.logs = l
}

func (r *repositoryImpl) Payloads() PayloadStoreRepository {
	return r.payloadStore
}

func (r *repositoryImpl) OverwriteExternalPayloadStore(o ExternalStore, nativeStoreTTL time.Duration) {
	r.payloadStore.OverwriteExternalStore(o, nativeStoreTTL)
}

func (r *repositoryImpl) Workers() WorkerRepository {
	return r.workers
}

func (r *repositoryImpl) Workflows() WorkflowRepository {
	return r.workflows
}

func (r *repositoryImpl) Ticker() TickerRepository {
	return r.ticker
}

func (r *repositoryImpl) Filters() FilterRepository {
	return r.filters
}

func (r *repositoryImpl) Webhooks() WebhookRepository {
	return r.webhooks
}

func (r *repositoryImpl) Idempotency() IdempotencyRepository {
	return r.idempotency
}

func (r *repositoryImpl) IntervalSettings() IntervalSettingsRepository {
	return r.intervals
}

func (r *repositoryImpl) PGHealth() PGHealthRepository {
	return r.pgHealth
}
