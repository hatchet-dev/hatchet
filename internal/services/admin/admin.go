package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type AdminService interface {
	contracts.WorkflowServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	repo repository.Repository
	tq   taskqueue.TaskQueue
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repo repository.Repository
	tq   taskqueue.TaskQueue
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	return &AdminServiceOpts{}
}

func WithRepository(r repository.Repository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repo = r
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.tq = tq
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

	if opts.tq == nil {
		return nil, fmt.Errorf("task queue is required. use WithTaskQueue")
	}

	return &AdminServiceImpl{
		repo: opts.repo,
		tq:   opts.tq,
	}, nil
}
