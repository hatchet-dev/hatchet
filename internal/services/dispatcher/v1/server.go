package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (d *DispatcherServiceImpl) RegisterDurableEvent(ctx context.Context, req *contracts.RegisterDurableEventRequest) (*contracts.RegisterDurableEventResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	taskId, err := uuid.Parse(req.TaskId)

	if err != nil {
		d.l.Error().Msgf("task id %s is not a valid uuid", req.TaskId)
		return nil, status.Error(codes.InvalidArgument, "task id is not a valid uuid")
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

	if err != nil {
		return nil, err
	}

	createConditionOpts := make([]v1.CreateExternalSignalConditionOpt, 0)

	for _, condition := range req.Conditions.SleepConditions {
		orGroupId, err := uuid.Parse(condition.Base.OrGroupId)

		if err != nil {
			d.l.Error().Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
			return nil, status.Error(codes.InvalidArgument, "or group id is not a valid uuid")
		}

		createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
			Kind:            v1.CreateExternalSignalConditionKindSLEEP,
			ReadableDataKey: condition.Base.ReadableDataKey,
			OrGroupId:       orGroupId,
			SleepFor:        &condition.SleepFor,
		})
	}

	for _, condition := range req.Conditions.UserEventConditions {
		orGroupId, err := uuid.Parse(condition.Base.OrGroupId)

		if err != nil {
			d.l.Error().Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
			return nil, status.Error(codes.InvalidArgument, "or group id is not a valid uuid")
		}

		createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
			Kind:            v1.CreateExternalSignalConditionKindUSEREVENT,
			ReadableDataKey: condition.Base.ReadableDataKey,
			OrGroupId:       orGroupId,
			UserEventKey:    &condition.UserEventKey,
			Expression:      condition.Base.Expression,
		})
	}

	createMatchOpts := make([]v1.ExternalCreateSignalMatchOpts, 0)

	createMatchOpts = append(createMatchOpts, v1.ExternalCreateSignalMatchOpts{
		Conditions:           createConditionOpts,
		SignalTaskId:         task.ID,
		SignalTaskInsertedAt: task.InsertedAt,
		SignalExternalId:     task.ExternalID,
		SignalKey:            req.SignalKey,
	})

	err = d.repo.Matches().RegisterSignalMatchConditions(ctx, tenantId, createMatchOpts)

	if err != nil {
		return nil, err
	}

	return &contracts.RegisterDurableEventResponse{}, nil
}

// map of durable signals to whether the durable signals are finished and have sent a message
// that the signal is finished
type durableEventAcks struct {
	acks map[v1.TaskIdInsertedAtSignalKey]uuid.UUID
	mu   sync.RWMutex
}

func (w *durableEventAcks) addEvent(taskExternalId uuid.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz, signalKey string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.acks[v1.TaskIdInsertedAtSignalKey{
		Id:         taskId,
		InsertedAt: taskInsertedAt,
		SignalKey:  signalKey,
	}] = taskExternalId
}

func (w *durableEventAcks) getNonAckdEvents() []v1.TaskIdInsertedAtSignalKey {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ids := make([]v1.TaskIdInsertedAtSignalKey, 0, len(w.acks))

	for id := range w.acks {
		if w.acks[id] != uuid.Nil {
			ids = append(ids, id)
		}
	}

	return ids
}

func (w *durableEventAcks) getExternalId(taskId int64, taskInsertedAt pgtype.Timestamptz, signalKey string) uuid.UUID {
	w.mu.Lock()
	defer w.mu.Unlock()

	k := v1.TaskIdInsertedAtSignalKey{
		Id:         taskId,
		InsertedAt: taskInsertedAt,
		SignalKey:  signalKey,
	}

	res := w.acks[k]

	return res
}

func (w *durableEventAcks) ackEvent(taskId int64, taskInsertedAt pgtype.Timestamptz, signalKey string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	k := v1.TaskIdInsertedAtSignalKey{
		Id:         taskId,
		InsertedAt: taskInsertedAt,
		SignalKey:  signalKey,
	}

	delete(w.acks, k)
}

