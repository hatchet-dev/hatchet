package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool
}

func (worker *subscribedWorker) StartStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunForEngineRow,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if stepRun.StepRun.Input != nil {
		inputBytes = stepRun.StepRun.Input
	}

	stepName := stepRun.StepReadableId.String

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.ActionId,
		ActionPayload: string(inputBytes),
		StepName:      stepName,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.StepRun.RetryCount,
	})
}

func (worker *subscribedWorker) StartGroupKeyAction(
	ctx context.Context,
	tenantId string,
	getGroupKeyRun *dbsqlc.GetGroupKeyRunForEngineRow,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-group-key-action") // nolint:ineffassign
	defer span.End()

	inputData := getGroupKeyRun.GetGroupKeyRun.Input
	workflowRunId := sqlchelpers.UUIDToStr(getGroupKeyRun.WorkflowRunId)
	getGroupKeyRunId := sqlchelpers.UUIDToStr(getGroupKeyRun.GetGroupKeyRun.ID)

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:         tenantId,
		WorkflowRunId:    workflowRunId,
		GetGroupKeyRunId: getGroupKeyRunId,
		ActionType:       contracts.ActionType_START_GET_GROUP_KEY,
		ActionId:         getGroupKeyRun.ActionId,
		ActionPayload:    string(inputData),
	})
}

func (worker *subscribedWorker) CancelStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunForEngineRow,
) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-step-run") // nolint:ineffassign
	defer span.End()

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
		ActionType:    contracts.ActionType_CANCEL_STEP_RUN,
		StepName:      stepRun.StepReadableId.String,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.StepRun.RetryCount,
	})
}

func (s *DispatcherImpl) Register(ctx context.Context, request *contracts.WorkerRegisterRequest) (*contracts.WorkerRegisterResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received register request from ID %s with actions %v", request.WorkerName, request.Actions)

	svcs := request.Services

	if len(svcs) == 0 {
		svcs = []string{"default"}
	}

	opts := &repository.CreateWorkerOpts{
		DispatcherId: s.dispatcherId,
		Name:         request.WorkerName,
		Actions:      request.Actions,
		Services:     svcs,
	}

	if request.MaxRuns != nil {
		mr := int(*request.MaxRuns)
		opts.MaxRuns = &mr
	}

	if apiErrors, err := s.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	// create a worker in the database
	worker, err := s.repo.Worker().CreateNewWorker(ctx, tenantId, opts)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not create worker for tenant %s", tenantId)
		return nil, err
	}

	workerId := sqlchelpers.UUIDToStr(worker.ID)

	s.l.Debug().Msgf("Registered worker with ID: %s", workerId)

	// return the worker id to the worker
	return &contracts.WorkerRegisterResponse{
		TenantId:   tenantId,
		WorkerId:   workerId,
		WorkerName: worker.Name,
	}, nil
}

// Subscribe handles a subscribe request from a client
func (s *DispatcherImpl) Listen(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenServer) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	sessionId := uuid.New().String()

	s.l.Debug().Msgf("Received subscribe request from ID: %s", request.WorkerId)

	ctx := stream.Context()

	worker, err := s.repo.Worker().GetWorkerForEngine(ctx, tenantId, request.WorkerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := !worker.DispatcherId.Valid || sqlchelpers.UUIDToStr(worker.DispatcherId) != s.dispatcherId

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repo.Worker().UpdateWorker(ctx, tenantId, request.WorkerId, &repository.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	fin := make(chan bool)

	s.workers.Add(request.WorkerId, sessionId, &subscribedWorker{stream: stream, finished: fin})

	defer func() {
		// non-blocking send
		select {
		case fin <- true:
		default:
		}

		s.workers.DeleteForSession(request.WorkerId, sessionId)
	}()

	// update the worker with a last heartbeat time every 5 seconds as long as the worker is connected
	go func() {
		timer := time.NewTicker(100 * time.Millisecond)

		// set the last heartbeat to 6 seconds ago so the first heartbeat is sent immediately
		lastHeartbeat := time.Now().UTC().Add(-6 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)
				return
			case <-fin:
				s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)
				return
			case <-timer.C:
				if now := time.Now().UTC(); lastHeartbeat.Add(4 * time.Second).Before(now) {
					s.l.Debug().Msgf("updating worker %s heartbeat", request.WorkerId)

					_, err := s.repo.Worker().UpdateWorker(ctx, tenantId, request.WorkerId, &repository.UpdateWorkerOpts{
						LastHeartbeatAt: &now,
						IsActive:        repository.BoolPtr(true),
					})

					if err != nil {
						s.l.Error().Err(err).Msgf("could not update worker %s heartbeat", request.WorkerId)
						return
					}

					lastHeartbeat = time.Now().UTC()
				}
			}
		}
	}()

	// Keep the connection alive for sending messages
	for {
		select {
		case <-fin:
			s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)
			return nil
		case <-ctx.Done():
			s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)
			return nil
		}
	}
}

