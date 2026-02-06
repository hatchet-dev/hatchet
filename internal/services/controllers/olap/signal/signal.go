package signal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type OLAPSignaler struct {
	mq        msgqueue.MessageQueue
	repo      v1.Repository
	pubBuffer *msgqueue.MQPubBuffer
	l         *zerolog.Logger
}

func NewOLAPSignaler(mq msgqueue.MessageQueue, repo v1.Repository, l *zerolog.Logger, pubBuffer *msgqueue.MQPubBuffer) *OLAPSignaler {
	return &OLAPSignaler{
		mq:        mq,
		l:         l,
		repo:      repo,
		pubBuffer: pubBuffer,
	}
}

func (s *OLAPSignaler) SignalDAGsCreated(ctx context.Context, tenantId uuid.UUID, dags []*v1.DAGWithData) error {
	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, dag := range dags {
		dagCp := dag
		msg, err := tasktypes.CreatedDAGMessage(tenantId, dagCp)

		if err != nil {
			s.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add message to olap queue")
			continue
		}
	}

	return nil
}

func (s *OLAPSignaler) SignalTasksCreated(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	// group tasks by initial states
	queuedTasks := make([]*v1.V1TaskWithPayload, 0)
	failedTasks := make([]*v1.V1TaskWithPayload, 0)
	cancelledTasks := make([]*v1.V1TaskWithPayload, 0)
	skippedTasks := make([]*v1.V1TaskWithPayload, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}

		msg, err := tasktypes.CreatedTaskMessage(tenantId, task)

		if err != nil {
			s.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add message to olap queue")
			continue
		}
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (s *OLAPSignaler) SignalTasksUpdated(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	// group tasks by initial states
	queuedTasks := make([]*v1.V1TaskWithPayload, 0)
	failedTasks := make([]*v1.V1TaskWithPayload, 0)
	cancelledTasks := make([]*v1.V1TaskWithPayload, 0)
	skippedTasks := make([]*v1.V1TaskWithPayload, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (s *OLAPSignaler) signalTasksCreatedAndQueued(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	// get all unique queues and notify them
	queues := make(map[string]struct{})

	for _, task := range tasks {
		queues[task.Queue] = struct{}{}
	}

	tenant, err := s.repo.Tenant().GetTenantByID(ctx, tenantId)

	if err != nil {
		return err
	}

	if tenant.SchedulerPartitionId.Valid {
		msg, err := tasktypes.NotifyTaskCreated(tenantId, tasks)

		if err != nil {
			s.l.Err(err).Msg("could not create message for scheduler partition queue")
		} else {
			err = s.mq.SendMessage(
				ctx,
				msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler),
				msg,
			)

			if err != nil {
				s.l.Err(err).Msg("could not add message to scheduler partition queue")
			}
		}
	}

	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg := ""

		if len(task.ConcurrencyKeys) > 0 {
			msg = "concurrency keys evaluated as:"

			for _, key := range task.ConcurrencyKeys {
				msg += fmt.Sprintf(" %s", key)
			}
		}

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.ID,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapQUEUED,
				EventTimestamp: time.Now(),
				EventMessage:   msg,
			},
		)

		if err != nil {
			s.l.Err(err).Msg("could not create monitoring event message")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add monitoring event message to olap queue")
			continue
		}
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.TenantCreatedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	}()

	return nil
}

func (s *OLAPSignaler) signalTasksCreatedAndCancelled(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := task.ExternalID

		dataBytes := v1.NewCancelledTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCANCELLED,
			Data:           dataBytes,
		})
	}

	err := s.SendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapCANCELLING,
			EventTimestamp: time.Now(),
		})

		if err != nil {
			s.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add message to olap queue")
			continue
		}
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.TenantCreatedTasks.WithLabelValues(tenantId.String()).Inc()
			prometheus.CancelledTasks.Inc()
			prometheus.TenantCancelledTasks.WithLabelValues(tenantId.String()).Inc()
		}
	}()

	return nil
}

