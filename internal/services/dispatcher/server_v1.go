package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowRunsV1(server contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	tenant := server.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received subscribe request for tenant: %s", tenantId)

	acks := &workflowRunAcks{
		acks: make(map[string]bool),
	}

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	wg := sync.WaitGroup{}
	sendMu := sync.Mutex{}
	iterMu := sync.Mutex{}

	sendEvent := func(e *contracts.WorkflowRunEvent) error {
		results := cleanResults(e.Results)

		if results == nil {
			s.l.Warn().Msgf("results size for workflow run %s exceeds 3MB and cannot be reduced", e.WorkflowRunId)
			e.Results = nil
		}

		// send the task to the client
		sendMu.Lock()
		err := server.Send(e)
		sendMu.Unlock()

		if err != nil {
			s.l.Error().Err(err).Msgf("could not subscribe to workflow events for run %s", e.WorkflowRunId)
			return err
		}

		acks.ackWorkflowRun(e.WorkflowRunId)

		return nil
	}

	iter := func(workflowRunIds []string) error {
		if len(workflowRunIds) == 0 {
			return nil
		}

		if !iterMu.TryLock() {
			s.l.Warn().Msg("could not acquire lock")
			return nil
		}

		defer iterMu.Unlock()

		workflowRunIds = workflowRunIds[:min(1000, len(workflowRunIds))]
		start := time.Now()

		finalizedWorkflowRuns, err := s.repov1.Tasks().ListFinalizedWorkflowRuns(ctx, tenantId, workflowRunIds)

		if err != nil {
			s.l.Error().Err(err).Msg("could not list finalized workflow runs")
			return err
		}

		events, err := s.taskEventsToWorkflowRunEvent(tenantId, finalizedWorkflowRuns)

		if err != nil {
			s.l.Error().Err(err).Msg("could not convert task events to workflow run events")
			return err
		}

		if time.Since(start) > 100*time.Millisecond {
			s.l.Warn().Msgf("list finalized workflow runs for %d workflows took %s", len(workflowRunIds), time.Since(start))
		}

		for _, event := range events {
			err := sendEvent(event)

			if err != nil {
				return err
			}
		}

		return nil
	}

	// start a new goroutine to handle client-side streaming
	go func() {
		for {
			req, err := server.Recv()

			if err != nil {
				cancel()
				if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
					return
				}

				s.l.Error().Err(err).Msg("could not receive message from client")
				return
			}

			if !strings.HasPrefix(req.WorkflowRunId, "id-") {
				if _, parseErr := uuid.Parse(req.WorkflowRunId); parseErr != nil {
					s.l.Warn().Err(parseErr).Msg("invalid workflow run id")
					continue
				}
			}

			acks.addWorkflowRun(req.WorkflowRunId)
		}
	}()

	// new goroutine to poll every second for finished workflow runs which are not ackd
	go func() {
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				workflowRunIds := acks.getNonAckdWorkflowRuns()

				if len(workflowRunIds) == 0 {
					continue
				}

				if err := iter(workflowRunIds); err != nil {
					s.l.Error().Err(err).Msg("could not iterate over workflow runs")
				}
			}
		}
	}()

	<-ctx.Done()

	// if err := cleanupQueue(); err != nil {
	// 	return fmt.Errorf("could not cleanup queue: %w", err)
	// }

	waitFor(&wg, 60*time.Second, s.l)

	return nil
}

func (s *DispatcherImpl) taskEventsToWorkflowRunEvent(tenantId string, finalizedWorkflowRuns []*v1.ListFinalizedWorkflowRunsResponse) ([]*contracts.WorkflowRunEvent, error) {
	res := make([]*contracts.WorkflowRunEvent, 0)

	for _, wr := range finalizedWorkflowRuns {
		status := contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED
		stepRunResults := make([]*contracts.StepRunResult, 0)

		for _, event := range wr.OutputEvents {
			res := &contracts.StepRunResult{
				StepRunId:      event.TaskExternalId,
				StepReadableId: event.StepReadableID,
				JobRunId:       event.TaskExternalId,
			}

			switch event.EventType {
			case sqlcv1.V1TaskEventTypeCOMPLETED:
				out := string(event.Output)

				res.Output = &out
			case sqlcv1.V1TaskEventTypeFAILED:
				res.Error = &event.ErrorMessage
			case sqlcv1.V1TaskEventTypeCANCELLED:
				res.Error = &event.ErrorMessage
			}

			stepRunResults = append(stepRunResults, res)
		}

		res = append(res, &contracts.WorkflowRunEvent{
			WorkflowRunId:  wr.WorkflowRunId,
			EventType:      status,
			EventTimestamp: timestamppb.New(time.Now()),
			Results:        stepRunResults,
		})
	}

	return res, nil
}

