package dispatcher

import (
	"context"
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

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (d *DispatcherImpl) RegisterDurableEvent(ctx context.Context, req *v1contracts.RegisterDurableEventRequest) (*v1contracts.RegisterDurableEventResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	taskId, err := uuid.Parse(req.TaskId)

	if err != nil {
		d.l.Error().Msgf("task id %s is not a valid uuid", req.TaskId)
		return nil, status.Error(codes.InvalidArgument, "task id is not a valid uuid")
	}

	task, err := d.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

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

	err = d.repov1.Matches().RegisterSignalMatchConditions(ctx, tenantId, createMatchOpts)

	if err != nil {
		return nil, err
	}

	return &v1contracts.RegisterDurableEventResponse{}, nil
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

func (d *DispatcherImpl) ListenForDurableEvent(server v1contracts.Dispatcher_ListenForDurableEventServer) error {
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
		externalId := acks.getExternalId(e.TaskID, e.TaskInsertedAt, e.EventKey.String)

		if externalId == uuid.Nil {
			d.l.Warn().Msgf("could not find external id for task %d, signal key %s", e.TaskID, e.EventKey.String)
			return fmt.Errorf("could not find external id for task %d, signal key %s", e.TaskID, e.EventKey.String)
		}

		sendMu.Lock()
		err := server.Send(&v1contracts.DurableEvent{
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

		dbEvents, err := d.repov1.Tasks().ListSignalCompletedEvents(ctx, tenantId, signalEvents)

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
			task, err := d.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

			if err != nil {
				d.l.Error().Err(err).Msg("could not get task by external id")
				continue
			}

			acks.addEvent(taskId, task.ID, task.InsertedAt, req.SignalKey)
		}
	}()

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

	waitFor(&wg, 60*time.Second, d.l)

	return nil
}

type durableTaskInvocation struct {
	server   v1contracts.Dispatcher_DurableTaskServer
	tenantId uuid.UUID
	workerId uuid.UUID
	l        *zerolog.Logger

	sendMu sync.Mutex
}

func (s *durableTaskInvocation) send(resp *v1contracts.DurableTaskResponse) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.server.Send(resp)
}

func (d *DispatcherImpl) DurableTask(server v1contracts.Dispatcher_DurableTaskServer) error {
	tenant := server.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	invocation := &durableTaskInvocation{
		server:   server,
		tenantId: tenantId,
		l:        d.l,
	}

	registeredTasks := make(map[uuid.UUID]struct{})
	defer func() {
		for taskId := range registeredTasks {
			d.durableInvocations.Delete(taskId)
		}
		d.workerInvocations.Delete(invocation.workerId)
	}()

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

		if event := req.GetEvent(); event != nil {
			taskExtId, err := uuid.Parse(event.DurableTaskExternalId)

			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid durable task external id: %v", err)
			}

			if _, exists := registeredTasks[taskExtId]; !exists {
				d.durableInvocations.Store(taskExtId, invocation)
				registeredTasks[taskExtId] = struct{}{}
			}
		}

		if err := d.handleDurableTaskRequest(ctx, invocation, req); err != nil {
			d.l.Error().Err(err).Msg("error handling durable task request")
		}
	}
}

func (d *DispatcherImpl) handleDurableTaskRequest(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *v1contracts.DurableTaskRequest,
) error {
	switch msg := req.GetMessage().(type) {
	case *v1contracts.DurableTaskRequest_RegisterWorker:
		return d.handleRegisterWorker(ctx, invocation, msg.RegisterWorker)
	case *v1contracts.DurableTaskRequest_Event:
		return d.handleDurableTaskEvent(ctx, invocation, msg.Event)
	case *v1contracts.DurableTaskRequest_EvictInvocation:
		return d.handleEvictInvocation(ctx, invocation, msg.EvictInvocation)
	case *v1contracts.DurableTaskRequest_WorkerStatus:
		return d.handleWorkerStatus(ctx, invocation, msg.WorkerStatus)
	default:
		return status.Errorf(codes.InvalidArgument, "unknown message type: %T", msg)
	}
}

func (d *DispatcherImpl) handleRegisterWorker(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *v1contracts.DurableTaskRequestRegisterWorker,
) error {
	workerId, err := uuid.Parse(req.WorkerId)

	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid worker id: %v", err)
	}

	invocation.workerId = workerId
	d.workerInvocations.Store(workerId, invocation)

	err = d.repov1.Workers().UpdateWorkerDurableTaskDispatcherId(ctx, invocation.tenantId, workerId, d.dispatcherId)

	if err != nil {
		return status.Errorf(codes.Internal, "failed to update worker durable task dispatcher id: %v", err)
	}

	return invocation.send(&v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &v1contracts.DurableTaskResponseRegisterWorker{
				WorkerId: req.WorkerId,
			},
		},
	})
}

func getDurableTaskEventKind(eventKind v1contracts.DurableTaskEventKind) (sqlcv1.V1DurableEventLogKind, error) {
	switch eventKind {
	case v1contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_WAIT_FOR:
		return sqlcv1.V1DurableEventLogKindWAITFOR, nil
	case v1contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_RUN:
		return sqlcv1.V1DurableEventLogKindRUN, nil
	case v1contracts.DurableTaskEventKind_DURABLE_TASK_TRIGGER_KIND_MEMO:
		return sqlcv1.V1DurableEventLogKindMEMO, nil
	default:
		return "", fmt.Errorf("unsupported event kind: %v", eventKind)
	}
}

