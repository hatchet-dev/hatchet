package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/partition"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type RetentionController interface {
	Start(ctx context.Context) error
}

type RetentionControllerImpl struct {
	l             *zerolog.Logger
	repo          repository.EngineRepository
	dv            datautils.DataDecoderValidator
	s             gocron.Scheduler
	tenantAlerter *alerting.TenantAlertManager
	a             *hatcheterrors.Wrapped
	p             *partition.Partition
}

type RetentionControllerOpt func(*RetentionControllerOpts)

type RetentionControllerOpts struct {
	l       *zerolog.Logger
	repo    repository.EngineRepository
	dv      datautils.DataDecoderValidator
	ta      *alerting.TenantAlertManager
	alerter hatcheterrors.Alerter
	p       *partition.Partition
}

func defaultRetentionControllerOpts() *RetentionControllerOpts {
	logger := logger.NewDefaultLogger("retention-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &RetentionControllerOpts{
		l:       &logger,
		dv:      datautils.NewDataDecoderValidator(),
		alerter: alerter,
	}
}

func WithLogger(l *zerolog.Logger) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.l = l
	}
}

func WithRepository(r repository.EngineRepository) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.repo = r
	}
}

func WithAlerter(a hatcheterrors.Alerter) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.alerter = a
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.dv = dv
	}
}

func WithTenantAlerter(ta *alerting.TenantAlertManager) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.ta = ta
	}
}

func WithPartition(p *partition.Partition) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.p = p
	}
}

func New(fs ...RetentionControllerOpt) (*RetentionControllerImpl, error) {
	opts := defaultRetentionControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.ta == nil {
		return nil, fmt.Errorf("tenant alerter is required. use WithTenantAlerter")
	}

	if opts.p == nil {
		return nil, fmt.Errorf("partition is required. use WithPartition")
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	newLogger := opts.l.With().Str("service", "retention-controller").Logger()
	opts.l = &newLogger

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "retention-controller"})

	return &RetentionControllerImpl{
		l:             opts.l,
		repo:          opts.repo,
		dv:            opts.dv,
		s:             s,
		tenantAlerter: opts.ta,
		a:             a,
		p:             opts.p,
	}, nil
}

func (rc *RetentionControllerImpl) Start() (func() error, error) {
	rc.l.Debug().Msg("starting retention controller")

	ctx, cancel := context.WithCancel(context.Background())

	interval := time.Second * 60 // run every 60 seconds

	_, err := rc.s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(
			rc.runDeleteExpiredWorkflowRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up runDeleteExpiredWorkflowRuns: %w", err)
	}

	_, err = rc.s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(
			rc.runDeleteExpiredEvents(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up runDeleteExpiredEvents: %w", err)
	}

	_, err = rc.s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(
			rc.runDeleteExpiredStepRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up runDeleteExpiredStepRuns: %w", err)
	}

	_, err = rc.s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(
			rc.runDeleteQueueItems(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up runDeleteQueueItems: %w", err)
	}

	_, err = rc.s.NewJob(
		gocron.DurationJob(interval),
		gocron.NewTask(
			rc.runDeleteExpiredJobRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not set up runDeleteExpiredJobRuns: %w", err)
	}
	rc.s.Start()

	cleanup := func() error {
		cancel()

		if err := rc.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		return nil
	}

	return cleanup, nil
}
