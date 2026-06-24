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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (d *DispatcherServiceImpl) RegisterDurableEvent(ctx context.Context, req *contracts.RegisterDurableEventRequest) (*contracts.RegisterDurableEventResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	d.analytics.Count(ctx, analytics.Worker, analytics.Register)
	taskId, err := uuid.Parse(req.TaskId)

	if err != nil {
		d.l.Error().Ctx(ctx).Msgf("task id %s is not a valid uuid", req.TaskId)
		return nil, status.Error(codes.InvalidArgument, "task id is not a valid uuid")
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

	if err != nil {
		return nil, err
	}

	createConditionOpts := make([]v1.CreateExternalSignalConditionOpt, 0)

	for _, condition := range req.Conditions.SleepConditions {
		orGroupId, parseErr := uuid.Parse(condition.Base.OrGroupId)

		if parseErr != nil {
			d.l.Error().Ctx(ctx).Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
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
		orGroupId, parseErr := uuid.Parse(condition.Base.OrGroupId)

		if parseErr != nil {
			d.l.Error().Ctx(ctx).Msgf("or group id %s is not a valid uuid", condition.Base.OrGroupId)
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
		SignalTaskExternalId: task.ExternalID,
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
	w.mu.RLock()
	defer w.mu.RUnlock()

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
	d.analytics.Count(server.Context(), analytics.Worker, analytics.Listen)

	acks := &durableEventAcks{
		acks: make(map[v1.TaskIdInsertedAtSignalKey]uuid.UUID),
	}

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	deregister := d.streamSessions.Register(cancel)
	defer deregister()

	wg := sync.WaitGroup{}
	sendMu := sync.Mutex{}
	iterMu := sync.Mutex{}

	sendEvent := func(e *v1.V1TaskEventWithPayload) error {
		// FIXME: check max size of msg
		// results := cleanResults(e.Results)

		// if results == nil {
		// 	s.l.Warn().Ctx(ctx).Msgf("results size for workflow run %s exceeds 3MB and cannot be reduced", e.WorkflowRunId)
		// 	e.Results = nil
		// }

		externalId := acks.getExternalId(e.TaskID, e.TaskInsertedAt, e.EventKey.String)

		if externalId == uuid.Nil {
			d.l.Warn().Ctx(ctx).Msgf("could not find external id for task %d, signal key %s", e.TaskID, e.EventKey.String)
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
			d.l.Error().Ctx(ctx).Err(err).Msgf("could not send durable event for task %s, key %s", externalId, e.EventKey.String)
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
			d.l.Warn().Ctx(ctx).Msg("could not acquire lock")
			return nil
		}

		defer iterMu.Unlock()

		signalEvents = signalEvents[:min(1000, len(signalEvents))]
		start := time.Now()

		dbEvents, err := d.repo.Tasks().ListSignalCompletedEvents(ctx, tenantId, signalEvents)

		if err != nil {
			d.l.Error().Ctx(ctx).Err(err).Msg("could not list signal completed events")
			return err
		}

		for _, dbEvent := range dbEvents {
			err := sendEvent(dbEvent)

			if err != nil {
				return err
			}
		}

		if time.Since(start) > 100*time.Millisecond {
			d.l.Warn().Ctx(ctx).Msgf("list durable events for %d signals took %s", len(signalEvents), time.Since(start))
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

				d.l.Error().Ctx(ctx).Err(err).Msg("could not receive message from client")
				return
			}

			taskId, err := uuid.Parse(req.TaskId)

			if err != nil {
				d.l.Warn().Ctx(ctx).Msgf("task id %s is not a valid uuid", req.TaskId)
				continue
			}

			// FIXME: buffer/batch this to make it more efficient
			task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

			if err != nil {
				d.l.Error().Ctx(ctx).Err(err).Msg("could not get task by external id")
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
					d.l.Error().Ctx(ctx).Err(err).Msg("could not iterate over workflow runs")
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

// durableTaskInvocation represents a single durable-task session. It is transport-agnostic:
// sendFn delivers a response to the client, whether that's a gRPC stream (server.Send) or a
// channel (operator). Everything downstream — handlers and the async response router — only
// needs send() and a slot in durableInvocations.
var errDurableTaskSessionClosed = fmt.Errorf("durable task session closed")

type durableTaskInvocation struct {
	sendFn   func(*contracts.DurableTaskResponse) error
	l        *zerolog.Logger
	sendMu   sync.Mutex
	tenantId uuid.UUID
	workerId uuid.UUID
	closed   bool // channel transport only; guarded by sendMu
}

func (s *durableTaskInvocation) send(resp *contracts.DurableTaskResponse) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if s.closed {
		return errDurableTaskSessionClosed
	}

	return s.sendFn(resp)
}

// processDurableTaskMessage performs the lazy task-id registration (so async responses route
// back to this invocation) and dispatches the request to the typed handlers. It is shared by
// the gRPC DurableTask loop and the channel-based RegisterDurableTask loop.
func (d *DispatcherServiceImpl) processDurableTaskMessage(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRequest,
	registerTask func(string),
) {
	switch msg := req.GetMessage().(type) {
	case *contracts.DurableTaskRequest_Memo:
		registerTask(msg.Memo.DurableTaskExternalId)
	case *contracts.DurableTaskRequest_TriggerRuns:
		registerTask(msg.TriggerRuns.DurableTaskExternalId)
	case *contracts.DurableTaskRequest_WaitFor:
		registerTask(msg.WaitFor.DurableTaskExternalId)
	}

	if err := d.handleDurableTaskRequest(ctx, invocation, req); err != nil {
		d.l.Error().Err(err).Msg("error handling durable task request")
	}
}

func (d *DispatcherServiceImpl) DurableTask(server contracts.V1Dispatcher_DurableTaskServer) error {
	tenant := server.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	ctx, cancel := context.WithCancel(server.Context())
	defer cancel()

	deregister := d.streamSessions.Register(cancel)
	defer deregister()

	invocation := &durableTaskInvocation{
		sendFn:   server.Send,
		tenantId: tenantId,
		l:        d.l,
	}

	registeredTasks := make(map[uuid.UUID]struct{})

	defer func() {
		for taskId := range registeredTasks {
			d.durableInvocations.Delete(taskId)
		}
	}()

	registerTask := func(externalIdStr string) {
		taskExtId, err := uuid.Parse(externalIdStr)
		if err != nil {
			return
		}

		if _, exists := registeredTasks[taskExtId]; !exists {
			d.durableInvocations.Store(taskExtId, invocation)
			registeredTasks[taskExtId] = struct{}{}
		}
	}

	type recvResult struct {
		req *contracts.DurableTaskRequest
		err error
	}

	msgCh := make(chan recvResult)

	// Recv runs in its own goroutine because it is only interrupted by the stream
	// ending, not by ctx; this lets the handler observe context cancellation (e.g.
	// server shutdown hanging up this stream) while a Recv is pending. The goroutine
	// does nothing but Recv and hand off, so all processing and the deferred cleanup
	// stay serialized on the handler goroutine. Once the handler returns, gRPC
	// cancels server.Context(), which unblocks the pending channel send (or the next
	// Recv) and lets the goroutine exit.
	go func() {
		for {
			req, err := server.Recv()

			select {
			case msgCh <- recvResult{req: req, err: err}:
			case <-ctx.Done():
				return
			}

			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case r := <-msgCh:
			if r.err != nil {
				if errors.Is(r.err, io.EOF) || status.Code(r.err) == codes.Canceled {
					return nil
				}

				d.l.Error().Err(r.err).Msg("error receiving durable task request")
				return r.err
			}

			d.processDurableTaskMessage(ctx, invocation, r.req, registerTask)
		}
	}
}

// RegisterDurableTask sets up a channel-backed durable-task session — the in-engine
// equivalent of the DurableTask gRPC stream, for operators that don't hold a gRPC stream.
// The caller writes DurableTaskRequests to the returned requestCh and reads
// DurableTaskResponses from respCh, reusing the same handlers and routing table
// (durableInvocations) as the gRPC path, so async responses (wait-for satisfied, evictions,
// acks) are delivered identically.
//
// externalId is registered up front so responses route immediately; additional task ids are
// registered lazily as messages reference them, matching DurableTask. The session is torn
// down (invocations deregistered, respCh closed) when ctx is cancelled or requestCh is
// closed.
func (d *DispatcherServiceImpl) RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *contracts.DurableTaskRequest, <-chan *contracts.DurableTaskResponse, error) {
	tenant, ok := ctx.Value("tenant").(*sqlcv1.Tenant)

	if !ok {
		return nil, nil, status.Error(codes.InvalidArgument, "tenant not found on context")
	}

	ctx, cancel := context.WithCancel(ctx)
	deregister := d.streamSessions.Register(cancel)

	// Buffer large enough to hold all tasks in a single DAG layer so taskEmitter can
	// queue all concurrent triggers without blocking on the goroutine's send-ack cycle.
	requestCh := make(chan *contracts.DurableTaskRequest, 64)
	respCh := make(chan *contracts.DurableTaskResponse)

	invocation := &durableTaskInvocation{
		tenantId: tenant.ID,
		l:        d.l,
		sendFn: func(resp *contracts.DurableTaskResponse) error {
			select {
			case respCh <- resp:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	registeredTasks := make(map[uuid.UUID]struct{})

	registerTask := func(externalIdStr string) {
		taskExtId, err := uuid.Parse(externalIdStr)
		if err != nil {
			return
		}

		if _, exists := registeredTasks[taskExtId]; !exists {
			d.durableInvocations.Store(taskExtId, invocation)
			registeredTasks[taskExtId] = struct{}{}
		}
	}

	// register the task up front so async responses route back to this invocation
	// immediately, before the caller sends its first message.
	d.durableInvocations.Store(externalId, invocation)
	registeredTasks[externalId] = struct{}{}

	go func() {
		defer deregister()
		defer cancel()

		// Teardown order matters: deregister from durableInvocations first so new async
		// routers can't find this invocation, then close respCh under sendMu so any in-flight
		// send (which holds sendMu and selects on ctx.Done()) has already returned. Marking
		// closed before close() makes later sends return an error instead of panicking on a
		// closed channel.
		defer func() {
			for taskId := range registeredTasks {
				d.durableInvocations.Delete(taskId)
			}

			invocation.sendMu.Lock()
			invocation.closed = true
			close(respCh)
			invocation.sendMu.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case req, ok := <-requestCh:
				if !ok {
					return
				}

				d.processDurableTaskMessage(ctx, invocation, req, registerTask)
			}
		}
	}()

	return requestCh, respCh, nil
}

// RegisterDurableTask delegates to the V1 dispatcher service so DispatcherImpl satisfies
// operator.TaskEventWriter, giving operators the in-engine equivalent of the DurableTask RPC.
func (d *DispatcherImpl) RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *contracts.DurableTaskRequest, <-chan *contracts.DurableTaskResponse, error) {
	return d.serviceV1.RegisterDurableTask(ctx, externalId)
}

func (d *DispatcherServiceImpl) handleDurableTaskRequest(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRequest,
) error {
	switch msg := req.GetMessage().(type) {
	case *contracts.DurableTaskRequest_RegisterWorker:
		return d.handleRegisterWorker(ctx, invocation, msg.RegisterWorker)
	case *contracts.DurableTaskRequest_Memo:
		return d.handleMemo(ctx, invocation, msg.Memo)
	case *contracts.DurableTaskRequest_TriggerRuns:
		return d.handleTriggerRuns(ctx, invocation, msg.TriggerRuns)
	case *contracts.DurableTaskRequest_WaitFor:
		return d.handleWaitFor(ctx, invocation, msg.WaitFor)
	case *contracts.DurableTaskRequest_EvictInvocation:
		return d.handleEvictInvocation(ctx, invocation, msg.EvictInvocation)
	case *contracts.DurableTaskRequest_WorkerStatus:
		return d.handleWorkerStatus(ctx, invocation, msg.WorkerStatus)
	case *contracts.DurableTaskRequest_CompleteMemo:
		return d.handleCompleteMemo(ctx, invocation, msg.CompleteMemo)
	default:
		return status.Errorf(codes.InvalidArgument, "unknown message type: %T", msg)
	}
}

func (d *DispatcherServiceImpl) handleRegisterWorker(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskRequestRegisterWorker,
) error {
	workerId, err := uuid.Parse(req.WorkerId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid worker id: %v", err)
	}

	d.analytics.Count(ctx, analytics.DurableTask, analytics.Register)

	invocation.workerId = workerId

	err = d.repo.Workers().UpdateWorkerDurableTaskDispatcherId(ctx, invocation.tenantId, workerId, d.dispatcherId)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to update worker durable task dispatcher id: %v", err)
	}

	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_RegisterWorker{
			RegisterWorker: &contracts.DurableTaskResponseRegisterWorker{
				WorkerId: req.WorkerId,
			},
		},
	})
}

func newEntryRef(taskExternalId string, invocationCount int32, nodeAndBranch v1.NodeIdBranchIdTuple) *contracts.DurableEventLogEntryRef {
	return &contracts.DurableEventLogEntryRef{
		DurableTaskExternalId: taskExternalId,
		InvocationCount:       invocationCount,
		BranchId:              nodeAndBranch.BranchId,
		NodeId:                nodeAndBranch.NodeId,
	}
}

func (d *DispatcherServiceImpl) sendNonDeterminismError(invocation *durableTaskInvocation, nde *v1.NonDeterminismError, invocationCount int32) error {
	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_Error{
			Error: &contracts.DurableTaskErrorResponse{
				Ref: &contracts.DurableEventLogEntryRef{
					DurableTaskExternalId: nde.TaskExternalId.String(),
					InvocationCount:       invocationCount,
					BranchId:              nde.BranchId,
					NodeId:                nde.NodeId,
				},
				ErrorType:    contracts.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
				ErrorMessage: nde.Error(),
			},
		},
	})
}

func (d *DispatcherServiceImpl) sendStaleInvocationEviction(invocation *durableTaskInvocation, sie *v1.StaleInvocationError) error {
	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_ServerEvict{
			ServerEvict: &contracts.DurableTaskServerEvictNotice{
				DurableTaskExternalId: sie.TaskExternalId.String(),
				InvocationCount:       sie.ActualInvocationCount,
				Reason:                sie.Error(),
			},
		},
	})
}