func (d *DispatcherServiceImpl) ListenForDurableEvent(server contracts.V1Dispatcher_ListenForDurableEventServer) error {
	tenant := server.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	acks := &durableEventAcks{
		acks: make(map[v1.TaskIdInsertedAtSignalKey]uuid.UUID),
	}

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	wg := sync.WaitGroup{}
	sendMu := sync.Mutex{}
	iterMu := sync.Mutex{}

	sendEvent := func(e *v1.V1TaskEventWithPayload) error {
		// FIXME: check max size of msg
		// results := cleanResults(e.Results)

		// if results == nil {
		// 	s.l.Warn().Msgf("results size for workflow run %s exceeds 3MB and cannot be reduced", e.WorkflowRunId)
		// 	e.Results = nil
		// }

		externalId := acks.getExternalId(e.TaskID, e.TaskInsertedAt, e.EventKey.String)

		if externalId == uuid.Nil {
			d.l.Warn().Msgf("could not find external id for task %d, signal key %s", e.TaskID, e.EventKey.String)
			return fmt.Errorf("could not find external id for task %d, signal key %s", e.TaskID, e.EventKey.String)
		}

		// send the task to the client
		sendMu.Lock()
		err := server.Send(&contracts.DurableEvent{
			TaskId:    externalId.String(),
			SignalKey: e.EventKey.String,
			Data:      e.Payload,
		})
		sendMu.Unlock()

		if err != nil {
			d.l.Error().Err(err).Msgf("could not send durable event for task %s, key %s", externalId, e.EventKey.String)
			return err
		}

		acks.ackEvent(e.TaskID, e.TaskInsertedAt, e.EventKey.String)

		return nil
	}

	iter := func(signalEvents []v1.TaskIdInsertedAtSignalKey) error {
		if len(signalEvents) == 0 {
			return nil
		}

		if !iterMu.TryLock() {
			d.l.Warn().Msg("could not acquire lock")
			return nil
		}

		defer iterMu.Unlock()

		signalEvents = signalEvents[:min(1000, len(signalEvents))]
		start := time.Now()

		dbEvents, err := d.repo.Tasks().ListSignalCompletedEvents(ctx, tenantId, signalEvents)

		if err != nil {
			d.l.Error().Err(err).Msg("could not list signal completed events")
			return err
		}

		for _, dbEvent := range dbEvents {
			err := sendEvent(dbEvent)

			if err != nil {
				return err
			}
		}

		if time.Since(start) > 100*time.Millisecond {
			d.l.Warn().Msgf("list durable events for %d signals took %s", len(signalEvents), time.Since(start))
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

				d.l.Error().Err(err).Msg("could not receive message from client")
				return
			}

			taskId, err := uuid.Parse(req.TaskId)

			if err != nil {
				d.l.Warn().Msgf("task id %s is not a valid uuid", req.TaskId)
				continue
			}

			// FIXME: buffer/batch this to make it more efficient
			task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

			if err != nil {
				d.l.Error().Err(err).Msg("could not get task by external id")
				continue
			}

			acks.addEvent(taskId, task.ID, task.InsertedAt, req.SignalKey)
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
				signalEvents := acks.getNonAckdEvents()

				if len(signalEvents) == 0 {
					continue
				}

				if err := iter(signalEvents); err != nil {
					d.l.Error().Err(err).Msg("could not iterate over workflow runs")
				}
			}
		}
	}()

	<-ctx.Done()

	// if err := cleanupQueue(); err != nil {
	// 	return fmt.Errorf("could not cleanup queue: %w", err)
	// }

	waitFor(&wg, 60*time.Second, d.l)

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

type durableTaskInvocationKey struct {
	taskExternalId  string
	invocationCount int64
}

type durableTaskInvocation struct {
	server   contracts.V1Dispatcher_DurableTaskServer
	tenantId uuid.UUID
	workerId string
	l        *zerolog.Logger

	sendMu sync.Mutex

	nextNodeIdMu sync.Mutex
	nextNodeId   map[durableTaskInvocationKey]int64
}

