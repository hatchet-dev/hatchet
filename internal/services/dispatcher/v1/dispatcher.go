package v1

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/task/trigger"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DispatcherService interface {
	contracts.V1DispatcherServer
}

type DispatcherServiceImpl struct {
	contracts.UnimplementedV1DispatcherServer

	repo          v1.Repository
	mq            msgqueue.MessageQueue
	v             validator.Validator
	l             *zerolog.Logger
	triggerWriter *trigger.TriggerWriter
}

type DispatcherServiceOpt func(*DispatcherServiceOpts)

type DispatcherServiceOpts struct {
	repo v1.Repository
	mq   msgqueue.MessageQueue
	v    validator.Validator
	l    *zerolog.Logger
}

func defaultDispatcherServiceOpts() *DispatcherServiceOpts {
	v := validator.NewDefaultValidator()
	logger := logger.NewDefaultLogger("dispatcher")

	return &DispatcherServiceOpts{
		v: v,
		l: &logger,
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

func WithLogger(l *zerolog.Logger) DispatcherServiceOpt {
	return func(opts *DispatcherServiceOpts) {
		opts.l = l
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

	pubBuffer := msgqueue.NewMQPubBuffer(opts.mq)
	tw := trigger.NewTriggerWriter(opts.mq, opts.repo, opts.l, pubBuffer, 0)

	return &DispatcherServiceImpl{
		repo:          opts.repo,
		mq:            opts.mq,
		v:             opts.v,
		l:             opts.l,
		triggerWriter: tw,
	}, nil
}
