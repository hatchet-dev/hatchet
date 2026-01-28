package scheduler

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	schedulingv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

func (s *Scheduler) RunOptimisticScheduling(ctx context.Context, tenantId string, opts []*v1.WorkflowNameTriggerOpts, localWorkerIds map[string]struct{}) (map[string][]*schedulingv1.AssignedItemWithTask, error) {
	localTasks, tasks, dags, err := s.pool.RunOptimisticScheduling(ctx, tenantId, opts, localWorkerIds)

	if err != nil {
		return nil, err
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		eg := &errgroup.Group{}

		eg.Go(func() error {
			return s.signaler.SignalTasksCreated(ctx, tenantId, tasks)
		})

		eg.Go(func() error {
			return s.signaler.SignalDAGsCreated(ctx, tenantId, dags)
		})

		innerErr := eg.Wait()

		if innerErr != nil {
			s.l.Error().Err(innerErr).Msgf("failed to signal optimistic scheduling results for tenant %s", tenantId)
		}
	}()

	return localTasks, err
}

func (s *Scheduler) RunOptimisticSchedulingFromEvents(ctx context.Context, tenantId string, opts []v1.EventTriggerOpts, localWorkerIds map[string]struct{}) (map[string][]*schedulingv1.AssignedItemWithTask, error) {
	localTasks, eventRes, err := s.pool.RunOptimisticSchedulingFromEvents(ctx, tenantId, opts, localWorkerIds)

	if err != nil {
		return nil, err
	}

	eventIdToOpts := make(map[string]v1.EventTriggerOpts)

	for _, opt := range opts {
		eventIdToOpts[opt.ExternalId] = opt
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		eg := &errgroup.Group{}

		eg.Go(func() error {
			return s.signaler.SignalEventsCreated(ctx, tenantId, eventIdToOpts, eventRes.EventExternalIdToRuns)
		})

		eg.Go(func() error {
			return s.signaler.SignalCELEvaluationFailures(ctx, tenantId, eventRes.CELEvaluationFailures)
		})

		eg.Go(func() error {
			return s.signaler.SignalTasksCreated(ctx, tenantId, eventRes.Tasks)
		})

		eg.Go(func() error {
			return s.signaler.SignalDAGsCreated(ctx, tenantId, eventRes.Dags)
		})

		innerErr := eg.Wait()

		if innerErr != nil {
			s.l.Error().Err(innerErr).Msgf("failed to signal optimistic scheduling results for tenant %s", tenantId)
		}
	}()

	return localTasks, err
}
