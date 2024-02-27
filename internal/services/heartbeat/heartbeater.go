package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type Heartbeater interface {
	Start(ctx context.Context) error
}

type HeartbeaterImpl struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
	s    gocron.Scheduler
}

type HeartbeaterOpt func(*HeartbeaterOpts)

type HeartbeaterOpts struct {
	tq   taskqueue.TaskQueue
	l    *zerolog.Logger
	repo repository.Repository
}

func defaultHeartbeaterOpts() *HeartbeaterOpts {
	logger := logger.NewDefaultLogger("heartbeater")
	return &HeartbeaterOpts{
		l: &logger,
	}
}

func WithTaskQueue(tq taskqueue.TaskQueue) HeartbeaterOpt {
	return func(opts *HeartbeaterOpts) {
		opts.tq = tq
	}
}

func WithRepository(r repository.Repository) HeartbeaterOpt {
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

	if opts.tq == nil {
		return nil, fmt.Errorf("task queue is required. use WithTaskQueue")
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
		tq:   opts.tq,
		l:    opts.l,
		repo: opts.repo,
		s:    s,
	}, nil
}

func (t *HeartbeaterImpl) Start() (func() error, error) {
	t.l.Debug().Msg("starting heartbeater")

	_, err := t.s.NewJob(
		gocron.DurationJob(time.Second*5),
		gocron.NewTask(
			t.removeStaleTickers(),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("could not schedule ticker removal: %w", err)
	}

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