func (d *DispatcherServiceImpl) deliverSatisfiedEntries(taskExternalId string, result *v1.IngestDurableTaskEventResult) error {
	switch result.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		for _, entry := range result.TriggerRunsResult.Entries {
			if entry.IsSatisfied {
				taskExtId, _ := uuid.Parse(taskExternalId)
				if err := d.DeliverDurableEventLogEntryCompletion(
					taskExtId,
					result.TriggerRunsResult.InvocationCount,
					entry.BranchId,
					entry.NodeId,
					entry.ResultPayload,
					entry.ChildTaskIsFailure,
					entry.ChildTaskErrorMessage,
				); err != nil {
					return fmt.Errorf("failed to deliver callback completion for node %d: %w", entry.NodeId, err)
				}
			}
		}
	case sqlcv1.V1DurableEventLogKindMEMO:
		if result.MemoResult.IsSatisfied {
			taskExtId, _ := uuid.Parse(taskExternalId)
			if err := d.DeliverDurableEventLogEntryCompletion(
				taskExtId,
				result.MemoResult.InvocationCount,
				result.MemoResult.BranchId,
				result.MemoResult.NodeId,
				result.MemoResult.ResultPayload,
				false,
				nil,
			); err != nil {
				return fmt.Errorf("failed to deliver callback completion for node %d: %w", result.MemoResult.NodeId, err)
			}
		}
	case sqlcv1.V1DurableEventLogKindWAITFOR:
		if result.WaitForResult.IsSatisfied {
			taskExtId, _ := uuid.Parse(taskExternalId)
			if err := d.DeliverDurableEventLogEntryCompletion(
				taskExtId,
				result.WaitForResult.InvocationCount,
				result.WaitForResult.BranchId,
				result.WaitForResult.NodeId,
				result.WaitForResult.ResultPayload,
				false,
				nil,
			); err != nil {
				return fmt.Errorf("failed to deliver callback completion for node %d: %w", result.WaitForResult.NodeId, err)
			}
		}
	default:
		return fmt.Errorf("unknown durable event log kind: %s", result.Kind)
	}
	return nil
}

