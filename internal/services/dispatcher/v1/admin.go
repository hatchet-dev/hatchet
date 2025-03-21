package v1

import (
	"fmt"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DispatcherService interface {
	contracts.V1DispatcherServer
}

type DispatcherServiceImpl struct {
	contracts.UnimplementedV1DispatcherServer

	repo v1.Repository
	mq   msgqueue.MessageQueue
	v    validator.Validator
}

type DispatcherServiceOpt func(*DispatcherServiceOpts)

type DispatcherServiceOpts struct {
	repo v1.Repository
	mq   msgqueue.MessageQueue
	v    validator.Validator
}

func defaultDispatcherServiceOpts() *DispatcherServiceOpts {
	v := validator.NewDefaultValidator()

	return &DispatcherServiceOpts{
		v: v,
	}
}

func WithRepository(r v1.Repository) DispatcherServiceOpt {
	return func(opts *DispatcherServiceOpts) {
		opts.repo = r
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) DispatcherServiceOpt {
	return func(opts *DispatcherServiceOpts) {
		opts.mq = mq
	}
}

func WithValidator(v validator.Validator) DispatcherServiceOpt {
	return func(opts *DispatcherServiceOpts) {
		opts.v = v
	}
}

func NewDispatcherService(fs ...DispatcherServiceOpt) (DispatcherService, error) {
	opts := defaultDispatcherServiceOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	return &DispatcherServiceImpl{
		repo: opts.repo,
		mq:   opts.mq,
		v:    opts.v,
	}, nil
}