func (inv *durableTaskInvocation) getNextNodeId(taskExternalId string, invocationCount int64) int64 {
	inv.nextNodeIdMu.Lock()
	defer inv.nextNodeIdMu.Unlock()

	if inv.nextNodeId == nil {
		inv.nextNodeId = make(map[durableTaskInvocationKey]int64)
	}

	key := durableTaskInvocationKey{taskExternalId: taskExternalId, invocationCount: invocationCount}
	nodeId := inv.nextNodeId[key] + 1
	inv.nextNodeId[key] = nodeId
	return nodeId
}

func (s *durableTaskInvocation) send(resp *contracts.DurableTaskResponse) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.server.Send(resp)
}

func (d *DispatcherServiceImpl) DurableTask(server contracts.V1Dispatcher_DurableTaskServer) error {
	tenant := server.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	invocation := &durableTaskInvocation{
		server:   server,
		tenantId: tenantId,
		l:        d.l,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := server.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}
			d.l.Error().Err(err).Msg("error receiving durable task request")
			return err
		}

		if err := d.handleDurableTaskRequest(ctx, invocation, req); err != nil {
			d.l.Error().Err(err).Msg("error handling durable task request")
			// Continue processing other requests rather than closing the stream
		}
	}
}

func (d *DispatcherServiceImpl) handleDurableTaskRequest(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRequest,
) error {
	switch msg := req.GetMessage().(type) {
	case *contracts.DurableTaskRequest_RegisterWorker:
		return d.handleRegisterWorker(ctx, invocation, msg.RegisterWorker)

	case *contracts.DurableTaskRequest_Event:
		return d.handleDurableTaskEvent(ctx, invocation, msg.Event)

	case *contracts.DurableTaskRequest_RegisterCallback:
		return d.handleRegisterCallback(ctx, invocation, msg.RegisterCallback)

	case *contracts.DurableTaskRequest_EvictInvocation:
		return d.handleEvictInvocation(ctx, invocation, msg.EvictInvocation)

	default:
		return status.Errorf(codes.InvalidArgument, "unknown message type: %T", msg)
	}
}

func (d *DispatcherServiceImpl) handleRegisterWorker(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRequestRegisterWorker,
) error {
	invocation.workerId = req.WorkerId

	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &contracts.DurableTaskResponseRegisterWorker{
				WorkerId: req.WorkerId,
			},
		},
	})
}

func getDurableTaskSignalKey(taskExternalId string, nodeId int64) string {
	return fmt.Sprintf("durable:%s:%d", taskExternalId, nodeId)
}