func (d *DispatcherServiceImpl) handleMemo(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskMemoRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	d.analytics.Count(ctx, analytics.DurableTask, analytics.Memo)

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	ingestionResult, err := d.repo.DurableEvents().IngestDurableTaskEvent(ctx, v1.IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &v1.BaseIngestEventOpts{
			TenantId:        invocation.tenantId,
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindMEMO,
			InvocationCount: req.InvocationCount,
		},
		Memo: &v1.IngestMemoOpts{
			Payload: req.Payload,
			MemoKey: req.Key,
		},
	})

	var nde *v1.NonDeterminismError
	var sie *v1.StaleInvocationError

	switch {
	case err != nil && errors.As(err, &nde):
		return d.sendNonDeterminismError(invocation, nde, req.InvocationCount)
	case err != nil && errors.As(err, &sie):
		return d.sendStaleInvocationEviction(invocation, sie)
	case err != nil:
		return status.Errorf(codes.Internal, "failed to ingest memo event: %v", err)
	}

	err = invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_MemoAck{
			MemoAck: &contracts.DurableTaskEventMemoAckResponse{
				Ref: newEntryRef(req.DurableTaskExternalId, req.InvocationCount, v1.NodeIdBranchIdTuple{
					NodeId:   ingestionResult.MemoResult.NodeId,
					BranchId: ingestionResult.MemoResult.BranchId,
				}),
				MemoAlreadyExisted: ingestionResult.MemoResult.AlreadyExisted,
				MemoResultPayload:  ingestionResult.MemoResult.ResultPayload,
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send memo ack: %v", err)
	}

	return d.deliverSatisfiedEntries(req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleTriggerRuns(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskTriggerRunsRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	for _, w := range req.TriggerOpts {
		d.analytics.Count(ctx, analytics.WorkflowRun, analytics.Create, analytics.Props(
			"parent_is_durable_task", w.ParentTaskRunExternalId != nil,
			"has_priority", w.Priority != nil,
			"is_child", w.ParentId != nil,
			"has_additional_meta", w.AdditionalMetadata != nil,
			"has_desired_worker_id", w.DesiredWorkerId != nil,
			"has_desired_worker_labels", len(w.DesiredWorkerLabels) > 0,
		))
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	triggerOpts := make([]*v1.WorkflowNameTriggerOpts, 0, len(req.TriggerOpts))
	for _, triggerReq := range req.TriggerOpts {
		triggerTaskData, triggerErr := d.repo.Triggers().NewTriggerTaskData(ctx, invocation.tenantId, triggerReq, task)
		if triggerErr != nil {
			return status.Errorf(codes.Internal, "failed to create trigger options: %v", triggerErr)
		}
		triggerOpts = append(triggerOpts, &v1.WorkflowNameTriggerOpts{
			TriggerTaskData: triggerTaskData,
		})
	}

	if populateErr := d.repo.Triggers().PopulateExternalIdsForWorkflow(ctx, invocation.tenantId, triggerOpts); populateErr != nil {
		return status.Errorf(codes.Internal, "failed to populate external ids for workflow: %v", populateErr)
	}

	ingestionResult, err := d.repo.DurableEvents().IngestDurableTaskEvent(ctx, v1.IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &v1.BaseIngestEventOpts{
			TenantId:        invocation.tenantId,
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindRUN,
			InvocationCount: req.InvocationCount,
		},
		TriggerRuns: &v1.IngestTriggerRunsOpts{
			TriggerOpts: triggerOpts,
		},
	})

	var nde *v1.NonDeterminismError
	var sie *v1.StaleInvocationError

	switch {
	case err != nil && errors.As(err, &nde):
		return d.sendNonDeterminismError(invocation, nde, req.InvocationCount)
	case err != nil && errors.As(err, &sie):
		return d.sendStaleInvocationEviction(invocation, sie)
	case err != nil:
		return status.Errorf(codes.Internal, "failed to ingest trigger runs event: %v", err)
	}

	ackResp := &contracts.DurableTaskEventTriggerRunsAckResponse{
		DurableTaskExternalId: req.DurableTaskExternalId,
		InvocationCount:       req.InvocationCount,
	}

	for _, entry := range ingestionResult.TriggerRunsResult.Entries {
		ackResp.RunEntries = append(ackResp.RunEntries, &contracts.DurableTaskRunAckEntry{
			NodeId:                entry.NodeId,
			BranchId:              entry.BranchId,
			WorkflowRunExternalId: entry.WorkflowRunExternalId.String(),
		})
	}

	dags := ingestionResult.TriggerRunsResult.CreatedDAGs
	tasks := ingestionResult.TriggerRunsResult.CreatedTasks

	if len(dags) > 0 || len(tasks) > 0 {
		if sigErr := d.triggerWriter.SignalCreated(ctx, invocation.tenantId, tasks, dags); sigErr != nil {
			d.l.Error().Err(sigErr).Msg("failed to signal created tasks/DAGs for durable run trigger")
		}
	}

	err = invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_TriggerRunsAck{
			TriggerRunsAck: ackResp,
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send trigger runs ack: %v", err)
	}

	return d.deliverSatisfiedEntries(req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleWaitFor(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskWaitForRequest,
) error {
	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	var hasSleep, hasUserEvent bool
	if req.WaitForConditions != nil {
		hasSleep = len(req.WaitForConditions.SleepConditions) > 0
		hasUserEvent = len(req.WaitForConditions.UserEventConditions) > 0
	}
	d.analytics.Count(ctx, analytics.DurableTask, analytics.WaitFor, analytics.Props(
		"has_sleep", hasSleep,
		"has_user_event", hasUserEvent,
	))

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return status.Errorf(codes.NotFound, "task not found: %v", err)
	}

	var createConditionOpts []v1.CreateExternalSignalConditionOpt

	if req.WaitForConditions != nil {
		for _, condition := range req.WaitForConditions.SleepConditions {
			orGroupId, parseErr := uuid.Parse(condition.Base.OrGroupId)
			if parseErr != nil {
				return status.Errorf(codes.InvalidArgument, "or group id is not a valid uuid: %v", parseErr)
			}
			createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
				Kind:            v1.CreateExternalSignalConditionKindSLEEP,
				ReadableDataKey: condition.Base.ReadableDataKey,
				OrGroupId:       orGroupId,
				SleepFor:        &condition.SleepFor,
			})
		}

		for _, condition := range req.WaitForConditions.UserEventConditions {
			orGroupId, parseErr := uuid.Parse(condition.Base.OrGroupId)
			if parseErr != nil {
				return status.Errorf(codes.InvalidArgument, "or group id is not a valid uuid: %v", parseErr)
			}

			var considerEventsSince *time.Time
			if condition.ConsiderEventsSince != nil {
				ces := condition.ConsiderEventsSince.AsTime()
				considerEventsSince = &ces
			}

			createConditionOpts = append(createConditionOpts, v1.CreateExternalSignalConditionOpt{
				Kind:                         v1.CreateExternalSignalConditionKindUSEREVENT,
				ReadableDataKey:              condition.Base.ReadableDataKey,
				OrGroupId:                    orGroupId,
				UserEventKey:                 &condition.UserEventKey,
				UserEventScope:               condition.EventScope,
				UserEventConsiderEventsSince: considerEventsSince,
				Expression:                   condition.Base.Expression,
			})
		}
	}

	var waitForLabel *string
	if label := req.GetLabel(); label != "" {
		waitForLabel = &label
	}

	ingestionResult, err := d.repo.DurableEvents().IngestDurableTaskEvent(ctx, v1.IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &v1.BaseIngestEventOpts{
			TenantId:        invocation.tenantId,
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindWAITFOR,
			InvocationCount: req.InvocationCount,
		},
		WaitFor: &v1.IngestWaitForOpts{
			WaitForConditions: createConditionOpts,
			Label:             waitForLabel,
		},
	})

	var nde *v1.NonDeterminismError
	var sie *v1.StaleInvocationError

	switch {
	case err != nil && errors.As(err, &nde):
		return d.sendNonDeterminismError(invocation, nde, req.InvocationCount)
	case err != nil && errors.As(err, &sie):
		return d.sendStaleInvocationEviction(invocation, sie)
	case err != nil:
		return status.Errorf(codes.Internal, "failed to ingest wait_for event: %v", err)
	}

	err = invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_WaitForAck{
			WaitForAck: &contracts.DurableTaskEventWaitForAckResponse{
				Ref: newEntryRef(req.DurableTaskExternalId, req.InvocationCount, v1.NodeIdBranchIdTuple{
					NodeId:   ingestionResult.WaitForResult.NodeId,
					BranchId: ingestionResult.WaitForResult.BranchId,
				}),
			},
		},
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to send wait_for ack: %v", err)
	}

	return d.deliverSatisfiedEntries(req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleCompleteMemo(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskCompleteMemoRequest,
) error {
	if req.Ref == nil {
		return status.Errorf(codes.InvalidArgument, "ref is required")
	}

	taskExternalId, err := uuid.Parse(req.Ref.DurableTaskExternalId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid durable_task_external_id: %v", err)
	}

	d.analytics.Count(ctx, analytics.DurableTask, analytics.Memo)

	err = d.repo.DurableEvents().CompleteMemoEntry(ctx, v1.CompleteMemoEntryOpts{
		TenantId:        invocation.tenantId,
		TaskExternalId:  taskExternalId,
		InvocationCount: req.Ref.InvocationCount,
		BranchId:        req.Ref.BranchId,
		NodeId:          req.Ref.NodeId,
		MemoKey:         req.MemoKey,
		Payload:         req.Payload,
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to complete memo entry: %v", err)
	}

	return nil
}

func (d *DispatcherServiceImpl) sendEvictionError(invocation *durableTaskInvocation, req *contracts.DurableTaskEvictInvocationRequest, errMsg string) error {
	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_Error{
			Error: &contracts.DurableTaskErrorResponse{
				Ref: &contracts.DurableEventLogEntryRef{
					DurableTaskExternalId: req.DurableTaskExternalId,
					InvocationCount:       req.InvocationCount,
				},
				ErrorType:    contracts.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_UNSPECIFIED,
				ErrorMessage: errMsg,
			},
		},
	})
}

