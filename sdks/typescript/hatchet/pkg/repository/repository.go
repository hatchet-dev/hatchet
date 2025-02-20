package repository

import (
	"github.com/rs/zerolog"
)

type APIRepository interface {
	Health() HealthRepository
	APIToken() APITokenRepository
	Event() EventAPIRepository
	Log() LogsAPIRepository
	Tenant() TenantAPIRepository
	TenantAlertingSettings() TenantAlertingAPIRepository
	TenantInvite() TenantInviteRepository
	Workflow() WorkflowAPIRepository
	WorkflowRun() WorkflowRunAPIRepository
	JobRun() JobRunAPIRepository
	StepRun() StepRunAPIRepository
	Slack() SlackRepository
	SNS() SNSRepository
	Step() StepRepository
	Worker() WorkerAPIRepository
	UserSession() UserSessionRepository
	User() UserRepository
	SecurityCheck() SecurityCheckRepository
	WebhookWorker() WebhookWorkerRepository
}

type EngineRepository interface {
	Health() HealthRepository
	APIToken() EngineTokenRepository
	Dispatcher() DispatcherEngineRepository
	Event() EventEngineRepository
	GetGroupKeyRun() GetGroupKeyRunEngineRepository
	JobRun() JobRunEngineRepository
	StepRun() StepRunEngineRepository
	Step() StepRepository
	Tenant() TenantEngineRepository
	TenantAlertingSettings() TenantAlertingEngineRepository
	Ticker() TickerEngineRepository
	Worker() WorkerEngineRepository
	Workflow() WorkflowEngineRepository
	WorkflowRun() WorkflowRunEngineRepository
	StreamEvent() StreamEventsEngineRepository
	Log() LogsEngineRepository
	RateLimit() RateLimitEngineRepository
	WebhookWorker() WebhookWorkerEngineRepository
	Scheduler() SchedulerRepository
	MessageQueue() MessageQueueRepository
}

type EntitlementsRepository interface {
	TenantLimit() TenantLimitRepository
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

type TenantScopedCallback[T any] func(string, T) error

func (c TenantScopedCallback[T]) Do(l *zerolog.Logger, tenantId string, v T) {
	// wrap in panic recover to avoid panics in the callback
	defer func() {
		if r := recover(); r != nil {
			if l != nil {
				l.Error().Interface("panic", r).Msg("panic in callback")
			}
		}
	}()

	go func() {
		err := c(tenantId, v)

		if err != nil {
			l.Error().Err(err).Msg("callback failed")
		}
	}()
}

type UnscopedCallback[T any] func(T) error

func (c UnscopedCallback[T]) Do(l *zerolog.Logger, v T) {
	// wrap in panic recover to avoid panics in the callback
	defer func() {
		if r := recover(); r != nil {
			if l != nil {
				l.Error().Interface("panic", r).Msg("panic in callback")
			}
		}
	}()

	go func() {
		err := c(v)

		if err != nil {
			l.Error().Err(err).Msg("callback failed")
		}
	}()
}

type CallbackOptFunc[T any] func(*TenantCallbackOpts[T])

func WithPreCommitCallback[T any](cb TenantScopedCallback[T]) CallbackOptFunc[T] {
	return func(opts *TenantCallbackOpts[T]) {
		opts.cbs = append(opts.cbs, cb)
	}
}

func WithPostCommitCallback[T any](cb TenantScopedCallback[T]) CallbackOptFunc[T] {
	return func(opts *TenantCallbackOpts[T]) {
		opts.cbs = append(opts.cbs, cb)
	}
}

func RunPreCommit[T any](l *zerolog.Logger, tenantId string, v T, opts []CallbackOptFunc[T]) {
	// initialize the opts
	o := &TenantCallbackOpts[T]{
		cbs: make([]TenantScopedCallback[T], 0),
	}

	// apply the opts
	for _, opt := range opts {
		opt(o)
	}

	o.Run(l, tenantId, v)
}

func RunPostCommit[T any](l *zerolog.Logger, tenantId string, v T, opts []CallbackOptFunc[T]) {
	// initialize the opts
	o := &TenantCallbackOpts[T]{
		cbs: make([]TenantScopedCallback[T], 0),
	}

	// apply the opts
	for _, opt := range opts {
		opt(o)
	}

	o.Run(l, tenantId, v)
}

type TenantCallbackOpts[T any] struct {
	cbs []TenantScopedCallback[T]
}

func (o *TenantCallbackOpts[T]) Run(l *zerolog.Logger, tenantId string, v T) {
	for _, cb := range o.cbs {
		cb.Do(l, tenantId, v)
	}
}
