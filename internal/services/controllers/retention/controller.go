package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
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
	partitionId   string
}

type RetentionControllerOpt func(*RetentionControllerOpts)

type RetentionControllerOpts struct {
	l           *zerolog.Logger
	repo        repository.EngineRepository
	dv          datautils.DataDecoderValidator
	ta          *alerting.TenantAlertManager
	alerter     hatcheterrors.Alerter
	partitionId string
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

func WithPartitionId(partitionId string) RetentionControllerOpt {
	return func(opts *RetentionControllerOpts) {
		opts.partitionId = partitionId
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

	if opts.partitionId == "" {
		return nil, fmt.Errorf("partition ID is required. use WithPartitionId")
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	newLogger := opts.l.With().Str("service", "workflows-controller").Logger()
	opts.l = &newLogger

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "workflows-controller"})

	return &RetentionControllerImpl{
		l:             opts.l,
		repo:          opts.repo,
		dv:            opts.dv,
		s:             s,
		tenantAlerter: opts.ta,
		a:             a,
		partitionId:   opts.partitionId,
	}, nil
}

func (rc *RetentionControllerImpl) Start() (func() error, error) {
	rc.l.Debug().Msg("starting workflows controller")

	ctx, cancel := context.WithCancel(context.Background())

	// TODO clean up soft delete cron
	// _, err = wc.s.NewJob(
	// 	gocron.DurationJob(time.Second*60),
	// 	gocron.NewTask(
	// 		wc.runSoftDeleteCleanup(ctx),
	// 	),
	// )

	// if err != nil {
	// 	cancel()
	// 	return nil, fmt.Errorf("could not delete expired workflow runs: %w", err)
	// }

	_, err := rc.s.NewJob(
		gocron.DurationJob(time.Second*60),
		gocron.NewTask(
			rc.runDeleteExpiredWorkflowRuns(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not delete expired workflow runs: %w", err)
	}

	_, err = rc.s.NewJob(
		gocron.DurationJob(time.Second*60),
		gocron.NewTask(
			rc.runDeleteExpiredEvents(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not delete expired events: %w", err)
	}

	rc.s.Start()

	cleanup := func() error {
		cancel()
		return nil
	}

	return cleanup, nil
}
