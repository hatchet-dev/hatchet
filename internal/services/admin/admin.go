package admin

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	msgqueuev1 "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.WorkflowServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedWorkflowServiceServer

	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	repov1       v1.Repository
	mq           msgqueue.MessageQueue
	mqv1         msgqueuev1.MessageQueue
	v            validator.Validator
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	entitlements repository.EntitlementsRepository
	repo         repository.EngineRepository
	repov1       v1.Repository
	mq           msgqueue.MessageQueue
	mqv1         msgqueuev1.MessageQueue
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

func WithRepositoryV1(r v1.Repository) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.repov1 = r
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

func WithMessageQueueV1(mq msgqueuev1.MessageQueue) AdminServiceOpt {
	return func(opts *AdminServiceOpts) {
		opts.mqv1 = mq
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

	if opts.repov1 == nil {
		return nil, fmt.Errorf("repository v1 is required. use WithRepositoryV1")
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.mqv1 == nil {
		return nil, fmt.Errorf("task queue v1 is required. use WithMessageQueueV1")
	}

	if opts.entitlements == nil {
		return nil, fmt.Errorf("entitlements repository is required. use WithEntitlementsRepository")
	}

	return &AdminServiceImpl{
		repo:         opts.repo,
		repov1:       opts.repov1,
		entitlements: opts.entitlements,
		mq:           opts.mq,
		mqv1:         opts.mqv1,
		v:            opts.v,
	}, nil
}
