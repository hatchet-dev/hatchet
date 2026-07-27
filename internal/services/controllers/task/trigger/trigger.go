package trigger

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"

	"github.com/rs/zerolog"
)

// TriggerWriter writes workflow triggers through the trigger repository, which stages
// all resulting OLAP messages on the trigger transaction and runs the non-transactional
// signaling side effects post-commit.
type TriggerWriter struct {
	repo v1.Repository
	l    *zerolog.Logger

	semaphore chan struct{}
}

var ErrNoTriggerSlots = errors.New("no trigger slots available")

// NewTriggerWriter creates a new TriggerWriter with the given number of slots for concurrency control.
// If the number of slots is 0, there is no limit to concurrency.
func NewTriggerWriter(repo v1.Repository, l *zerolog.Logger, slots int) *TriggerWriter {
	var sem chan struct{}

	if slots > 0 {
		sem = make(chan struct{}, slots)
	}

	return &TriggerWriter{
		l:         l,
		repo:      repo,
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

	_, err := tw.repo.Triggers().TriggerFromEvents(ctx, tenantId, opts)

	if err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			tw.l.Warn().Ctx(ctx).Str("tenantId", tenantId.String()).Msg("resource exhausted while calling TriggerFromEvents. Not retrying")

			return nil
		}

		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	return nil
}

func (tw *TriggerWriter) TriggerFromWorkflowNames(ctx context.Context, tenantId uuid.UUID, opts []*v1.WorkflowNameTriggerOpts) ([]v1.IdempotencyCollision, error) {
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
			return nil, ErrNoTriggerSlots
		}
	}

	_, _, idempotencyKeyCollisions, _, err := tw.repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)

	if err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			tw.l.Warn().Ctx(ctx).Str("tenantId", tenantId.String()).Msg("resource exhausted while calling TriggerFromWorkflowNames. Not retrying")

			return nil, nil
		}

		return nil, fmt.Errorf("could not trigger workflows from names: %w", err)
	}

	return idempotencyKeyCollisions, nil
}