func (d *DispatcherServiceImpl) handleEvictInvocation(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskEvictInvocationRequest,
) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	taskExternalId, err := uuid.Parse(req.DurableTaskExternalId)
	if err != nil {
		return d.sendEvictionError(invocation, req, fmt.Sprintf("invalid durable_task_external_id: %v", err))
	}

	d.analytics.Count(ctx, analytics.DurableTask, analytics.Evict)

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, invocation.tenantId, taskExternalId, false)
	if err != nil {
		return d.sendEvictionError(invocation, req, fmt.Sprintf("task not found: %v", err))
	}

	wasEvicted, err := d.repo.Tasks().EvictTask(ctx, invocation.tenantId, v1.TaskIdInsertedAtRetryCount{
		Id:         task.ID,
		InsertedAt: task.InsertedAt,
		RetryCount: task.RetryCount,
	})
	if err != nil {
		return d.sendEvictionError(invocation, req, fmt.Sprintf("failed to evict task: %v", err))
	}

	if wasEvicted {
		msg, err := tasktypes.MonitoringEventMessageFromInternal(
			invocation.tenantId,
			tasktypes.CreateMonitoringEventPayload{
				TaskId:                 task.ID,
				RetryCount:             task.RetryCount,
				DurableInvocationCount: req.InvocationCount,
				EventTimestamp:         time.Now(),
				EventType:              sqlcv1.V1EventTypeOlapDURABLEEVICTED,
				EventMessage:           durableEvictionMessage(req),
			},
		)
		if err != nil {
			d.l.Warn().Err(err).Msg("failed to build DURABLE_EVICTED monitoring message")
		} else if err := d.pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false); err != nil {
			d.l.Warn().Err(err).Msg("failed to publish DURABLE_EVICTED to OLAP")
		}
	} else {
		d.l.Debug().Str("task_external_id", req.DurableTaskExternalId).Msg("eviction skipped, task likely already timed out")
	}

	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EvictionAck{
			EvictionAck: &contracts.DurableTaskEvictionAckResponse{
				InvocationCount:       req.InvocationCount,
				DurableTaskExternalId: req.DurableTaskExternalId,
			},
		},
	})
}

