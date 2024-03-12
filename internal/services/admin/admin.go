package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
)

type AdminService interface {
	contracts.WorkflowServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	repo repository.Repository
	mq   msgqueue.MessageQueue
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repo repository.Repository
	mq   msgqueue.MessageQueue
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	return &AdminServiceOpts{}
}

func WithRepository(r repository.Repository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repo = r
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.mq = mq
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

	return &AdminServiceImpl{
		repo: opts.repo,
		mq:   opts.mq,
	}, nil
}