// ListenV2 is like Listen, but implementation does not include heartbeats. This should only used by SDKs
// against engine version v0.18.1+
func (s *DispatcherImpl) ListenV2(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenV2Server) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	sessionId := uuid.New().String()

	ctx := stream.Context()

	s.l.Debug().Msgf("Received subscribe request from ID: %s", request.WorkerId)

	worker, err := s.repo.Worker().GetWorkerForEngine(ctx, tenantId, request.WorkerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := !worker.DispatcherId.Valid || sqlchelpers.UUIDToStr(worker.DispatcherId) != s.dispatcherId

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repo.Worker().UpdateWorker(ctx, tenantId, request.WorkerId, &repository.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	sessionEstablished := time.Now().UTC()

	_, err = s.repo.Worker().UpdateWorkerActiveStatus(ctx, tenantId, request.WorkerId, true, sessionEstablished)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not update worker %s active status", request.WorkerId)
		return err
	}

	fin := make(chan bool)

	s.workers.Add(request.WorkerId, sessionId, &subscribedWorker{stream: stream, finished: fin})

	defer func() {
		// non-blocking send
		select {
		case fin <- true:
		default:
		}

		s.workers.DeleteForSession(request.WorkerId, sessionId)
	}()

	// Keep the connection alive for sending messages
	for {
		select {
		case <-fin:
			s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)

			_, err = s.repo.Worker().UpdateWorkerActiveStatus(ctx, tenantId, request.WorkerId, false, sessionEstablished)

			if err != nil {
				s.l.Error().Err(err).Msgf("could not update worker %s active status", request.WorkerId)
				return err
			}

			return nil
		case <-ctx.Done():
			s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			_, err = s.repo.Worker().UpdateWorkerActiveStatus(ctx, tenantId, request.WorkerId, false, sessionEstablished)

			if err != nil {
				s.l.Error().Err(err).Msgf("could not update worker %s active status", request.WorkerId)
				return err
			}

			return nil
		}
	}
}

const HeartbeatInterval = 4 * time.Second