func (d *DispatcherServiceImpl) handleDurableTaskEvent(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskEventRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	nodeId := invocation.getNextNodeId(req.DurableTaskExternalId, req.InvocationCount)
	existingEntry, err := d.repo.DurableEvents().GetEventLogEntry(ctx, invocation.tenantId, task.ID, task.InsertedAt, nodeId)

	if err == nil && existingEntry != nil {

		return invocation.send(&contracts.DurableTaskResponse{
			Message: &contracts.DurableTaskResponse_TriggerAck{
				TriggerAck: &contracts.DurableTaskEventAckResponse{
					InvocationCount:       req.InvocationCount,
					DurableTaskExternalId: req.DurableTaskExternalId,
					NodeId:                nodeId,
				},
			},
		})
	}

	logFile, err := d.repo.DurableEvents().GetOrCreateEventLogFileForTask(ctx, invocation.tenantId, task.ID, task.InsertedAt)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get or create event log file: %v", err)
	}

	var entryKind sqlcv1.V1DurableEventLogEntryKind
	var triggeredRunExternalId *uuid.UUID
	entryData := req.Payload

	switch req.Kind {
	case contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_WAIT_FOR:
		entryKind = sqlcv1.V1DurableEventLogEntryKindWAITFORSTARTED
	case contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_RUN:
		entryKind = sqlcv1.V1DurableEventLogEntryKindRUNTRIGGERED

		spawnedChild, err := d.spawnChildWorkflow(ctx, invocation.tenantId, task, nodeId, req)

		if err != nil {
			return status.Errorf(codes.Internal, "failed to spawn child: %v", err)
		}

		triggeredRunExternalId = &spawnedChild.ChildExternalId

		if err != nil {
			d.l.Error().Err(err).Msg("failed to spawn child workflow")
			return status.Errorf(codes.Internal, "failed to spawn child workflow: %v", err)
		}
	case contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_MEMO:
		entryKind = sqlcv1.V1DurableEventLogEntryKindMEMOSTARTED
	default:
		return status.Errorf(codes.InvalidArgument, "unsupported event kind: %v", req.Kind)
	}

	now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())
	externalId := uuid.New()
	_, err = d.repo.DurableEvents().CreateEventLogEntries(ctx, []v1.CreateEventLogEntryOpts{{
		TenantId:               invocation.tenantId,
		ExternalId:             externalId,
		DurableTaskId:          task.ID,
		DurableTaskInsertedAt:  task.InsertedAt,
		InsertedAt:             now,
		Kind:                   entryKind,
		NodeId:                 nodeId,
		ParentNodeId:           logFile.LatestNodeID,
		BranchId:               logFile.LatestBranchID,
		Data:                   entryData,
		TriggeredRunExternalId: triggeredRunExternalId,
	}})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to create event log entry: %v", err)
	}

	if req.Kind == contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_WAIT_FOR && req.WaitForConditions != nil {
		signalKey := getDurableTaskSignalKey(req.DurableTaskExternalId, nodeId)

		createConditionOpts := make([]v1.CreateExternalSignalConditionOpt, 0)

		for _, condition := range req.WaitForConditions.SleepConditions {
			orGroupId, err := uuid.Parse(condition.Base.OrGroupId)
			if err != nil {
				d.l.Error().Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
				return status.Error(codes.InvalidArgument, "or group id is not a valid uuid")
			}

			createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
				Kind:            v1.CreateExternalSignalConditionKindSLEEP,
				ReadableDataKey: condition.Base.ReadableDataKey,
				OrGroupId:       orGroupId,
				SleepFor:        &condition.SleepFor,
			})
		}

		for _, condition := range req.WaitForConditions.UserEventConditions {
			orGroupId, err := uuid.Parse(condition.Base.OrGroupId)
			if err != nil {
				d.l.Error().Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
				return status.Error(codes.InvalidArgument, "or group id is not a valid uuid")
			}

			createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
				Kind:            v1.CreateExternalSignalConditionKindUSEREVENT,
				ReadableDataKey: condition.Base.ReadableDataKey,
				OrGroupId:       orGroupId,
				UserEventKey:    &condition.UserEventKey,
				Expression:      condition.Base.Expression,
			})
		}

		if len(createConditionOpts) > 0 {
			createMatchOpts := []v1.ExternalCreateSignalMatchOpts{{
				Conditions:           createConditionOpts,
				SignalTaskId:         task.ID,
				SignalTaskInsertedAt: task.InsertedAt,
				SignalExternalId:     task.ExternalID,
				SignalKey:            signalKey,
			}}

			err = d.repo.Matches().RegisterSignalMatchConditions(ctx, invocation.tenantId, createMatchOpts)
			if err != nil {
				d.l.Error().Err(err).Msg("failed to register signal match conditions")
				return status.Errorf(codes.Internal, "failed to register signal match conditions: %v", err)
			}
		}
	}

	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_TriggerAck{
			TriggerAck: &contracts.DurableTaskEventAckResponse{
				InvocationCount:       req.InvocationCount,
				DurableTaskExternalId: req.DurableTaskExternalId,
				NodeId:                nodeId,
			},
		},
	})
}

type spawnChildWorkflowResult struct {
	ChildExternalId uuid.UUID
	WasSkipped      bool
}

