package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
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

	immediateSendFilter := &sendTimeFilter{}
	iterSendFilter := &sendTimeFilter{}

	iter := func(workflowRunIds []string) error {
		limit := 1000

		taskIdEventKeyTuples := make([]v1.TaskIdEventKeyTuple, 0, limit)
		taskExternalIds := make([]string, 0, limit)
		mapKeyToWorkflowRunIds := make(map[string]string)

		for i := 0; i < limit; i++ {
			// parse the workflow run id
			if i >= len(workflowRunIds) {
				break
			}

			if strings.HasPrefix(workflowRunIds[i], "id-") {
				taskIdStr := strings.TrimPrefix(workflowRunIds[i], "id-")

				// parse by <parent>-<index>
				parts := strings.Split(taskIdStr, "-")

				if len(parts) != 2 {
					s.l.Warn().Msgf("invalid task id %s", taskIdStr)
					continue
				}

				taskIdInt, err := strconv.ParseInt(parts[0], 10, 64)

				if err != nil {
					s.l.Warn().Err(err).Msgf("invalid task id %s", taskIdStr)
					continue
				}

				taskIdEventKeyTuples = append(taskIdEventKeyTuples, v1.TaskIdEventKeyTuple{
					Id:       taskIdInt,
					EventKey: fmt.Sprintf("%d.%s", taskIdInt, parts[1]),
				})

				taskEventKey := getTaskEventKey(taskIdInt, fmt.Sprintf("%d.%s", taskIdInt, parts[1]))

				mapKeyToWorkflowRunIds[taskEventKey] = workflowRunIds[i]
			} else {
				taskExternalIds = append(taskExternalIds, workflowRunIds[i])
			}
		}

		var events []*contracts.WorkflowRunEvent

		if len(taskIdEventKeyTuples) > 0 {
			signalEvents, err := s.repov1.Tasks().ListCompletedTaskSignals(ctx, tenantId, taskIdEventKeyTuples)

			if err != nil {
				s.l.Error().Err(err).Msg("could not get completed task signals")
				return err
			}

			resEvents, err := s.taskEventsToWorkflowRunEvent(tenantId, mapKeyToWorkflowRunIds, signalEvents)

			if err != nil {
				s.l.Error().Err(err).Msg("could not convert task events to workflow run events")
				return err
			}

			events = append(events, resEvents...)
		}

		if len(taskExternalIds) > 0 {
			// workflowRuns, err := s.repo.WorkflowRun().ListWorkflowRuns(ctx, tenantId, &repository.ListWorkflowRunsOpts{
			// 	Ids:   workflowRunIds,
			// 	Limit: &limit,
			// })

			// if err != nil {
			// 	s.l.Error().Err(err).Msg("could not get workflow runs")
			// 	return nil
			// }

			// events, err := s.toWorkflowRunEvent(tenantId, workflowRuns.Rows)

			// if err != nil {
			// 	s.l.Error().Err(err).Msg("could not convert workflow run to event")
			// 	return nil
			// }
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

			if immediateSendFilter.canSend() {
				if err := iter([]string{req.WorkflowRunId}); err != nil {
					s.l.Error().Err(err).Msg("could not iterate over workflow runs")
				}
			}
		}
	}()

	// f := func(task *msgqueue.Message) error {
	// 	wg.Add(1)
	// 	defer wg.Done()

	// 	workflowRunIds := acks.getNonAckdWorkflowRuns()

	// 	if matchedWorkflowRunId, ok := s.isMatchingWorkflowRun(task, workflowRunIds...); ok {
	// 		if immediateSendFilter.canSend() {
	// 			if err := iter([]string{matchedWorkflowRunId}); err != nil {
	// 				s.l.Error().Err(err).Msg("could not iterate over workflow runs")
	// 			}
	// 		}
	// 	}

	// 	return nil
	// }

	// // subscribe to the task queue for the tenant
	// cleanupQueue, err := s.sharedReader.Subscribe(tenantId, f)

	// if err != nil {
	// 	return err
	// }

	// new goroutine to poll every second for finished workflow runs which are not ackd
	go func() {
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !iterSendFilter.canSend() {
					continue
				}

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

func getTaskEventKey(taskId int64, eventKey string) string {
	return fmt.Sprintf("%d-%s", taskId, eventKey)
}

type stepResult struct {
	Output []byte `json:"output,omitempty"`

	StepReadableId string `json:"step_readable_id,omitempty"`

	Error string `json:"error,omitempty"`
}

type signalEventData map[string]map[string][]stepResult

func (s *DispatcherImpl) taskEventsToWorkflowRunEvent(tenantId string, keyToId map[string]string, events []*sqlcv1.V1TaskEvent) ([]*contracts.WorkflowRunEvent, error) {
	res := make([]*contracts.WorkflowRunEvent, 0)

	// TODO: this currently only does completed events
	for _, event := range events {
		parsedEventData := make(signalEventData)

		err := json.Unmarshal([]byte(event.Data), &parsedEventData)

		if err != nil {
			s.l.Error().Err(err).Msg("could not unmarshal event data")
			continue
		}

		stepRunResults := []*contracts.StepRunResult{}

		for actionKind, eventDatas := range parsedEventData {
			if actionKind != "CREATE" {
				continue
			}

			for signalKey, stepResults := range eventDatas {
				var taskExternalId string

				if strings.HasPrefix(signalKey, "task.completed.") {
					taskExternalId = strings.TrimPrefix(signalKey, "task.completed.")
				} else if strings.HasPrefix(signalKey, "task.failed.") {
					taskExternalId = strings.TrimPrefix(signalKey, "task.failed.")
				} else if strings.HasPrefix(signalKey, "task.cancelled.") {
					taskExternalId = strings.TrimPrefix(signalKey, "task.cancelled.")
				}

				if taskExternalId != "" {
					for _, stepResult := range stepResults {
						res := &contracts.StepRunResult{
							StepRunId:      taskExternalId,
							StepReadableId: stepResult.StepReadableId,
							JobRunId:       taskExternalId,
						}

						if stepResult.Output != nil {
							out := string(stepResult.Output)

							res.Output = &out
						}

						if stepResult.Error != "" {
							res.Error = &stepResult.Error
						}

						stepRunResults = append(stepRunResults, res)
					}
				}
			}
		}

		mapKey := getTaskEventKey(event.TaskID, event.EventKey.String)

		if keyToId[mapKey] == "" {
			s.l.Warn().Msgf("could not find workflow run id for key %s", mapKey)
			continue
		}

		workflowRunEvent := &contracts.WorkflowRunEvent{
			EventType:      contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
			EventTimestamp: timestamppb.New(event.CreatedAt.Time),
			WorkflowRunId:  keyToId[mapKey],
			Results:        stepRunResults,
		}

		res = append(res, workflowRunEvent)
	}

	return res, nil
}

func (s *DispatcherImpl) sendStepActionEventV1(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	if strings.HasPrefix(request.StepRunId, "id-") {
		taskIdRetryCountStr := strings.TrimPrefix(request.StepRunId, "id-")

		parts := strings.Split(taskIdRetryCountStr, "-")

		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid task id %s", request.StepRunId)
		}

		taskIdStr := parts[0]
		retryCountStr := parts[1]

		taskId, err := strconv.ParseInt(taskIdStr, 10, 64)

		if err != nil {
			return nil, fmt.Errorf("could not parse task id: %w", err)
		}

		retryCount, err := strconv.ParseInt(retryCountStr, 10, 64)

		if err != nil {
			return nil, fmt.Errorf("could not parse retry count: %w", err)
		}

		switch request.EventType {
		case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
			return s.handleTaskStarted(ctx, taskId, int32(retryCount), request)
		case contracts.StepActionEventType_STEP_EVENT_TYPE_ACKNOWLEDGED:
			return &contracts.ActionEventResponse{
				TenantId: sqlchelpers.UUIDToStr(ctx.Value("tenant").(*dbsqlc.Tenant).ID),
				WorkerId: request.WorkerId,
			}, nil
		case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
			return s.handleTaskCompleted(ctx, taskId, int32(retryCount), request)
		case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
			return s.handleTaskFailed(ctx, taskId, int32(retryCount), request)
		}
	}

	return nil, status.Errorf(codes.InvalidArgument, "invalid step run id %s", request.StepRunId)
}

func (s *DispatcherImpl) handleTaskStarted(inputCtx context.Context, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	msg, err := tasktypes.MonitoringEventMessageFromActionEvent(
		tenantId,
		taskId,
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

func (s *DispatcherImpl) handleTaskCompleted(inputCtx context.Context, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// if request.RetryCount == nil {
	// 	return nil, fmt.Errorf("retry count is required in v2")
	// }

	go func() {
		olapMsg, err := tasktypes.MonitoringEventMessageFromActionEvent(
			tenantId,
			taskId,
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

	msg, err := tasktypes.CompletedTaskMessage(tenantId, taskId, *request.RetryCount, []byte(request.EventPayload))

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

func (s *DispatcherImpl) handleTaskFailed(inputCtx context.Context, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// if request.RetryCount == nil {
	// 	return nil, fmt.Errorf("retry count is required in v2")
	// }

	go func() {
		olapMsg, err := tasktypes.MonitoringEventMessageFromActionEvent(
			tenantId,
			taskId,
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

	msg, err := tasktypes.FailedTaskMessage(tenantId, taskId, *request.RetryCount, true, request.EventPayload)

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
