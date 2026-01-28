package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
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
	APIToken() APITokenRepository
	Dispatcher() DispatcherRepository
	Health() HealthRepository
	MessageQueue() MessageQueueRepository
	RateLimit() RateLimitRepository
	Triggers() TriggerRepository
	Tasks() TaskRepository
	Scheduler() SchedulerRepository
	Matches() MatchRepository
	OLAP() OLAPRepository
	OverwriteOLAPRepository(o OLAPRepository)
	Logs() LogLineRepository
	OverwriteLogsRepository(l LogLineRepository)
	Payloads() PayloadStoreRepository
	OverwriteExternalPayloadStore(o ExternalStore)
	Workers() WorkerRepository
	Workflows() WorkflowRepository
	Ticker() TickerRepository
	Filters() FilterRepository
	Webhooks() WebhookRepository
	Idempotency() IdempotencyRepository
	IntervalSettings() IntervalSettingsRepository
	PGHealth() PGHealthRepository
	SecurityCheck() SecurityCheckRepository
	Slack() SlackRepository
	SNS() SNSRepository
	TenantInvite() TenantInviteRepository
	TenantLimit() TenantLimitRepository
	TenantAlertingSettings() TenantAlertingRepository
	Tenant() TenantRepository
	User() UserRepository
	UserSession() UserSessionRepository
	WorkflowSchedules() WorkflowScheduleRepository
	OTelCollector() OTelCollectorRepository
	OverwriteOTelCollectorRepository(o OTelCollectorRepository)
}

type repositoryImpl struct {
	apiToken          APITokenRepository
	dispatcher        DispatcherRepository
	health            HealthRepository
	messageQueue      MessageQueueRepository
	rateLimit         RateLimitRepository
	triggers          TriggerRepository
	tasks             TaskRepository
	scheduler         SchedulerRepository
	matches           MatchRepository
	olap              OLAPRepository
	logs              LogLineRepository
	workers           WorkerRepository
	workflows         WorkflowRepository
	ticker            TickerRepository
	filters           FilterRepository
	webhooks          WebhookRepository
	payloadStore      PayloadStoreRepository
	idempotency       IdempotencyRepository
	intervals         IntervalSettingsRepository
	pgHealth          PGHealthRepository
	securityCheck     SecurityCheckRepository
	slack             SlackRepository
	sns               SNSRepository
	tenantInvite      TenantInviteRepository
	tenantLimit       TenantLimitRepository
	tenantAlerting    TenantAlertingRepository
	tenant            TenantRepository
	user              UserRepository
	userSession       UserSessionRepository
	workflowSchedules WorkflowScheduleRepository
	otelcol           OTelCollectorRepository
}