func durableEvictionMessage(req *contracts.DurableTaskEvictInvocationRequest) string {
	if reason := req.GetReason(); reason != "" {
		return reason
	}
	return "Task paused and evicted from worker"
}

func (d *DispatcherServiceImpl) handleWorkerStatus(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskWorkerStatusRequest,
) error {
	if len(req.WaitingEntries) == 0 {
		return nil
	}

	uniqueExternalIds := make(map[uuid.UUID]int32)
	waiting := make([]v1.TaskExternalIdNodeIdBranchId, 0, len(req.WaitingEntries))

	for _, cb := range req.WaitingEntries {
		taskExternalId, err := uuid.Parse(cb.DurableTaskExternalId)
		if err != nil {
			d.l.Warn().Err(err).Msgf("invalid durable_task_external_id in worker_status: %s", cb.DurableTaskExternalId)
			continue
		}

		uniqueExternalIds[taskExternalId] = cb.InvocationCount

		waiting = append(waiting, v1.TaskExternalIdNodeIdBranchId{
			TaskExternalId: taskExternalId,
			NodeId:         cb.NodeId,
			BranchId:       cb.BranchId,
		})
	}

	if len(waiting) == 0 {
		return nil
	}

	if len(uniqueExternalIds) > 0 {
		externalIds := make([]uuid.UUID, 0, len(uniqueExternalIds))
		for extId := range uniqueExternalIds {
			externalIds = append(externalIds, extId)
		}

		tasks, err := d.repo.Tasks().FlattenExternalIds(ctx, invocation.tenantId, externalIds)
		if err != nil {
			return fmt.Errorf("failed to look up tasks for invocation count check in worker_status: %w", err)
		}
		if len(tasks) > 0 {
			idInsertedAts := make([]v1.IdInsertedAt, 0, len(tasks))
			taskIdToExternalId := make(map[v1.IdInsertedAt]uuid.UUID, len(tasks))

			for _, t := range tasks {
				key := v1.IdInsertedAt{ID: t.ID, InsertedAt: t.InsertedAt}
				idInsertedAts = append(idInsertedAts, key)
				taskIdToExternalId[key] = t.ExternalID
			}

			idInsertedAtToInvocationCount, err := d.repo.DurableEvents().GetDurableTaskInvocationCounts(ctx, invocation.tenantId, idInsertedAts)
			if err != nil {
				return fmt.Errorf("failed to get invocation counts in worker_status: %w", err)
			}
			for key, currentCount := range idInsertedAtToInvocationCount {
				extId, ok := taskIdToExternalId[key]
				if !ok || currentCount == nil {
					continue
				}
				workerInvocationCount, has := uniqueExternalIds[extId]
				if !has {
					continue
				}
				if workerInvocationCount < *currentCount {
					err = invocation.send(&contracts.DurableTaskResponse{
						Message: &contracts.DurableTaskResponse_ServerEvict{
							ServerEvict: &contracts.DurableTaskServerEvictNotice{
								DurableTaskExternalId: extId.String(),
								InvocationCount:       workerInvocationCount,
								Reason:                fmt.Sprintf("stale invocation: server has %d, worker sent %d", *currentCount, workerInvocationCount),
							},
						},
					})
					if err != nil {
						d.l.Error().Err(err).Msgf("failed to send server eviction notification for task %s", extId.String())
					}
				}
			}
		}
	}

	callbacks, err := d.repo.DurableEvents().GetSatisfiedDurableEvents(ctx, invocation.tenantId, waiting)
	if err != nil {
		return fmt.Errorf("failed to get satisfied callbacks: %w", err)
	}

	for _, cb := range callbacks {
		if err := d.deliverEntryCompleted(invocation, cb); err != nil {
			d.l.Error().Err(err).Msgf("failed to send event_log_entry for task %s node %d", cb.TaskExternalId, cb.NodeID)
		}
	}

	return nil
}