func (d *DispatcherServiceImpl) spawnChildWorkflow(
	ctx context.Context,
	tenantId uuid.UUID,
	parentTask *sqlcv1.FlattenExternalIdsRow,
	nodeId int64,
	req *contracts.DurableTaskEventRequest,
) (*spawnChildWorkflowResult, error) {
	if req.TriggerOpts == nil {
		return nil, fmt.Errorf("trigger options are required for spawning a child workflow")
	}

	triggerOpt, err := d.repo.Triggers().NewTriggerOpt(ctx, tenantId, req.TriggerOpts, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create trigger options: %w", err)
	}

	if triggerOpt.ShouldSkip {
		return &spawnChildWorkflowResult{
			ChildExternalId: triggerOpt.ExternalId,
			WasSkipped:      true,
		}, nil
	}

	err = d.triggerWriter.TriggerFromWorkflowNames(ctx, tenantId, []*v1.WorkflowNameTriggerOpts{triggerOpt})
	if err != nil {
		return nil, fmt.Errorf("failed to trigger child workflow: %w", err)
	}

	return &spawnChildWorkflowResult{
		ChildExternalId: triggerOpt.ExternalId,
		WasSkipped:      false,
	}, nil
}

func (d *DispatcherServiceImpl) handleRegisterCallback(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRegisterCallbackRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	tenantId := invocation.tenantId
	callbackKey := fmt.Sprintf("%s:%d", req.DurableTaskExternalId, req.NodeId)

	existingCallback, err := d.repo.DurableEvents().GetEventLogCallback(ctx, tenantId, task.ID, task.InsertedAt, callbackKey)

	if err == nil && existingCallback != nil && existingCallback.Callback.IsSatisfied {
		if ackErr := invocation.send(&contracts.DurableTaskResponse{
			Message: &contracts.DurableTaskResponse_RegisterCallbackAck{
				RegisterCallbackAck: &contracts.DurableTaskRegisterCallbackAckResponse{
					InvocationCount:       req.InvocationCount,
					DurableTaskExternalId: req.DurableTaskExternalId,
					NodeId:                req.NodeId,
				},
			},
		}); ackErr != nil {
			return ackErr
		}

		return invocation.send(&contracts.DurableTaskResponse{
			Message: &contracts.DurableTaskResponse_CallbackCompleted{
				CallbackCompleted: &contracts.DurableTaskCallbackCompletedResponse{
					InvocationCount:       req.InvocationCount,
					DurableTaskExternalId: req.DurableTaskExternalId,
					NodeId:                req.NodeId,
					Payload:               existingCallback.Result,
				},
			},
		})
	}

	entryWithData, err := d.repo.DurableEvents().GetEventLogEntry(ctx, tenantId, task.ID, task.InsertedAt, req.NodeId)
	if err != nil {
		return status.Errorf(codes.NotFound, "event log entry not found for node_id %d: %v", req.NodeId, err)
	}
	entry := entryWithData.Entry

	now := sqlchelpers.TimestamptzFromTime(time.Now().UTC())
	externalId := uuid.New()

	var callbackKind sqlcv1.V1DurableEventLogCallbackKind
	switch entry.Kind.V1DurableEventLogEntryKind {
	case sqlcv1.V1DurableEventLogEntryKindRUNTRIGGERED:
		callbackKind = sqlcv1.V1DurableEventLogCallbackKindRUNCOMPLETED
	default:
		callbackKind = sqlcv1.V1DurableEventLogCallbackKindWAITFORCOMPLETED
	}

	_, err = d.repo.DurableEvents().CreateEventLogCallbacks(ctx, []v1.CreateEventLogCallbackOpts{{
		DurableTaskId:         task.ID,
		DurableTaskInsertedAt: task.InsertedAt,
		InsertedAt:            now,
		Kind:                  callbackKind,
		Key:                   callbackKey,
		NodeId:                req.NodeId,
		IsSatisfied:           false,
		ExternalId:            externalId,
	}})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to create callback entry: %v", err)
	}

	err = invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_RegisterCallbackAck{
			RegisterCallbackAck: &contracts.DurableTaskRegisterCallbackAckResponse{
				InvocationCount:       req.InvocationCount,
				DurableTaskExternalId: req.DurableTaskExternalId,
				NodeId:                req.NodeId,
			},
		},
	})

	if err != nil {
		return err
	}

	signalKey := getDurableTaskSignalKey(req.DurableTaskExternalId, req.NodeId)
	childExternalId := entryWithData.Entry.TriggeredRunExternalID

	go d.pollForCompletion(invocation, task, signalKey, childExternalId, req, callbackKey)

	return nil
}