// Heartbeat is used to update the last heartbeat time for a worker
func (s *DispatcherImpl) Heartbeat(ctx context.Context, req *contracts.HeartbeatRequest) (*contracts.HeartbeatResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	heartbeatAt := time.Now().UTC()

	s.l.Debug().Msgf("Received heartbeat request from ID: %s", req.WorkerId)

	// if heartbeat time is greater than expected heartbeat interval, show a warning
	if req.HeartbeatAt.AsTime().Before(heartbeatAt.Add(-1 * HeartbeatInterval)) {
		s.l.Warn().Msgf("heartbeat time is greater than expected heartbeat interval")
	}

	worker, err := s.repo.Worker().GetWorkerForEngine(ctx, tenantId, req.WorkerId)

	if err != nil {
		return nil, err
	}

	// if we haven't seen the dispatcher for 6 seconds (one interval plus latency), reject the heartbeat as the client
	// should reconnect
	if worker.DispatcherLastHeartbeatAt.Time.Before(time.Now().Add(-6 * time.Second)) {
		return nil, status.Errorf(codes.FailedPrecondition, "Heartbeat rejected, worker stream for %s is not active", req.WorkerId)
	}

	if worker.LastListenerEstablished.Valid && !worker.IsActive {
		return nil, status.Errorf(codes.FailedPrecondition, "Heartbeat rejected, worker stream for %s is not active", req.WorkerId)
	}

	_, err = s.repo.Worker().UpdateWorker(ctx, tenantId, req.WorkerId, &repository.UpdateWorkerOpts{
		// use the system time for heartbeat
		LastHeartbeatAt: &heartbeatAt,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.HeartbeatResponse{}, nil
}

func (s *DispatcherImpl) ReleaseSlot(ctx context.Context, req *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if req.StepRunId == "" {
		return nil, fmt.Errorf("step run id is required")
	}

	err := s.repo.StepRun().ReleaseStepRunSemaphore(ctx, tenantId, req.StepRunId)

	if err != nil {
		return nil, err
	}

	return &contracts.ReleaseSlotResponse{}, nil
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) SubscribeToWorkflowEvents(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received subscribe request for workflow: %s", request.WorkflowRunId)

	q, err := msgqueue.TenantEventConsumerQueue(tenantId)

	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// if the workflow run is in a final state, hang up the connection
	workflowRun, err := s.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, request.WorkflowRunId)

	if err != nil {
		if errors.Is(err, repository.ErrWorkflowRunNotFound) {
			return status.Errorf(codes.NotFound, "workflow run %s not found", request.WorkflowRunId)
		}

		return err
	}

	if repository.IsFinalWorkflowRunStatus(workflowRun.WorkflowRun.Status) {
		return nil
	}

	wg := sync.WaitGroup{}

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		e, err := s.tenantTaskToWorkflowEvent(task, tenantId, request.WorkflowRunId)

		if err != nil {
			s.l.Error().Err(err).Msgf("could not convert task to workflow event")
			return nil
		} else if e == nil {
			return nil
		}

		// send the task to the client
		err = stream.Send(e)

		if err != nil {
			cancel() // FIXME is this necessary?
			s.l.Error().Err(err).Msgf("could not send workflow event to client")
			return nil
		}

		if e.Hangup {
			cancel()
		}

		return nil
	}

	// subscribe to the task queue for the tenant
	cleanupQueue, err := s.mq.Subscribe(q, msgqueue.NoOpHook, f)

	if err != nil {
		return err
	}

	<-ctx.Done()
	if err := cleanupQueue(); err != nil {
		return fmt.Errorf("could not cleanup queue: %w", err)
	}

	waitFor(&wg, 60*time.Second, s.l)

	return nil
}

// map of workflow run ids to whether the workflow runs are finished and have sent a message
// that the workflow run is finished
type workflowRunAcks struct {
	acks map[string]bool
	mu   sync.RWMutex
}

func (w *workflowRunAcks) addWorkflowRun(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.acks[id] = false
}

func (w *workflowRunAcks) getNonAckdWorkflowRuns() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ids := make([]string, 0, len(w.acks))

	for id := range w.acks {
		if !w.acks[id] {
			ids = append(ids, id)
		}
	}

	return ids
}

func (w *workflowRunAcks) ackWorkflowRun(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.acks[id] = true
}

type sendTimeFilter struct {
	mu sync.Mutex
}