func (d *DispatcherServiceImpl) deliverEntryCompleted(invocation *durableTaskInvocation, cb *v1.SatisfiedEventWithPayload) error {
	ref := &contracts.DurableEventLogEntryRef{
		DurableTaskExternalId: cb.TaskExternalId.String(),
		InvocationCount:       cb.InvocationCount,
		BranchId:              cb.BranchID,
		NodeId:                cb.NodeID,
	}
	resp := &contracts.DurableTaskEventLogEntryCompletedResponse{
		Ref:     ref,
		Payload: cb.Result,
	}
	if cb.ChildTaskIsFailure {
		resp.Payload = nil
		resp.IsFailure = true
		resp.ErrorMessage = cb.ChildTaskErrorMessage
	}
	return invocation.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: resp,
		},
	})
}

func (d *DispatcherServiceImpl) DeliverDurableEventLogEntryCompletion(taskExternalId uuid.UUID, invocationCount int32, branchId, nodeId int64, payload []byte, isFailure bool, errorMessage *string) error {
	inv, ok := d.durableInvocations.Load(taskExternalId)
	if !ok {
		return fmt.Errorf("no active invocation found for task %s", taskExternalId)
	}

	ref := &contracts.DurableEventLogEntryRef{
		DurableTaskExternalId: taskExternalId.String(),
		InvocationCount:       invocationCount,
		BranchId:              branchId,
		NodeId:                nodeId,
	}
	resp := &contracts.DurableTaskEventLogEntryCompletedResponse{
		Ref:     ref,
		Payload: payload,
	}
	if isFailure {
		resp.Payload = nil
		resp.IsFailure = true
		resp.ErrorMessage = errorMessage
	}
	return inv.send(&contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: resp,
		},
	})
}

