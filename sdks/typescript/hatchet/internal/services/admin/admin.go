package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.WorkflowServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	mq           msgqueue.MessageQueue
	v            validator.Validator
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	mq           msgqueue.MessageQueue
	v            validator.Validator
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()

	return &AdminServiceOpts{
		v: v,
	}
}

func WithRepository(r repository.EngineRepository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repo = r
	}
}

func WithEntitlementsRepository(r repository.EntitlementsRepository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.entitlements = r
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
		repo:         opts.repo,
		entitlements: opts.entitlements,
		mq:           opts.mq,
		v:            opts.v,
	}, nil
}