func NewRepository(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	cacheDuration time.Duration,
	taskRetentionPeriod, olapRetentionPeriod time.Duration,
	maxInternalRetryCount int32,
	taskLimits TaskOperationLimits,
	payloadStoreOpts PayloadStoreRepositoryOpts,
	statusUpdateBatchSizeLimits StatusUpdateBatchSizeLimits,
	tenantLimitConfig limits.LimitConfigFile,
	enforceLimits bool,
	enforceLimitsFunc func(ctx context.Context, tenantId string) (bool, error),
) (Repository, func() error) {
	v := validator.NewDefaultValidator()

	shared, cleanupShared := newSharedRepository(pool, v, l, payloadStoreOpts, tenantLimitConfig, enforceLimits, enforceLimitsFunc, cacheDuration)

	mq, cleanupMq := newMessageQueueRepository(shared)

	impl := &repositoryImpl{
		apiToken:          newAPITokenRepository(shared, cacheDuration),
		dispatcher:        newDispatcherRepository(shared),
		health:            newHealthRepository(shared),
		messageQueue:      mq,
		rateLimit:         newRateLimitRepository(shared),
		triggers:          newTriggerRepository(shared),
		tasks:             newTaskRepository(shared, taskRetentionPeriod, maxInternalRetryCount, taskLimits.TimeoutLimit, taskLimits.ReassignLimit, taskLimits.RetryQueueLimit, taskLimits.DurableSleepLimit),
		scheduler:         newSchedulerRepository(shared),
		matches:           newMatchRepository(shared),
		olap:              newOLAPRepository(shared, olapRetentionPeriod, true, statusUpdateBatchSizeLimits),
		logs:              newLogLineRepository(shared),
		workers:           newWorkerRepository(shared),
		workflows:         newWorkflowRepository(shared),
		ticker:            newTickerRepository(shared),
		filters:           newFilterRepository(shared),
		webhooks:          newWebhookRepository(shared),
		payloadStore:      shared.payloadStore,
		idempotency:       newIdempotencyRepository(shared),
		intervals:         newIntervalSettingsRepository(shared),
		pgHealth:          newPGHealthRepository(shared),
		securityCheck:     newSecurityCheckRepository(shared),
		slack:             newSlackRepository(shared),
		sns:               newSNSRepository(shared),
		tenantInvite:      newTenantInviteRepository(shared),
		tenantLimit:       newTenantLimitRepository(shared, tenantLimitConfig, enforceLimits, enforceLimitsFunc, cacheDuration),
		tenantAlerting:    newTenantAlertingRepository(shared, cacheDuration),
		tenant:            newTenantRepository(shared, cacheDuration),
		user:              newUserRepository(shared),
		userSession:       newUserSessionRepository(shared),
		workflowSchedules: newWorkflowScheduleRepository(shared),
		otelcol:           newOTelCollectorRepository(shared),
	}

	return impl, func() error {
		var multiErr error

		if err := cleanupMq(); err != nil {
			multiErr = fmt.Errorf("failed to cleanup message queue repository: %w", err)
		}

		if err := cleanupShared(); err != nil {
			multiErr = fmt.Errorf("failed to cleanup shared repository: %w", err)
		}

		return multiErr
	}
}

func (r *repositoryImpl) APIToken() APITokenRepository {
	return r.apiToken
}

func (r *repositoryImpl) Dispatcher() DispatcherRepository {
	return r.dispatcher
}

func (r *repositoryImpl) Health() HealthRepository {
	return r.health
}

func (r *repositoryImpl) Triggers() TriggerRepository {
	return r.triggers
}

func (r *repositoryImpl) MessageQueue() MessageQueueRepository {
	return r.messageQueue
}

func (r *repositoryImpl) RateLimit() RateLimitRepository {
	return r.rateLimit
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

func (r *repositoryImpl) OverwriteExternalPayloadStore(o ExternalStore) {
	r.payloadStore.OverwriteExternalStore(o)
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

func (r *repositoryImpl) SecurityCheck() SecurityCheckRepository {
	return r.securityCheck
}

func (r *repositoryImpl) Slack() SlackRepository {
	return r.slack
}

func (r *repositoryImpl) SNS() SNSRepository {
	return r.sns
}

func (r *repositoryImpl) TenantInvite() TenantInviteRepository {
	return r.tenantInvite
}

func (r *repositoryImpl) TenantLimit() TenantLimitRepository {
	return r.tenantLimit
}

func (r *repositoryImpl) TenantAlertingSettings() TenantAlertingRepository {
	return r.tenantAlerting
}

func (r *repositoryImpl) Tenant() TenantRepository {
	return r.tenant
}

func (r *repositoryImpl) User() UserRepository {
	return r.user
}

func (r *repositoryImpl) UserSession() UserSessionRepository {
	return r.userSession
}

func (r *repositoryImpl) WorkflowSchedules() WorkflowScheduleRepository {
	return r.workflowSchedules
}

func (r *repositoryImpl) OTelCollector() OTelCollectorRepository {
	return r.otelcol
}

func (r *repositoryImpl) OverwriteOTelCollectorRepository(o OTelCollectorRepository) {
	r.otelcol = o
}