func (s *sendTimeFilter) canSend() bool {
	if !s.mu.TryLock() {
		return false
	}

	go func() {
		time.Sleep(time.Second - 10*time.Millisecond)
		s.mu.Unlock()
	}()

	return true
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) SubscribeToWorkflowRuns(server contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	tenant := server.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received subscribe request for tenant: %s", tenantId)

	acks := &workflowRunAcks{
		acks: make(map[string]bool),
	}

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	// subscribe to the task queue for the tenant
	q, err := msgqueue.TenantEventConsumerQueue(tenantId)

	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	sendEvent := func(e *contracts.WorkflowRunEvent) error {
		// send the task to the client
		err := server.Send(e)

		if err != nil {
			cancel() // FIXME is this necessary?
			s.l.Error().Err(err).Msgf("could not send workflow event to client")
			return err
		}

		acks.ackWorkflowRun(e.WorkflowRunId)

		return nil
	}

	immediateSendFilter := &sendTimeFilter{}
	iterSendFilter := &sendTimeFilter{}

	iter := func(workflowRunIds []string) error {
		limit := 1000

		workflowRuns, err := s.repo.WorkflowRun().ListWorkflowRuns(ctx, tenantId, &repository.ListWorkflowRunsOpts{
			Ids:   workflowRunIds,
			Limit: &limit,
		})

		if err != nil {
			s.l.Error().Err(err).Msg("could not get workflow runs")
			return nil
		}

		events, err := s.toWorkflowRunEvent(tenantId, workflowRuns.Rows)

		if err != nil {
			s.l.Error().Err(err).Msg("could not convert workflow run to event")
			return nil
		} else if events == nil {
			return nil
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

			acks.addWorkflowRun(req.WorkflowRunId)

			if immediateSendFilter.canSend() {
				if err := iter([]string{req.WorkflowRunId}); err != nil {
					s.l.Error().Err(err).Msg("could not iterate over workflow runs")
				}
			}
		}
	}()

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		workflowRunIds := acks.getNonAckdWorkflowRuns()

		if matchedWorkflowRunId, ok := s.isMatchingWorkflowRun(task, workflowRunIds...); ok {
			if immediateSendFilter.canSend() {
				if err := iter([]string{matchedWorkflowRunId}); err != nil {
					s.l.Error().Err(err).Msg("could not iterate over workflow runs")
				}
			}
		}

		return nil
	}

	// subscribe to the task queue for the tenant
	cleanupQueue, err := s.mq.Subscribe(q, msgqueue.NoOpHook, f)

	if err != nil {
		return err
	}

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

	if err := cleanupQueue(); err != nil {
		return fmt.Errorf("could not cleanup queue: %w", err)
	}

	waitFor(&wg, 60*time.Second, s.l)

	return nil
}

func waitFor(wg *sync.WaitGroup, timeout time.Duration, l *zerolog.Logger) {
	done := make(chan struct{})

	go func() {
		wg.Wait()
		defer close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		l.Error().Msg("timed out waiting for wait group")
	}
}