func (s *OLAPSignaler) signalTasksCreatedAndFailed(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := task.ExternalID

		dataBytes := v1.NewFailedTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeFAILED,
			Data:           dataBytes,
		})
	}

	err := s.SendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapFAILED,
			EventPayload:   task.InitialStateReason.String,
			EventTimestamp: time.Now(),
		})

		if err != nil {
			s.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add message to olap queue")
			continue
		}
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.TenantCreatedTasks.WithLabelValues(tenantId.String()).Inc()
			prometheus.FailedTasks.Inc()
			prometheus.TenantFailedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	}()

	return nil
}

func (s *OLAPSignaler) signalTasksCreatedAndSkipped(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	internalEvents := make([]v1.InternalTaskEvent, 0)

	for _, task := range tasks {
		taskExternalId := task.ExternalID

		dataBytes := v1.NewSkippedTaskOutputEventFromTask(task).Bytes()

		internalEvents = append(internalEvents, v1.InternalTaskEvent{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskExternalID: taskExternalId,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
			Data:           dataBytes,
		})
	}

	err := s.SendInternalEvents(ctx, tenantId, internalEvents)

	if err != nil {
		return err
	}

	// notify that tasks have been cancelled
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(tenantId, tasktypes.CreateMonitoringEventPayload{
			TaskId:         task.ID,
			RetryCount:     task.RetryCount,
			EventType:      sqlcv1.V1EventTypeOlapSKIPPED,
			EventTimestamp: time.Now(),
		})

		if err != nil {
			s.l.Err(err).Msg("could not create message for olap queue")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			msg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add message to olap queue")
			continue
		}
	}

	// instrumentation
	go func() {
		for range tasks {
			prometheus.CreatedTasks.Inc()
			prometheus.TenantCreatedTasks.WithLabelValues(tenantId.String()).Inc()
			prometheus.SkippedTasks.Inc()
			prometheus.TenantSkippedTasks.WithLabelValues(tenantId.String()).Inc()
		}
	}()

	return nil
}

func (s *OLAPSignaler) SignalTasksReplayed(ctx context.Context, tenantId uuid.UUID, tasks []v1.TaskIdInsertedAtRetryCount) error {
	// notify that tasks have been created
	// TODO: make this transactionally safe?
	for _, task := range tasks {
		msg := "Task was replayed, resetting task result."

		olapMsg, err := tasktypes.MonitoringEventMessageFromInternal(
			tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:         task.Id,
				RetryCount:     task.RetryCount,
				EventType:      sqlcv1.V1EventTypeOlapRETRIEDBYUSER,
				EventTimestamp: time.Now(),
				EventMessage:   msg,
			},
		)

		if err != nil {
			s.l.Err(err).Msg("could not create monitoring event message")
			continue
		}

		err = s.pubBuffer.Pub(
			ctx,
			msgqueue.OLAP_QUEUE,
			olapMsg,
			false,
		)

		if err != nil {
			s.l.Err(err).Msg("could not add monitoring event message to olap queue")
			continue
		}
	}

	return nil
}