func (d *DispatcherServiceImpl) TriggerDAGStep(ctx context.Context, tenantId uuid.UUID, req *operator.DAGStepTriggerRequest) (*operator.DAGStepTriggerResult, error) {
	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, req.ParentTaskExternalId, false)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	childIndex := int64(req.ChildIndex)
	stepLabel := req.ActionId
	if parts := strings.SplitN(req.ActionId, ":", 2); len(parts) == 2 {
		stepLabel = parts[1]
	}
	triggerOpts := []*v1.WorkflowNameTriggerOpts{{
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName:            req.WorkflowName,
			TargetActionId:          &req.ActionId,
			UserMessage:             &stepLabel,
			Data:                    []byte(req.Input),
			ParentExternalId:        &task.ExternalID,
			ParentTaskId:            &task.ID,
			ParentTaskInsertedAt:    &task.InsertedAt.Time,
			ChildIndex:              &childIndex,
			DagParentWorkflowRunIds: req.DagParentRunIds,
		},
	}}

	if err := d.repo.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, triggerOpts); err != nil {
		return nil, fmt.Errorf("failed to populate external ids: %w", err)
	}

	ingestionResult, err := d.repo.DurableEvents().IngestDurableTaskEvent(ctx, v1.IngestDurableTaskEventOpts{
		BaseIngestEventOpts: &v1.BaseIngestEventOpts{
			TenantId:        tenantId,
			Task:            task,
			Kind:            sqlcv1.V1DurableEventLogKindRUN,
			InvocationCount: req.InvocationCount,
		},
		TriggerRuns: &v1.IngestTriggerRunsOpts{
			TriggerOpts: triggerOpts,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ingest durable task event: %w", err)
	}

	dags := ingestionResult.TriggerRunsResult.CreatedDAGs
	tasks := ingestionResult.TriggerRunsResult.CreatedTasks

	if len(dags) > 0 || len(tasks) > 0 {
		if sigErr := d.triggerWriter.SignalCreated(ctx, tenantId, tasks, dags); sigErr != nil {
			d.l.Error().Err(sigErr).Msg("failed to signal created tasks/dags for dag step trigger")
		}
	}

	if len(ingestionResult.TriggerRunsResult.Entries) == 0 {
		return nil, fmt.Errorf("no entries returned from durable event ingestion")
	}

	entry := ingestionResult.TriggerRunsResult.Entries[0]

	return &operator.DAGStepTriggerResult{
		NodeId:                entry.NodeId,
		BranchId:              entry.BranchId,
		WorkflowRunExternalId: entry.WorkflowRunExternalId.String(),
	}, nil
}