func (s *DispatcherImpl) sendStepActionEventV1(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	// if there's no retry count, we need to read it from the task, so we can't skip the cache
	skipCache := request.RetryCount == nil

	task, err := s.getSingleTask(ctx, sqlchelpers.UUIDToStr(tenant.ID), request.StepRunId, skipCache)

	if err != nil {
		return nil, fmt.Errorf("could not get task %s: %w", request.StepRunId, err)
	}

	retryCount := task.RetryCount

	if request.RetryCount != nil {
		retryCount = *request.RetryCount
	} else {
		s.l.Warn().Msg("retry count is nil, using task's current retry count")
	}

	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		return s.handleTaskStarted(ctx, task, retryCount, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_ACKNOWLEDGED:
		// TODO: IMPLEMENT
		return &contracts.ActionEventResponse{
			TenantId: sqlchelpers.UUIDToStr(ctx.Value("tenant").(*dbsqlc.Tenant).ID),
			WorkerId: request.WorkerId,
		}, nil
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.handleTaskCompleted(ctx, task, retryCount, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		return s.handleTaskFailed(ctx, task, retryCount, request)
	}

	return nil, status.Errorf(codes.InvalidArgument, "invalid step run id %s", request.StepRunId)
}

func (s *DispatcherImpl) handleTaskStarted(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	msg, err := tasktypes.MonitoringEventMessageFromActionEvent(
		tenantId,
		task.ID,
		retryCount,
		request,
	)

	if err != nil {
		return nil, err
	}

	err = s.pubBuffer.Pub(inputCtx, msgqueue.OLAP_QUEUE, msg, false)

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleTaskCompleted(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// if request.RetryCount == nil {
	// 	return nil, fmt.Errorf("retry count is required in v2")
	// }

	go func() {
		olapMsg, err := tasktypes.MonitoringEventMessageFromActionEvent(
			tenantId,
			task.ID,
			retryCount,
			request,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not create monitoring event message")
			return
		}

		err = s.pubBuffer.Pub(inputCtx, msgqueue.OLAP_QUEUE, olapMsg, false)

		if err != nil {
			s.l.Error().Err(err).Msg("could not publish to OLAP queue")
		}
	}()

	msg, err := tasktypes.CompletedTaskMessage(tenantId, task.ID, task.InsertedAt, retryCount, []byte(request.EventPayload))

	if err != nil {
		return nil, err
	}

	err = s.mqv1.SendMessage(inputCtx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleTaskFailed(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// if request.RetryCount == nil {
	// 	return nil, fmt.Errorf("retry count is required in v2")
	// }

	go func() {
		olapMsg, err := tasktypes.MonitoringEventMessageFromActionEvent(
			tenantId,
			task.ID,
			retryCount,
			request,
		)

		if err != nil {
			s.l.Error().Err(err).Msg("could not create monitoring event message")
			return
		}

		err = s.pubBuffer.Pub(inputCtx, msgqueue.OLAP_QUEUE, olapMsg, false)

		if err != nil {
			s.l.Error().Err(err).Msg("could not publish to OLAP queue")
		}
	}()

	msg, err := tasktypes.FailedTaskMessage(tenantId, task.ID, task.InsertedAt, retryCount, true, request.EventPayload)

	if err != nil {
		return nil, err
	}

	err = s.mqv1.SendMessage(inputCtx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (d *DispatcherImpl) getSingleTask(ctx context.Context, tenantId, taskExternalId string, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	return d.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskExternalId, skipCache)
}

func (d *DispatcherImpl) refreshTimeoutV1(ctx context.Context, tenant *dbsqlc.Tenant, request *contracts.RefreshTimeoutRequest) (*contracts.RefreshTimeoutResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	opts := v1.RefreshTimeoutBy{
		TaskExternalId:     request.StepRunId,
		IncrementTimeoutBy: request.IncrementTimeoutBy,
	}

	if apiErrors, err := d.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	taskRuntime, err := d.repov1.Tasks().RefreshTimeoutBy(ctx, tenantId, opts)

	if err != nil {
		return nil, err
	}

	workerId := sqlchelpers.UUIDToStr(taskRuntime.WorkerID)

	// send to the OLAP repository
	msg, err := tasktypes.MonitoringEventMessageFromInternal(
		tenantId,
		tasktypes.CreateMonitoringEventPayload{
			TaskId:         taskRuntime.TaskID,
			RetryCount:     taskRuntime.RetryCount,
			WorkerId:       &workerId,
			EventTimestamp: time.Now(),
			EventType:      sqlcv1.V1EventTypeOlapTIMEOUTREFRESHED,
			EventMessage:   fmt.Sprintf("Timeout refreshed by %s", request.IncrementTimeoutBy),
		},
	)

	if err != nil {
		return nil, err
	}

	err = d.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)

	if err != nil {
		return nil, err
	}

	return &contracts.RefreshTimeoutResponse{
		TimeoutAt: timestamppb.New(taskRuntime.TimeoutAt.Time),
	}, nil
}

func (d *DispatcherImpl) releaseSlotV1(ctx context.Context, tenant *dbsqlc.Tenant, request *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	releasedSlot, err := d.repov1.Tasks().ReleaseSlot(ctx, tenantId, request.StepRunId)

	if err != nil {
		return nil, err
	}

	workerId := sqlchelpers.UUIDToStr(releasedSlot.WorkerID)

	// send to the OLAP repository
	msg, err := tasktypes.MonitoringEventMessageFromInternal(
		tenantId,
		tasktypes.CreateMonitoringEventPayload{
			TaskId:         releasedSlot.ID,
			RetryCount:     releasedSlot.RetryCount,
			WorkerId:       &workerId,
			EventTimestamp: time.Now(),
			EventType:      sqlcv1.V1EventTypeOlapSLOTRELEASED,
		},
	)

	if err != nil {
		return nil, err
	}

	err = d.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)

	if err != nil {
		return nil, err
	}

	return &contracts.ReleaseSlotResponse{}, nil
}
