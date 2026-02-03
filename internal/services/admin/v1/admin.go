package v1

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"

	"github.com/rs/zerolog"
)

type AdminService interface {
	contracts.AdminServiceServer
	Cleanup() error
}

type AdminServiceImpl struct {
	contracts.UnimplementedAdminServiceServer

	repo      v1.Repository
	mq        msgqueue.MessageQueue
	v         validator.Validator
	analytics analytics.Analytics

	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger

	tw        *trigger.TriggerWriter
	pubBuffer *msgqueue.MQPubBuffer
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repo      v1.Repository
	mq        msgqueue.MessageQueue
	v         validator.Validator
	analytics analytics.Analytics

	localScheduler              *scheduler.Scheduler
	localDispatcher             *dispatcher.DispatcherImpl
	l                           *zerolog.Logger
	optimisticSchedulingEnabled bool

	grpcTriggersEnabled bool
	grpcTriggerSlots    int
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()
	logger := logger.NewDefaultLogger("v1_admin_service")

	return &AdminServiceOpts{
		v: v,
		l: &logger,
	}
}

func WithRepository(r v1.Repository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repo = r
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.mq = mq
	}
}

func WithValidator(v validator.Validator) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.v = v
	}
}

func WithAnalytics(a analytics.Analytics) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.analytics = a
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

func NewAdminService(fs ...AdminServiceOpt) (AdminService, error) {
	opts := defaultAdminServiceOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	var tw *trigger.TriggerWriter
	var pubBuffer *msgqueue.MQPubBuffer

	if opts.grpcTriggersEnabled {
		pubBuffer = msgqueue.NewMQPubBuffer(opts.mq)

		tw = trigger.NewTriggerWriter(opts.mq, opts.repo, opts.l, pubBuffer, opts.grpcTriggerSlots)
	}

	var localScheduler *scheduler.Scheduler

	if opts.optimisticSchedulingEnabled && opts.localScheduler != nil {
		localScheduler = opts.localScheduler
	} else if opts.optimisticSchedulingEnabled && opts.localScheduler == nil {
		return nil, fmt.Errorf("optimistic writes enabled but no local scheduler provided")
	}

	return &AdminServiceImpl{
		repo:            opts.repo,
		mq:              opts.mq,
		v:               opts.v,
		analytics:       opts.analytics,
		localScheduler:  localScheduler,
		localDispatcher: opts.localDispatcher,
		l:               opts.l,
		tw:              tw,
		pubBuffer:       pubBuffer,
	}, nil
}

// Cleanup stops the pubBuffer goroutines if they exist
func (a *AdminServiceImpl) Cleanup() error {
	if a.pubBuffer != nil {
		a.pubBuffer.Stop()
	}
	return nil
}
