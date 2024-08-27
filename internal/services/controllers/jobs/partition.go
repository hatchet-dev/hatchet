package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type Partition struct {
	mq          msgqueue.MessageQueue
	l           *zerolog.Logger
	repo        repository.EngineRepository
	dv          datautils.DataDecoderValidator
	s           gocron.Scheduler
	a           *hatcheterrors.Wrapped
	partitionId string

	tenantOperations sync.Map
}

func NewPartition(
	mq msgqueue.MessageQueue,
	l *zerolog.Logger,
	repo repository.EngineRepository,
	dv datautils.DataDecoderValidator,
	a *hatcheterrors.Wrapped,
	partitionId string,
) (*Partition, error) {
	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	return &Partition{
		mq:          mq,
		l:           l,
		repo:        repo,
		dv:          dv,
		s:           s,
		a:           a,
		partitionId: partitionId,
	}, nil
}

type operation struct {
	mu             sync.RWMutex
	shouldContinue bool
	isRunning      bool
	tenantId       string
}

func (o *operation) run(l *zerolog.Logger, scheduler func(context.Context, string) (bool, error)) {
	if o.getRunning() {
		return
	}

	o.setRunning(true)

	go func() {
		defer func() {
			o.setRunning(false)
		}()

		f := func() {
			o.setContinue(false)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			shouldContinue, err := scheduler(ctx, o.tenantId)

			if err != nil {
				l.Err(err).Msgf("could not schedule step runs for tenant")
				return
			}

			// if a continue was set during execution of the scheduler, we'd like to continue no matter what.
			// if a continue was not set, we'd like to set it to the value returned by the scheduler.
			if !o.getContinue() {
				o.setContinue(shouldContinue)
			}
		}

		f()

		for o.getContinue() {
			f()
		}
	}()
}

func (o *operation) setRunning(isRunning bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.isRunning = isRunning
}

func (o *operation) getRunning() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.isRunning
}

func (o *operation) setContinue(shouldContinue bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.shouldContinue = shouldContinue
}

func (o *operation) getContinue() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.shouldContinue
}

func (p *Partition) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	_, err := p.s.NewJob(
		gocron.DurationJob(time.Second*1),
		gocron.NewTask(
			p.runTenantQueues(ctx),
		),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule step run reassign: %w", err)
	}

	p.s.Start()

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		err := p.handleTask(context.Background(), task)
		if err != nil {
			p.l.Error().Err(err).Msg("could not handle job task")
			return p.a.WrapErr(fmt.Errorf("could not handle job task: %w", err), map[string]interface{}{"task_id": task.ID}) // nolint: errcheck
		}

		return nil
	}

	cleanupQueue, err := p.mq.Subscribe(msgqueue.QueueTypeFromPartitionID(p.partitionId), f, msgqueue.NoOpHook)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not subscribe to job processing queue: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupQueue(); err != nil {
			return fmt.Errorf("could not cleanup job processing queue: %w", err)
		}

		if err := p.s.Shutdown(); err != nil {
			return fmt.Errorf("could not shutdown scheduler: %w", err)
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (p *Partition) handleTask(ctx context.Context, task *msgqueue.Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(p.l, p.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	if task.ID == "check-tenant-queue" {
		return p.handleCheckQueue(ctx, task)
	}

	return fmt.Errorf("unknown task: %s", task.ID)
}

func (p *Partition) handleCheckQueue(ctx context.Context, task *msgqueue.Message) error {
	_, span := telemetry.NewSpan(ctx, "handle-check-queue")
	defer span.End()

	metadata := tasktypes.CheckTenantQueueMetadata{}

	err := p.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode check queue metadata: %w", err)
	}

	// if this tenant is registered, then we should check the queue
	if opInt, ok := p.tenantOperations.Load(metadata.TenantId); ok {
		op := opInt.(*operation)

		op.setContinue(true)
		op.run(p.l, p.scheduleStepRuns)
	}

	return nil
}

func (p *Partition) runTenantQueues(ctx context.Context) func() {
	return func() {
		p.l.Debug().Msgf("partition: checking step run requeue")

		// list all tenants
		tenants, err := p.repo.Tenant().ListTenantsByControllerPartition(ctx, p.partitionId)

		if err != nil {
			p.l.Err(err).Msg("could not list tenants")
			return
		}

		for i := range tenants {
			tenantId := sqlchelpers.UUIDToStr(tenants[i].ID)

			var op *operation

			opInt, ok := p.tenantOperations.Load(tenantId)

			if !ok {
				op = &operation{
					tenantId: tenantId,
				}

				p.tenantOperations.Store(tenantId, op)
			} else {
				op = opInt.(*operation)
			}

			op.run(p.l, p.scheduleStepRuns)
		}
	}
}

func (p *Partition) scheduleStepRuns(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "schedule-step-runs")
	defer span.End()

	dbCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	queueResults, err := p.repo.StepRun().QueueStepRuns(dbCtx, tenantId)

	if err != nil {
		return false, fmt.Errorf("could not queue step runs: %w", err)
	}

	for _, queueResult := range queueResults.Queued {
		// send a task to the dispatcher
		innerErr := p.mq.AddMessage(
			ctx,
			msgqueue.QueueTypeFromDispatcherID(queueResult.DispatcherId),
			stepRunAssignedTask(tenantId, queueResult.StepRunId, queueResult.WorkerId, queueResult.DispatcherId),
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not send queued step run: %w", innerErr))
		}
	}

	for _, toCancel := range queueResults.SchedulingTimedOut {
		innerErr := p.mq.AddMessage(
			ctx,
			msgqueue.JOB_PROCESSING_QUEUE,
			getStepRunCancelTask(
				tenantId,
				toCancel,
				"SCHEDULING_TIMED_OUT",
			),
		)

		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not send cancel step run event: %w", innerErr))
		}
	}

	return queueResults.Continue, err
}

func getStepRunCancelTask(tenantId, stepRunId, reason string) *msgqueue.Message {
	payload, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskPayload{
		StepRunId:       stepRunId,
		CancelledReason: reason,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunCancelTaskMetadata{
		TenantId: tenantId,
	})

	return &msgqueue.Message{
		ID:       "step-run-cancel",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	}
}