func (d *DispatcherServiceImpl) pollForCompletion(
	invocation *durableTaskInvocation,
	task *sqlcv1.FlattenExternalIdsRow,
	signalKey string,
	childExternalId *uuid.UUID,
	req *contracts.DurableTaskRegisterCallbackRequest,
	callbackKey string,
) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	pollCtx := context.Background()

	for {
		select {
		case <-ticker.C:
			signalEvents := []v1.TaskIdInsertedAtSignalKey{{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				SignalKey:  signalKey,
			}}

			completedEvents, err := d.repo.Tasks().ListSignalCompletedEvents(pollCtx, invocation.tenantId, signalEvents)
			if err != nil {
				d.l.Error().Err(err).Msg("failed to list signal completed events")
				continue
			}

			for _, event := range completedEvents {
				if event.EventKey.String == signalKey {
					d.sendCallbackCompleted(invocation, task, req, callbackKey, event.Payload, event.TenantID)
					return
				}
			}

			if childExternalId != nil {
				finalized, err := d.repo.Tasks().ListFinalizedWorkflowRuns(pollCtx, invocation.tenantId, []uuid.UUID{*childExternalId})
				if err != nil {
					d.l.Error().Err(err).Msg("failed to list finalized workflow runs")
					continue
				}

				if len(finalized) > 0 && finalized[0] != nil {
					result := finalized[0]

					var resultPayload []byte
					if len(result.OutputEvents) == 1 {
						resultPayload = result.OutputEvents[0].Output
					} else if len(result.OutputEvents) > 1 {
						outputs := make(map[string]json.RawMessage)
						for _, event := range result.OutputEvents {
							if event.Output != nil {
								outputs[event.StepReadableID] = event.Output
							}
						}
						resultPayload, _ = json.Marshal(outputs)
					}

					d.sendCallbackCompleted(invocation, task, req, callbackKey, resultPayload, invocation.tenantId)
					return
				}
			}
		case <-invocation.server.Context().Done():
			d.l.Debug().Msg("stream closed, stopping completion polling")
			return
		}
	}
}

func (d *DispatcherServiceImpl) sendCallbackCompleted(
	invocation *durableTaskInvocation,
	task *sqlcv1.FlattenExternalIdsRow,
	req *contracts.DurableTaskRegisterCallbackRequest,
	callbackKey string,
	payload []byte,
	tenantId uuid.UUID,
) {
	sendErr := invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_CallbackCompleted{
			CallbackCompleted: &contracts.DurableTaskCallbackCompletedResponse{
				InvocationCount:       req.InvocationCount,
				DurableTaskExternalId: req.DurableTaskExternalId,
				NodeId:                req.NodeId,
				Payload:               payload,
			},
		},
	})

	if sendErr != nil {
		d.l.Error().Err(sendErr).Msg("failed to send callback_completed")
	}

	pollCtx := context.Background()
	_, updateErr := d.repo.DurableEvents().UpdateEventLogCallbackSatisfied(
		pollCtx,
		tenantId,
		task.ID,
		task.InsertedAt,
		callbackKey,
		true,
		payload,
	)

	if updateErr != nil {
		d.l.Error().Err(updateErr).Msg("failed to update callback as satisfied")
	}
}

func (d *DispatcherServiceImpl) handleEvictInvocation(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskEvictInvocationRequest,
) error {
	// TODO: Clean up any state associated with this invocation

	return nil
}