func (d *DispatcherImpl) handleDurableTaskEvent(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *v1contracts.DurableTaskEventRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	task, err := d.repov1.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	kind, err := getDurableTaskEventKind(req.Kind)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid event kind: %v", err)
	}

	var triggerOpts *v1.WorkflowNameTriggerOpts

	if kind == sqlcv1.V1DurableEventLogKindRUN {
		ttd, err := d.repov1.Triggers().NewTriggerTaskData(ctx, invocation.tenantId, req.TriggerOpts, task)

		if err != nil {
			return status.Errorf(codes.Internal, "failed to create trigger options: %v", err)
		}

		optsSlice := []*v1.WorkflowNameTriggerOpts{{
			TriggerTaskData: ttd,
		}}

		err = d.repov1.Triggers().PopulateExternalIdsForWorkflow(ctx, invocation.tenantId, optsSlice)

		if err != nil {
			return status.Errorf(codes.Internal, "failed to populate external ids for workflow: %v", err)
		}

		triggerOpts = optsSlice[0]
	}

	createConditionOpts := make([]v1.CreateExternalSignalConditionOpt, 0)

	if req.WaitForConditions != nil {
		for _, condition := range req.WaitForConditions.SleepConditions {
			orGroupId, err := uuid.Parse(condition.Base.OrGroupId)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "or group id is not a valid uuid: %v", err)
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
				return status.Errorf(codes.InvalidArgument, "or group id is not a valid uuid: %v", err)
			}

			createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
				Kind:            v1.CreateExternalSignalConditionKindUSEREVENT,
				ReadableDataKey: condition.Base.ReadableDataKey,
				OrGroupId:       orGroupId,
				UserEventKey:    &condition.UserEventKey,
				Expression:      condition.Base.Expression,
			})
		}
	}

	ingestionResult, err := d.repov1.DurableEvents().IngestDurableTaskEvent(ctx, v1.IngestDurableTaskEventOpts{
		TenantId:          invocation.tenantId,
		Task:              task,
		Kind:              kind,
		Payload:           req.Payload,
		WaitForConditions: createConditionOpts,
		InvocationCount:   req.InvocationCount,
		TriggerOpts:       triggerOpts,
	})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to ingest durable task event: %v", err)
	}

	if len(ingestionResult.CreatedTasks) > 0 || len(ingestionResult.CreatedDAGs) > 0 {
		if sigErr := d.triggerWriter.SignalCreated(ctx, invocation.tenantId, ingestionResult.CreatedTasks, ingestionResult.CreatedDAGs); sigErr != nil {
			d.l.Error().Err(sigErr).Msg("failed to signal created tasks/DAGs for durable run trigger")
		}
	}

	err = invocation.send(&v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_TriggerAck{
			TriggerAck: &v1contracts.DurableTaskEventAckResponse{
				InvocationCount:       req.InvocationCount,
				DurableTaskExternalId: req.DurableTaskExternalId,
				NodeId:                ingestionResult.NodeId,
			},
		},
	})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to send trigger ack: %v", err)
	}

	if ingestionResult.Callback.Callback.IsSatisfied {
		err := d.DeliverCallbackCompletion(
			taskExternalId,
			ingestionResult.NodeId,
			ingestionResult.Callback.Result,
		)

		if err != nil {
			d.l.Error().Err(err).Msgf("failed to deliver callback completion for task %s node %d", taskExternalId, ingestionResult.NodeId)
			return status.Errorf(codes.Internal, "failed to deliver callback completion: %v", err)
		}
	}

	return nil
}

func (d *DispatcherImpl) handleEvictInvocation(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *v1contracts.DurableTaskEvictInvocationRequest,
) error {
	return nil
}

func (d *DispatcherImpl) handleWorkerStatus(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *v1contracts.DurableTaskWorkerStatusRequest,
) error {
	if len(req.WaitingCallbacks) == 0 {
		return nil
	}

	waiting := make([]v1.TaskExternalIdNodeId, 0, len(req.WaitingCallbacks))
	for _, cb := range req.WaitingCallbacks {
		taskExternalId, err := uuid.Parse(cb.DurableTaskExternalId)
		if err != nil {
			d.l.Warn().Err(err).Msgf("invalid durable_task_external_id in worker_status: %s", cb.DurableTaskExternalId)
			continue
		}
		waiting = append(waiting, v1.TaskExternalIdNodeId{
			TaskExternalId: taskExternalId,
			NodeId:         cb.NodeId,
		})
	}

	if len(waiting) == 0 {
		return nil
	}

	callbacks, err := d.repov1.DurableEvents().GetSatisfiedCallbacks(ctx, invocation.tenantId, waiting)
	if err != nil {
		return fmt.Errorf("failed to get satisfied callbacks: %w", err)
	}

	for _, cb := range callbacks {
		if err := invocation.send(&v1contracts.DurableTaskResponse{
			Message: &v1contracts.DurableTaskResponse_CallbackCompleted{
				CallbackCompleted: &v1contracts.DurableTaskCallbackCompletedResponse{
					DurableTaskExternalId: cb.TaskExternalId.String(),
					NodeId:                cb.NodeID,
					Payload:               cb.Result,
				},
			},
		}); err != nil {
			d.l.Error().Err(err).Msgf("failed to send callback_completed for task %s node %d", cb.TaskExternalId, cb.NodeID)
		}
	}

	return nil
}

func (d *DispatcherImpl) DeliverCallbackCompletion(taskExternalId uuid.UUID, nodeId int64, payload []byte) error {
	inv, ok := d.durableInvocations.Load(taskExternalId)
	if !ok {
		return fmt.Errorf("no active invocation found for task %s", taskExternalId)
	}

	return inv.send(&v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_CallbackCompleted{
			CallbackCompleted: &v1contracts.DurableTaskCallbackCompletedResponse{
				DurableTaskExternalId: taskExternalId.String(),
				NodeId:                nodeId,
				Payload:               payload,
			},
		},
	})
}