func (s *OLAPSignaler) SignalTasksReplayedFromMatch(ctx context.Context, tenantId uuid.UUID, tasks []*v1.V1TaskWithPayload) error {
	// group tasks by initial states
	queuedTasks := make([]*v1.V1TaskWithPayload, 0)
	failedTasks := make([]*v1.V1TaskWithPayload, 0)
	cancelledTasks := make([]*v1.V1TaskWithPayload, 0)
	skippedTasks := make([]*v1.V1TaskWithPayload, 0)

	for _, task := range tasks {
		switch task.InitialState {
		case sqlcv1.V1TaskInitialStateQUEUED:
			queuedTasks = append(queuedTasks, task)
		case sqlcv1.V1TaskInitialStateFAILED:
			failedTasks = append(failedTasks, task)
		case sqlcv1.V1TaskInitialStateCANCELLED:
			cancelledTasks = append(cancelledTasks, task)
		case sqlcv1.V1TaskInitialStateSKIPPED:
			skippedTasks = append(skippedTasks, task)
		}
	}

	eg := &errgroup.Group{}

	if len(queuedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndQueued(ctx, tenantId, queuedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(failedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndFailed(ctx, tenantId, failedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(cancelledTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndCancelled(ctx, tenantId, cancelledTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	if len(skippedTasks) > 0 {
		eg.Go(func() error {
			err := s.signalTasksCreatedAndSkipped(ctx, tenantId, skippedTasks)

			if err != nil {
				return fmt.Errorf("could not signal created tasks: %w", err)
			}

			return nil
		})
	}

	return eg.Wait()
}

func (s *OLAPSignaler) SendInternalEvents(ctx context.Context, tenantId uuid.UUID, events []v1.InternalTaskEvent) error {
	ctx, span := telemetry.NewSpan(ctx, "OLAPSignaler.SendInternalEvents")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant.id", Value: tenantId})

	if len(events) == 0 {
		return nil
	}

	msg, err := tasktypes.NewInternalEventMessage(tenantId, time.Now(), events...)

	if err != nil {
		err = fmt.Errorf("could not create internal event message: %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "could not create internal event message")
		return err
	}

	return s.mq.SendMessage(
		ctx,
		msgqueue.TASK_PROCESSING_QUEUE,
		msg,
	)
}

func (s *OLAPSignaler) SignalEventsCreated(ctx context.Context, tenantId uuid.UUID, eventIdToOpts map[uuid.UUID]v1.EventTriggerOpts, eventIdsToRuns map[uuid.UUID][]*v1.Run) error {
	eventTriggerOpts := make([]tasktypes.CreatedEventTriggerPayloadSingleton, 0)

	// FIXME: Should `SeenAt` be set on the SDK when the event is created?
	eventSeenAt := time.Now()

	for eventExternalId, runs := range eventIdsToRuns {
		opts := eventIdToOpts[eventExternalId]

		if len(runs) == 0 {
			eventTriggerOpts = append(eventTriggerOpts, tasktypes.CreatedEventTriggerPayloadSingleton{
				EventSeenAt:             eventSeenAt,
				EventKey:                opts.Key,
				EventExternalId:         opts.ExternalId,
				EventPayload:            opts.Data,
				EventAdditionalMetadata: opts.AdditionalMetadata,
				TriggeringWebhookName:   opts.TriggeringWebhookName,
				EventScope:              opts.Scope,
			})
		} else {
			for _, run := range runs {
				eventTriggerOpts = append(eventTriggerOpts, tasktypes.CreatedEventTriggerPayloadSingleton{
					MaybeRunId:              &run.Id,
					MaybeRunInsertedAt:      &run.InsertedAt,
					EventSeenAt:             eventSeenAt,
					EventKey:                opts.Key,
					EventExternalId:         opts.ExternalId,
					EventPayload:            opts.Data,
					EventAdditionalMetadata: opts.AdditionalMetadata,
					EventScope:              opts.Scope,
					FilterId:                run.FilterId,
					TriggeringWebhookName:   opts.TriggeringWebhookName,
				})
			}
		}
	}

	msg, err := tasktypes.CreatedEventTriggerMessage(
		tenantId,
		tasktypes.CreatedEventTriggerPayload{
			Payloads: eventTriggerOpts,
		},
	)

	if err != nil {
		return fmt.Errorf("could not create event trigger message: %w", err)
	}

	err = s.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)

	if err != nil {
		return fmt.Errorf("could not trigger tasks from events: %w", err)
	}

	return nil
}

func (s *OLAPSignaler) SignalCELEvaluationFailures(ctx context.Context, tenantId uuid.UUID, failures []v1.CELEvaluationFailure) error {
	evalFailuresMsg, err := tasktypes.CELEvaluationFailureMessage(
		tenantId,
		failures,
	)

	if err != nil {
		return fmt.Errorf("could not create CEL evaluation failure message: %w", err)
	}

	err = s.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, evalFailuresMsg, false)

	if err != nil {
		return fmt.Errorf("could not deliver CEL evaluation failure message: %w", err)
	}

	return nil
}
