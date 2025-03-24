package v1

import (
	"fmt"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type AdminService interface {
	contracts.AdminServiceServer
}

type AdminServiceImpl struct {
	contracts.UnimplementedAdminServiceServer

	repo v1.Repository
	mq   msgqueue.MessageQueue
	v    validator.Validator
}

type AdminServiceOpt func(*AdminServiceOpts)

type AdminServiceOpts struct {
	repo v1.Repository
	mq   msgqueue.MessageQueue
	v    validator.Validator
}

func defaultAdminServiceOpts() *AdminServiceOpts {
	v := validator.NewDefaultValidator()

	return &AdminServiceOpts{
		v: v,
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
		v:    opts.v,
	}, nil
}
