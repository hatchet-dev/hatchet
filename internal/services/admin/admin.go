package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	scheduler "github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.WorkflowServiceServer
	Cleanup() error
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	repov1    v1.Repository
	mqv1      msgqueue.MessageQueue
	pubsub    msgqueue.PubSub
	v         validator.Validator
	analytics analytics.Analytics

	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger

	tw                  *trigger.TriggerWriter
	pubBuffer           *msgqueue.MQPubBuffer
	grpcTriggersEnabled bool
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repov1                      v1.Repository
	mqv1                        msgqueue.MessageQueue
	pubsub                      msgqueue.PubSub
	v                           validator.Validator
	analytics                   analytics.Analytics
	localScheduler              *scheduler.Scheduler
	localDispatcher             *dispatcher.DispatcherImpl
	l                           *zerolog.Logger
	grpcTriggerSlots            int
	optimisticSchedulingEnabled bool
	grpcTriggersEnabled         bool
	promGate                    *prometheus.Gate
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()
	logger := logger.NewDefaultLogger("admin_service")

	return &AdminServiceOpts{
		v:         v,
		l:         &logger,
		analytics: analytics.NoOpAnalytics{},
	}
}

func WithRepositoryV1(r v1.Repository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repov1 = r
	}
}

func WithMessageQueueV1(mq msgqueue.MessageQueue) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.mqv1 = mq
	}
}

func WithPubSub(pubsub msgqueue.PubSub) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.pubsub = pubsub
	}
}

func WithValidator(v validator.Validator) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.v = v
	}
}

func WithLocalScheduler(s *scheduler.Scheduler) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.localScheduler = s
	}
}

func WithLocalDispatcher(d *dispatcher.DispatcherImpl) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.localDispatcher = d
	}
}

func WithOptimisticSchedulingEnabled(enabled bool) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.optimisticSchedulingEnabled = enabled
	}
}

func WithLogger(l *zerolog.Logger) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.l = l
	}
}

func WithGrpcTriggersEnabled(enabled bool) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.grpcTriggersEnabled = enabled
	}
}

func WithGrpcTriggerSlots(slots int) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.grpcTriggerSlots = slots
	}
}

func WithAnalytics(a analytics.Analytics) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.analytics = a
	}
}

func WithPrometheusGate(gate *prometheus.Gate) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.promGate = gate
	}
}

func NewAdminService(fs ...AdminServiceOpt) (AdminService, error) {
	opts := defaultAdminServiceOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	if opts.pubsub == nil {
		return nil, fmt.Errorf("pubsub is required. use WithPubSub")
	}

	slots := 0
	if opts.grpcTriggersEnabled {
		slots = opts.grpcTriggerSlots
	}

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mqv1)
	tw := trigger.NewTriggerWriter(opts.mqv1, opts.pubsub, opts.repov1, opts.l, pubBuffer, slots, opts.promGate)

	var localScheduler *scheduler.Scheduler

	if opts.optimisticSchedulingEnabled && opts.localScheduler != nil {
		localScheduler = opts.localScheduler
	}

	return &AdminServiceImpl{
		repov1:              opts.repov1,
		mqv1:                opts.mqv1,
		pubsub:              opts.pubsub,
		v:                   opts.v,
		analytics:           opts.analytics,
		localScheduler:      localScheduler,
		localDispatcher:     opts.localDispatcher,
		l:                   opts.l,
		tw:                  tw,
		pubBuffer:           pubBuffer,
		grpcTriggersEnabled: opts.grpcTriggersEnabled,
	}, nil
}

// Cleanup stops the pubBuffer goroutines if they exist
func (a *AdminServiceImpl) Cleanup() error {
	if a.pubBuffer != nil {
		a.pubBuffer.Stop()
	}
	return nil
}
