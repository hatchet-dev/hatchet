package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	scheduler "github.com/hatchet-dev/hatchet/internal/services/scheduler/v1"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.WorkflowServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	repov1 v1.Repository
	mqv1   msgqueue.MessageQueue
	v      validator.Validator

	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repov1          v1.Repository
	mqv1            msgqueue.MessageQueue
	v               validator.Validator
	localScheduler  *scheduler.Scheduler
	localDispatcher *dispatcher.DispatcherImpl
	l               *zerolog.Logger
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()
	logger := logger.NewDefaultLogger("admin_service")

	return &AdminServiceOpts{
		v: v,
		l: &logger,
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

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	return &AdminServiceImpl{
		repov1:          opts.repov1,
		mqv1:            opts.mqv1,
		v:               opts.v,
		localScheduler:  opts.localScheduler,
		localDispatcher: opts.localDispatcher,
		l:               opts.l,
	}, nil
}
