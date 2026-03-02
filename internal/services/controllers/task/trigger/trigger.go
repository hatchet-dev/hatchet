package trigger

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/controllers/olap/signal"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"

	"github.com/rs/zerolog"
)

type TriggerWriter struct {
	mq        msgqueue.MessageQueue
	repo      v1.Repository
	pubBuffer *msgqueue.MQPubBuffer
	l         *zerolog.Logger
	signaler  *signal.OLAPSignaler

	semaphore chan struct{}
}

var ErrNoTriggerSlots = errors.New("no trigger slots available")

// NewTriggerWriter creates a new TriggerWriter with the given number of slots for concurrency control.
// If the number of slots is 0, there is no limit to concurrency.
func NewTriggerWriter(mq msgqueue.MessageQueue, repo v1.Repository, l *zerolog.Logger, pubBuffer *msgqueue.MQPubBuffer, slots int) *TriggerWriter {
	s := signal.NewOLAPSignaler(mq, repo, l, pubBuffer)

	var sem chan struct{}

	if slots > 0 {
		sem = make(chan struct{}, slots)
	}

	return &TriggerWriter{
		mq:        mq,
		l:         l,
		repo:      repo,
		pubBuffer: pubBuffer,
		signaler:  s,
		semaphore: sem,
	}
}

func (tw *TriggerWriter) TriggerFromEvents(ctx context.Context, tenantId uuid.UUID, eventIdToOpts map[uuid.UUID]v1.EventTriggerOpts) error {
	// attempt to acquire a slot in the semaphore
	if tw.semaphore != nil {
		select {
		case tw.semaphore <- struct{}{}:
			// acquired a slot
			defer func() {
				<-tw.semaphore
			}()
		default:
			// no slots available
			return ErrNoTriggerSlots
		}
	}

	opts := make([]v1.EventTriggerOpts, 0, len(eventIdToOpts))

	for _, opt := range eventIdToOpts {
		opts = append(opts, opt)
	}

	result, err := tw.repo.Triggers().TriggerFromEvents(ctx, tenantId, opts)

	if err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			tw.l.Warn().Str("tenantId", tenantId.String()).Msg("resource exhausted while calling TriggerFromEvents. Not retrying")

			return nil
		}

		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		return tw.signaler.SignalEventsCreated(ctx, tenantId, eventIdToOpts, result.EventExternalIdToRuns)
	})

	eg.Go(func() error {
		return tw.signaler.SignalCELEvaluationFailures(ctx, tenantId, result.CELEvaluationFailures)
	})

	eg.Go(func() error {
		return tw.signaler.SignalTasksCreated(ctx, tenantId, result.Tasks)
	})

	eg.Go(func() error {
		return tw.signaler.SignalDAGsCreated(ctx, tenantId, result.Dags)
	})

	// signaling errors do not result in a failure, since we have already written the tasks to the database, but
	// we log the error
	// FIXME: we need a mechanism to DLQ these failed signals
	if err := eg.Wait(); err != nil {
		tw.l.Error().Err(err).Msg("failed to signal created tasks and DAGs in TriggerFromEvents")
	}

	return nil
}

func (tw *TriggerWriter) TriggerFromWorkflowNames(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts) error {
	// attempt to acquire a slot in the semaphore
	if tw.semaphore != nil {
		select {
		case tw.semaphore <- struct{}{}:
			// acquired a slot
			defer func() {
				<-tw.semaphore
			}()
		default:
			// no slots available
			return ErrNoTriggerSlots
		}
	}

	tasks, dags, err := tw.repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)

	if err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			tw.l.Warn().Str("tenantId", tenantId.String()).Msg("resource exhausted while calling TriggerFromWorkflowNames. Not retrying")

			return nil
		}

		return fmt.Errorf("could not trigger workflows from names: %w", err)
	}

	eg := &errgroup.Group{}

	eg.Go(func() error {
		return tw.signaler.SignalTasksCreated(ctx, tenantId, tasks)
	})

	eg.Go(func() error {
		return tw.signaler.SignalDAGsCreated(ctx, tenantId, dags)
	})

	// signaling errors do not result in a failure, since we have already written the tasks to the database, but
	// we log the error
	// FIXME: we need a mechanism to DLQ these failed signals
	if err := eg.Wait(); err != nil {
		tw.l.Error().Err(err).Msg("failed to signal created tasks and DAGs in TriggerFromWorkflowNames")
	}

	return nil
}
