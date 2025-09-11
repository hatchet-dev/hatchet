package v1

import (
	"fmt"

	"github.com/rs/zerolog"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	scheduler "github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.AdminServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedAdminServiceServer

	entitlements    repository.EntitlementsRepository
	repo            v1.Repository
	mq              msgqueue.MessageQueue
	v               validator.Validator
	analytics       analytics.Analytics
	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	entitlements    repository.EntitlementsRepository
	repo            v1.Repository
	mq              msgqueue.MessageQueue
	v               validator.Validator
	analytics       analytics.Analytics
	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()
	logger := logger.NewDefaultLogger("v1_admin_service")

	return &AdminServiceOpts{
		v: v,
		l: &logger,
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.entitlements = r
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

func WithLogger(l *zerolog.Logger) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.l = l
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

	if opts.entitlements == nil {
		return nil, fmt.Errorf("entitlements repository is required. use WithEntitlementsRepository")
	}

	return &AdminServiceImpl{
		entitlements:    opts.entitlements,
		repo:            opts.repo,
		mq:              opts.mq,
		v:               opts.v,
		analytics:       opts.analytics,
		localScheduler:  opts.localScheduler,
		localDispatcher: opts.localDispatcher,
		l:               opts.l,
	}, nil
}