func (s *DispatcherImpl) SendStepActionEvent(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		return s.handleStepRunStarted(ctx, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.handleStepRunCompleted(ctx, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		return s.handleStepRunFailed(ctx, request)
	}

	return nil, fmt.Errorf("unknown event type %s", request.EventType)
}

func (s *DispatcherImpl) SendGroupKeyActionEvent(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	switch request.EventType {
	case contracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_STARTED:
		return s.handleGetGroupKeyRunStarted(ctx, request)
	case contracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_COMPLETED:
		return s.handleGetGroupKeyRunCompleted(ctx, request)
	case contracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_FAILED:
		return s.handleGetGroupKeyRunFailed(ctx, request)
	}

	return nil, fmt.Errorf("unknown event type %s", request.EventType)
}

func (s *DispatcherImpl) PutOverridesData(ctx context.Context, request *contracts.OverridesData) (*contracts.OverridesDataResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// ensure step run id
	if request.StepRunId == "" {
		return nil, fmt.Errorf("step run id is required")
	}

	opts := &repository.UpdateStepRunOverridesDataOpts{
		OverrideKey: request.Path,
		Data:        []byte(request.Value),
	}

	if request.CallerFilename != "" {
		opts.CallerFile = &request.CallerFilename
	}

	_, err := s.repo.StepRun().UpdateStepRunOverridesData(ctx, tenantId, request.StepRunId, opts)

	if err != nil {
		return nil, err
	}

	return &contracts.OverridesDataResponse{}, nil
}

func (s *DispatcherImpl) Unsubscribe(ctx context.Context, request *contracts.WorkerUnsubscribeRequest) (*contracts.WorkerUnsubscribeResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// remove the worker from the connection pool
	s.workers.Delete(request.WorkerId)

	return &contracts.WorkerUnsubscribeResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (d *DispatcherImpl) RefreshTimeout(ctx context.Context, request *contracts.RefreshTimeoutRequest) (*contracts.RefreshTimeoutResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	opts := repository.RefreshTimeoutBy{
		IncrementTimeoutBy: request.IncrementTimeoutBy,
	}

	if apiErrors, err := d.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	stepRun, err := d.repo.StepRun().RefreshTimeoutBy(ctx, tenantId, request.StepRunId, opts)

	if err != nil {
		return nil, err
	}

	timeoutAt := &timestamppb.Timestamp{
		Seconds: stepRun.TimeoutAt.Time.Unix(),
		Nanos:   int32(stepRun.TimeoutAt.Time.Nanosecond()),
	}

	return &contracts.RefreshTimeoutResponse{
		TimeoutAt: timeoutAt,
	}, nil

}

func (s *DispatcherImpl) handleStepRunStarted(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step started event for step run %s", request.StepRunId)

	startedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskPayload{
		StepRunId: request.StepRunId,
		StartedAt: startedAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "step-run-started",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleStepRunCompleted(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step completed event for step run %s", request.StepRunId)

	// verify that the event payload can be unmarshalled into a map type
	if request.EventPayload != "" {
		res := make(map[string]interface{})

		if err := json.Unmarshal([]byte(request.EventPayload), &res); err != nil {
			// if the payload starts with a [, then it is an array which we don't currently support
			if request.EventPayload[0] == '[' {
				return nil, status.Errorf(codes.InvalidArgument, "Return value is an array, which is not supported")
			}

			return nil, status.Errorf(codes.InvalidArgument, "Return value is not a valid JSON object")
		}
	}

	finishedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskPayload{
		StepRunId:      request.StepRunId,
		FinishedAt:     finishedAt.Format(time.RFC3339),
		StepOutputData: request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "step-run-finished",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleStepRunFailed(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step failed event for step run %s", request.StepRunId)

	failedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskPayload{
		StepRunId: request.StepRunId,
		FailedAt:  failedAt.Format(time.RFC3339),
		Error:     request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "step-run-failed",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleGetGroupKeyRunStarted(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step started event for step run %s", request.GetGroupKeyRunId)

	startedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunStartedTaskPayload{
		GetGroupKeyRunId: request.GetGroupKeyRunId,
		StartedAt:        startedAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunStartedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.WORKFLOW_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "get-group-key-run-started",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleGetGroupKeyRunCompleted(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step completed event for step run %s", request.GetGroupKeyRunId)

	finishedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunFinishedTaskPayload{
		GetGroupKeyRunId: request.GetGroupKeyRunId,
		FinishedAt:       finishedAt.Format(time.RFC3339),
		GroupKey:         request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunFinishedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.WORKFLOW_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "get-group-key-run-finished",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleGetGroupKeyRunFailed(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received step failed event for step run %s", request.GetGroupKeyRunId)

	failedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunFailedTaskPayload{
		GetGroupKeyRunId: request.GetGroupKeyRunId,
		FailedAt:         failedAt.Format(time.RFC3339),
		Error:            request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.GetGroupKeyRunFailedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err := s.mq.AddMessage(ctx, msgqueue.WORKFLOW_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "get-group-key-run-failed",
		Payload:  payload,
		Metadata: metadata,
		Retries:  3,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) tenantTaskToWorkflowEvent(task *msgqueue.Message, tenantId string, workflowRunIds ...string) (*contracts.WorkflowEvent, error) {
	workflowEvent := &contracts.WorkflowEvent{}

	var stepRunId string

	switch task.ID {
	case "step-run-started":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED
	case "step-run-finished":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
		workflowEvent.EventPayload = task.Payload["step_output_data"].(string)
	case "step-run-failed":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED
		workflowEvent.EventPayload = task.Payload["error"].(string)
	case "step-run-cancelled":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED
	case "step-run-timed-out":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_TIMED_OUT
	case "step-run-stream-event":
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM
	case "workflow-run-finished":
		workflowRunId := task.Payload["workflow_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN
		workflowEvent.ResourceId = workflowRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
		workflowEvent.Hangup = true
	}

	if workflowEvent.ResourceType == contracts.ResourceType_RESOURCE_TYPE_STEP_RUN {
		// determine if this step run matches the workflow run id
		stepRun, err := s.repo.StepRun().GetStepRunForEngine(context.Background(), tenantId, stepRunId)

		if err != nil {
			return nil, err
		}

		if !contains(workflowRunIds, sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)) {
			// this is an expected error, so we don't return it
			return nil, nil
		}

		workflowEvent.StepRetries = &stepRun.StepRetries
		workflowEvent.RetryCount = &stepRun.StepRun.RetryCount

		if workflowEvent.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {
			streamEventId, err := strconv.ParseInt(task.Metadata["stream_event_id"].(string), 10, 64)
			if err != nil {
				return nil, err
			}

			streamEvent, err := s.repo.StreamEvent().GetStreamEvent(context.Background(), tenantId, streamEventId)

			if err != nil {
				return nil, err
			}

			workflowEvent.EventPayload = string(streamEvent.Message)
		}

	} else if workflowEvent.ResourceType == contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN {
		if !contains(workflowRunIds, workflowEvent.ResourceId) {
			return nil, nil
		}

		workflowEvent.Hangup = true
	}

	return workflowEvent, nil
}

func (s *DispatcherImpl) isMatchingWorkflowRun(task *msgqueue.Message, workflowRunIds ...string) (string, bool) {
	if task.ID != "workflow-run-finished" {
		return "", false
	}

	workflowRunId := task.Payload["workflow_run_id"].(string)

	if contains(workflowRunIds, workflowRunId) {
		return workflowRunId, true
	}

	return "", false
}

func (s *DispatcherImpl) toWorkflowRunEvent(tenantId string, workflowRuns []*dbsqlc.ListWorkflowRunsRow) ([]*contracts.WorkflowRunEvent, error) {
	workflowRunIds := make([]string, 0)

	for _, workflowRun := range workflowRuns {
		if workflowRun.WorkflowRun.Status != dbsqlc.WorkflowRunStatusFAILED && workflowRun.WorkflowRun.Status != dbsqlc.WorkflowRunStatusSUCCEEDED {
			continue
		}

		workflowRunId := sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.ID)

		workflowRunIds = append(workflowRunIds, workflowRunId)
	}

	res := make([]*contracts.WorkflowRunEvent, 0)

	// get step run results for each workflow run
	mappedStepRunResults, err := s.getStepResultsForWorkflowRun(tenantId, workflowRunIds)

	if err != nil {
		return nil, err
	}

	for workflowRunId, stepRunResults := range mappedStepRunResults {
		res = append(res, &contracts.WorkflowRunEvent{
			WorkflowRunId:  workflowRunId,
			EventType:      contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
			EventTimestamp: timestamppb.Now(),
			Results:        stepRunResults,
		})
	}

	return res, nil
}

func (s *DispatcherImpl) getStepResultsForWorkflowRun(tenantId string, workflowRunIds []string) (map[string][]*contracts.StepRunResult, error) {
	stepRuns, err := s.repo.StepRun().ListStepRuns(context.Background(), tenantId, &repository.ListStepRunsOpts{
		WorkflowRunIds: workflowRunIds,
	})

	if err != nil {
		return nil, err
	}

	res := make(map[string][]*contracts.StepRunResult)

	for _, stepRun := range stepRuns {
		resStepRun := &contracts.StepRunResult{
			StepRunId:      sqlchelpers.UUIDToStr(stepRun.StepRun.ID),
			StepReadableId: stepRun.StepReadableId.String,
			JobRunId:       sqlchelpers.UUIDToStr(stepRun.JobRunId),
		}

		if stepRun.StepRun.Error.Valid {
			resStepRun.Error = &stepRun.StepRun.Error.String
		}

		if stepRun.StepRun.CancelledReason.Valid {
			errString := fmt.Sprintf("this step run was cancelled due to %s", stepRun.StepRun.CancelledReason.String)
			resStepRun.Error = &errString
		}

		if stepRun.StepRun.Output != nil {
			resStepRun.Output = repository.StringPtr(string(stepRun.StepRun.Output))
		}

		workflowRunId := sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)

		if currResults, ok := res[workflowRunId]; ok {
			res[workflowRunId] = append(currResults, resStepRun)
		} else {
			res[workflowRunId] = []*contracts.StepRunResult{resStepRun}
		}
	}

	return res, nil
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
