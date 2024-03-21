package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
)

type Heartbeater interface {
	Start(ctx context.Context) error
}

type HeartbeaterImpl struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.EngineRepository
	s    gocron.Scheduler
}

type HeartbeaterOpt func(*HeartbeaterOpts)

type HeartbeaterOpts struct {
	mq   msgqueue.MessageQueue
	l    *zerolog.Logger
	repo repository.EngineRepository
}

func defaultHeartbeaterOpts() *HeartbeaterOpts {
	logger := logger.NewDefaultLogger("heartbeater")
	return &HeartbeaterOpts{
		l: &logger,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) HeartbeaterOpt {
	return func(opts *HeartbeaterOpts) {
		opts.mq = mq
	}
}

func WithRepository(r repository.EngineRepository) HeartbeaterOpt {
	return func(opts *HeartbeaterOpts) {
		opts.repo = r
	}
}

func WithLogger(l *zerolog.Logger) HeartbeaterOpt {
	return func(opts *HeartbeaterOpts) {
		opts.l = l
	}
}

func New(fs ...HeartbeaterOpt) (*HeartbeaterImpl, error) {
	opts := defaultHeartbeaterOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	newLogger := opts.l.With().Str("service", "heartbeater").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &HeartbeaterImpl{
		mq:   opts.mq,
		l:    opts.l,
		repo: opts.repo,
		s:    s,
	}, nil
}

func (t *HeartbeaterImpl) Start() (func() error, error) {
	t.l.Debug().Msg("starting heartbeater")

	t.s.Start()

	cleanup := func() error {
		t.l.Debug().Msg("stopping heartbeater")
		if err := t.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}
		t.l.Debug().Msg("heartbeater has shutdown")
		return nil
	}

	return cleanup, nil
}
