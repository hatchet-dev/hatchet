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
	"github.com/jackc/pgx/v5"
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
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (d *DispatcherServiceImpl) RegisterDurableEvent(ctx context.Context, req *contracts.RegisterDurableEventRequest) (*contracts.RegisterDurableEventResponse, error) {
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.register-durable-event")
	defer span.End()

	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant_id", Value: tenantId})
	d.analytics.Count(ctx, analytics.Worker, analytics.Register)
	taskId, err := uuid.Parse(req.TaskId)

	if err != nil {
		d.l.Error().Ctx(ctx).Msgf("task id %s is not a valid uuid", req.TaskId)
		return nil, status.Error(codes.InvalidArgument, "task id is not a valid uuid")
	}

	task, err := d.repo.Tasks().GetTaskByExternalId(ctx, tenantId, taskId, false)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "task run not found: %s", taskId)
		}

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
	closed   bool // channel transport only; guarded by sendMu

	releasesMu sync.Mutex
	releases   map[orderedReleaseKey]*orderedRelease
}

type orderedReleaseKey struct {
	taskExternalId  uuid.UUID
	invocationCount int32
}

type orderedRelease struct {
	mu                           sync.Mutex
	maxSatisfiedOrderSentAlready int64
	bufferedCompletions          map[int64]*contracts.DurableTaskResponse
	oldestBufferedAt             time.Time
	lastActivityAt               time.Time
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
	if msg, isRegisterWorker := req.GetMessage().(*contracts.DurableTaskRequest_RegisterWorker); isRegisterWorker {
		if err := d.handleRegisterWorker(ctx, invocation, msg.RegisterWorker); err != nil {
			d.l.Error().Err(err).Msg("error handling durable task request")
		}
		return
	}

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

func (s *durableTaskInvocation) getRelease(key orderedReleaseKey) *orderedRelease {
	if s.releases == nil {
		s.releases = make(map[orderedReleaseKey]*orderedRelease)
	}

	if rel, ok := s.releases[key]; ok {
		return rel
	}

	for existing := range s.releases {
		if existing.taskExternalId == key.taskExternalId && existing.invocationCount < key.invocationCount {
			delete(s.releases, existing)
		}
	}

	rel := &orderedRelease{
		bufferedCompletions: make(map[int64]*contracts.DurableTaskResponse),
		lastActivityAt:      time.Now(),
	}
	s.releases[key] = rel
	return rel
}

func (s *durableTaskInvocation) clearRelease(key orderedReleaseKey) {
	s.releasesMu.Lock()
	defer s.releasesMu.Unlock()

	delete(s.releases, key)
}

func (s *durableTaskInvocation) pruneIdleReleases(idle time.Duration) {
	s.releasesMu.Lock()
	defer s.releasesMu.Unlock()

	now := time.Now()

	for key, rel := range s.releases {
		rel.mu.Lock()
		idleEnough := len(rel.bufferedCompletions) == 0 && now.Sub(rel.lastActivityAt) > idle
		rel.mu.Unlock()

		if idleEnough {
			delete(s.releases, key)
		}
	}
}

func (s *durableTaskInvocation) deliverOrdered(taskExternalId uuid.UUID, invocationCount int32, satisfiedOrder *int64, resp *contracts.DurableTaskResponse) error {
	if satisfiedOrder == nil {
		return s.send(resp)
	}

	order := *satisfiedOrder

	s.releasesMu.Lock()
	rel := s.getRelease(orderedReleaseKey{taskExternalId: taskExternalId, invocationCount: invocationCount})
	s.releasesMu.Unlock()

	rel.mu.Lock()
	defer rel.mu.Unlock()

	rel.lastActivityAt = time.Now()

	var toSend []*contracts.DurableTaskResponse

	switch {
	case order <= rel.maxSatisfiedOrderSentAlready:
		// already released (e.g. reconnect / worker-status re-delivery); the worker
		// dedupes by node id, so send it through again.
		toSend = []*contracts.DurableTaskResponse{resp}
	case order == rel.maxSatisfiedOrderSentAlready+1:
		toSend = append(toSend, resp)
		rel.maxSatisfiedOrderSentAlready++

		for {
			next, ok := rel.bufferedCompletions[rel.maxSatisfiedOrderSentAlready+1]
			if !ok {
				break
			}
			delete(rel.bufferedCompletions, rel.maxSatisfiedOrderSentAlready+1)
			toSend = append(toSend, next)
			rel.maxSatisfiedOrderSentAlready++
		}

		if len(rel.bufferedCompletions) == 0 {
			rel.oldestBufferedAt = time.Time{}
		}
	default:
		if buffered, exists := rel.bufferedCompletions[order]; exists {
			existingRef := buffered.GetEntryCompleted().GetRef()
			incomingRef := resp.GetEntryCompleted().GetRef()
			if existingRef.GetNodeId() != incomingRef.GetNodeId() || existingRef.GetBranchId() != incomingRef.GetBranchId() {
				s.l.Error().Msgf(
					"durable task %s (invocation %d): satisfied_order %d claimed by two different entries (buffered node %d/branch %d vs incoming node %d/branch %d); dropping the newer one",
					taskExternalId, invocationCount, order,
					existingRef.GetNodeId(), existingRef.GetBranchId(), incomingRef.GetNodeId(), incomingRef.GetBranchId(),
				)
			}
			return nil
		}

		rel.bufferedCompletions[order] = resp
		if rel.oldestBufferedAt.IsZero() {
			rel.oldestBufferedAt = time.Now()
		}
	}

	for _, r := range toSend {
		if err := s.send(r); err != nil {
			return err
		}
	}

	return nil
}

func (s *durableTaskInvocation) staleReleaseHolds(timeout time.Duration) []orderedReleaseKey {
	s.releasesMu.Lock()
	defer s.releasesMu.Unlock()

	now := time.Now()
	var stale []orderedReleaseKey

	for key, rel := range s.releases {
		rel.mu.Lock()
		isStale := len(rel.bufferedCompletions) > 0 && !rel.oldestBufferedAt.IsZero() && now.Sub(rel.oldestBufferedAt) > timeout
		rel.mu.Unlock()

		if isStale {
			stale = append(stale, key)
		}
	}

	return stale
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

	var reqWg sync.WaitGroup

	defer func() {
		for taskId := range registeredTasks {
			d.durableInvocations.Delete(durableInvocationsKey{
				tenantId: tenantId,
				taskId:   taskId,
			})
		}
	}()

	defer reqWg.Wait()

	registerTask := func(externalIdStr string) {
		taskExtId, err := uuid.Parse(externalIdStr)
		if err != nil {
			return
		}

		if _, exists := registeredTasks[taskExtId]; !exists {
			d.durableInvocations.Store(durableInvocationsKey{
				tenantId: tenantId,
				taskId:   taskExtId,
			}, invocation)
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

			if msg, isRegisterWorker := r.req.GetMessage().(*contracts.DurableTaskRequest_RegisterWorker); isRegisterWorker {
				if err := d.handleRegisterWorker(ctx, invocation, msg.RegisterWorker); err != nil {
					d.l.Error().Err(err).Msg("error handling durable task request")
				}
				continue
			}

			switch msg := r.req.GetMessage().(type) {
			case *contracts.DurableTaskRequest_Memo:
				registerTask(msg.Memo.DurableTaskExternalId)
			case *contracts.DurableTaskRequest_TriggerRuns:
				registerTask(msg.TriggerRuns.DurableTaskExternalId)
			case *contracts.DurableTaskRequest_WaitFor:
				registerTask(msg.WaitFor.DurableTaskExternalId)
			}

			reqWg.Add(1)
			go func(req *contracts.DurableTaskRequest) {
				defer reqWg.Done()

				if err := d.handleDurableTaskRequest(ctx, invocation, req); err != nil {
					d.l.Error().Err(err).Msg("error handling durable task request")
				}
			}(r.req)
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

	requestCh := make(chan *contracts.DurableTaskRequest)
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
			d.durableInvocations.Store(durableInvocationsKey{tenantId: invocation.tenantId, taskId: taskExtId}, invocation)
			registeredTasks[taskExtId] = struct{}{}
		}
	}

	// register the task up front so async responses route back to this invocation
	// immediately, before the caller sends its first message.
	d.durableInvocations.Store(durableInvocationsKey{tenantId: invocation.tenantId, taskId: externalId}, invocation)
	registeredTasks[externalId] = struct{}{}

	go func() {
		defer deregister()

		// Cancel first: an in-flight send holds sendMu and selects on ctx.Done(), so cancelling
		// before taking sendMu unblocks it and avoids deadlocking against that sender. Marking
		// closed before close() makes later sends return an error instead of panicking.
		defer func() {
			cancel()

			for taskId := range registeredTasks {
				d.durableInvocations.Delete(durableInvocationsKey{tenantId: invocation.tenantId, taskId: taskId})
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
	ctx, span := telemetry.NewRootSpan(ctx, "dispatcher.handle-durable-task-request")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "message_type", Value: fmt.Sprintf("%T", req.GetMessage())},
	)

	switch msg := req.GetMessage().(type) {
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

func (d *DispatcherServiceImpl) deliverSatisfiedEntries(tenantId uuid.UUID, taskExternalId string, result *v1.IngestDurableTaskEventResult) error {
	switch result.Kind {
	case sqlcv1.V1DurableEventLogKindRUN:
		for _, entry := range result.TriggerRunsResult.Entries {
			if entry.IsSatisfied {
				taskExtId, _ := uuid.Parse(taskExternalId)
				if err := d.DeliverDurableEventLogEntryCompletion(
					tenantId,
					taskExtId,
					result.TriggerRunsResult.InvocationCount,
					entry.BranchId,
					entry.NodeId,
					entry.ResultPayload,
					entry.SatisfiedOrder,
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
				tenantId,
				taskExtId,
				result.MemoResult.InvocationCount,
				result.MemoResult.BranchId,
				result.MemoResult.NodeId,
				result.MemoResult.ResultPayload,
				nil,
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
				tenantId,
				taskExtId,
				result.WaitForResult.InvocationCount,
				result.WaitForResult.BranchId,
				result.WaitForResult.NodeId,
				result.WaitForResult.ResultPayload,
				result.WaitForResult.SatisfiedOrder,
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
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-memo")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "durable_task_external_id", Value: req.DurableTaskExternalId},
		telemetry.AttributeKV{Key: "invocation_count", Value: req.InvocationCount},
	)

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

	return d.deliverSatisfiedEntries(invocation.tenantId, req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleTriggerRuns(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskTriggerRunsRequest,
) error {
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-trigger-runs")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "durable_task_external_id", Value: req.DurableTaskExternalId},
		telemetry.AttributeKV{Key: "invocation_count", Value: req.InvocationCount},
		telemetry.AttributeKV{Key: "trigger_opts_count", Value: len(req.TriggerOpts)},
	)

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

	if pending := ingestionResult.TriggerRunsResult.PendingTriggers; len(pending) > 0 {
		entries := make([]tasktypes.PendingDurableRunTriggerPayload, len(pending))

		for i, p := range pending {
			entries[i] = tasktypes.PendingDurableRunTriggerPayload{
				NodeId:      p.NodeId,
				BranchId:    p.BranchId,
				TriggerOpts: p.TriggerOpts,
			}
		}

		msg, msgErr := tasktypes.DurableRunTriggerTaskMessage(invocation.tenantId, tasktypes.DurableRunTriggerMessage{
			DurableTaskId:         task.ID,
			DurableTaskInsertedAt: task.InsertedAt,
			DurableTaskExternalId: task.ExternalID,
			Entries:               entries,
		})
		if msgErr != nil {
			return status.Errorf(codes.Internal, "failed to build durable run trigger message: %v", msgErr)
		}

		// publish before acking: if this is lost, the child task never gets created and the
		// durable task would hang forever with no other recovery path, so treat it as fatal
		// to the RPC rather than best-effort
		if sendErr := d.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); sendErr != nil {
			return status.Errorf(codes.Internal, "failed to enqueue durable run trigger: %v", sendErr)
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

	return d.deliverSatisfiedEntries(invocation.tenantId, req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleWaitFor(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskWaitForRequest,
) error {
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-wait-for")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "durable_task_external_id", Value: req.DurableTaskExternalId},
		telemetry.AttributeKV{Key: "invocation_count", Value: req.InvocationCount},
	)

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

	return d.deliverSatisfiedEntries(invocation.tenantId, req.DurableTaskExternalId, ingestionResult)
}

func (d *DispatcherServiceImpl) handleCompleteMemo(
	ctx context.Context,
	invocation *durableTaskInvocation,
	req *contracts.DurableTaskCompleteMemoRequest,
) error {
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-complete-memo")
	defer span.End()

	telemetry.WithAttributes(span, telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId})

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
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-evict-invocation")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "durable_task_external_id", Value: req.DurableTaskExternalId},
		telemetry.AttributeKV{Key: "invocation_count", Value: req.InvocationCount},
	)

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

	evictRes, err := d.repo.Tasks().EvictTask(ctx, invocation.tenantId, v1.TaskIdInsertedAtRetryCount{
		Id:         task.ID,
		InsertedAt: task.InsertedAt,
		RetryCount: task.RetryCount,
	})
	if err != nil {
		return d.sendEvictionError(invocation, req, fmt.Sprintf("failed to evict task: %v", err))
	}

	if !evictRes.HasUnsatisfiedEntries {
		// see comment on the `EvictTaskResult` - if the evicted task has no unsatisfied entries, we have to immediately restore it,
		// otherwise it can hang forever since nothing will ever wake it up. note that this does cause some churn, but I think that's okay
		// in exchange for not hitting the indefinitely hang case
		restoreMsg, msgErr := tasktypes.DurableRestoreTaskMessage(invocation.tenantId, task.ExternalID, "all durable events satisfied at eviction time")
		if msgErr != nil {
			return d.sendEvictionError(invocation, req, fmt.Sprintf("failed to build restore message: %v", msgErr))
		}

		if sendErr := d.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, restoreMsg); sendErr != nil {
			return d.sendEvictionError(invocation, req, fmt.Sprintf("failed to publish restore message: %v", sendErr))
		}
	}

	if evictRes.WasEvicted {
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
	ctx, span := telemetry.NewSpan(ctx, "dispatcher.handle-worker-status")
	defer span.End()

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: invocation.tenantId},
		telemetry.AttributeKV{Key: "waiting_entries_count", Value: len(req.WaitingEntries)},
	)

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
				key := v1.IdInsertedAt{ID: t.ID, InsertedAtUnixMicros: t.InsertedAt.Time.UnixMicro()}
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

	d.evictStalledOrderedReleases(invocation)
	invocation.pruneIdleReleases(durableReleaseIdleTTL)

	return nil
}

// durableOrderedReleaseGapTimeout bounds how long a held EntryCompleted may wait for a
// missing lower satisfied_order before the invocation is evicted to restart cleanly.
const durableOrderedReleaseGapTimeout = 60 * time.Second
const durableReleaseIdleTTL = 24 * time.Hour

func (d *DispatcherServiceImpl) evictStalledOrderedReleases(invocation *durableTaskInvocation) {
	for _, key := range invocation.staleReleaseHolds(durableOrderedReleaseGapTimeout) {
		d.l.Error().Msgf(
			"durable task %s (invocation %d): ordered release stalled waiting for a missing satisfied_order for over %s; evicting to restart. "+
				"if this repeats, the task was likely forked with BranchDurableTask across an out-of-order satisfaction, which is not supported",
			key.taskExternalId, key.invocationCount, durableOrderedReleaseGapTimeout,
		)

		if err := invocation.send(&contracts.DurableTaskResponse{
			Message: &contracts.DurableTaskResponse_ServerEvict{
				ServerEvict: &contracts.DurableTaskServerEvictNotice{
					DurableTaskExternalId: key.taskExternalId.String(),
					InvocationCount:       key.invocationCount,
					Reason:                "ordered durable completion release stalled on a missing entry",
				},
			},
		}); err != nil {
			d.l.Error().Err(err).Msgf("failed to send server eviction for stalled ordered release on task %s", key.taskExternalId)
		}

		invocation.clearRelease(key)
	}
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
	return invocation.deliverOrdered(cb.TaskExternalId, cb.InvocationCount, cb.SatisfiedOrder, &contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: resp,
		},
	})
}

func (d *DispatcherServiceImpl) DeliverDurableEventLogEntryCompletion(tenantId uuid.UUID, taskExternalId uuid.UUID, invocationCount int32, branchId, nodeId int64, payload []byte, satisfiedOrder *int64, isFailure bool, errorMessage *string) error {
	inv, ok := d.durableInvocations.Load(durableInvocationsKey{
		tenantId: tenantId,
		taskId:   taskExternalId,
	})

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
	return inv.deliverOrdered(taskExternalId, invocationCount, satisfiedOrder, &contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: resp,
		},
	})
}

func (d *DispatcherServiceImpl) replayDAGStepChild(ctx context.Context, tenantId, childExternalId uuid.UUID) error {
	childTasks, err := d.repo.Tasks().FlattenExternalIds(ctx, tenantId, []uuid.UUID{childExternalId})
	if err != nil {
		return fmt.Errorf("failed to look up child task %s for replay: %w", childExternalId, err)
	}

	if len(childTasks) == 0 {
		return fmt.Errorf("child task %s not found for replay", childExternalId)
	}

	replayTasks := make([]tasktypes.TaskIdInsertedAtRetryCountWithExternalId, 0, len(childTasks))

	for _, ct := range childTasks {
		replayTasks = append(replayTasks, tasktypes.TaskIdInsertedAtRetryCountWithExternalId{
			TaskIdInsertedAtRetryCount: v1.TaskIdInsertedAtRetryCount{
				Id:         ct.ID,
				InsertedAt: ct.InsertedAt,
				RetryCount: ct.RetryCount,
			},
			WorkflowRunExternalId: ct.WorkflowRunID,
			TaskExternalId:        ct.ExternalID,
		})
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDReplayTasks,
		false,
		true,
		tasktypes.ReplayTasksPayload{Tasks: replayTasks},
	)

	if err != nil {
		return fmt.Errorf("failed to create replay message for child task %s: %w", childExternalId, err)
	}

	if err := d.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
		return fmt.Errorf("failed to send replay message for child task %s: %w", childExternalId, err)
	}

	return nil
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

	orchestratorWorkflowRunId := task.ExternalID
	workflowVersionId := req.WorkflowVersionId
	triggerOpts := []*v1.WorkflowNameTriggerOpts{{
		ReplayOrphanedChildren: true,
		ParentReExecuted:       req.ParentReExecuted,
		IsDagStepTrigger:       true,
		TriggerTaskData: &v1.TriggerTaskData{
			WorkflowName: req.WorkflowName,
			// Pin to the DAG's original version so a mid-run deploy can't retarget the step.
			WorkflowVersionId:    &workflowVersionId,
			TargetActionId:       &req.ActionId,
			UserMessage:          &stepLabel,
			Data:                 []byte(req.Input),
			AdditionalMetadata:   req.AdditionalMetadata,
			ParentExternalId:     &task.ExternalID,
			ParentTaskId:         &task.ID,
			ParentTaskInsertedAt: &task.InsertedAt.Time,
			ChildIndex:           &childIndex,
			DagParentTaskRunIds:  req.DagParentTaskRunIds,
			IsSkipped:            req.IsSkipped,
			IsCancelled:          req.IsCancelled,
			DesiredWorkerLabels:  req.DesiredWorkerLabels,
			WorkflowRunId:        &orchestratorWorkflowRunId,
			OlapDagId:            &task.ID,
			OlapDagInsertedAt:    &task.InsertedAt.Time,
		},
	}}

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

	var dags []*v1.DAGWithData
	var tasks []*v1.V1TaskWithPayload

	if pending := ingestionResult.TriggerRunsResult.PendingTriggers; len(pending) > 0 {
		createdTasks, createdDags, _, triggerErr := d.repo.DurableEvents().TriggerPendingRunEntries(ctx, tenantId, []v1.TriggerPendingRunEntriesOpt{{
			Task:        task,
			PendingRuns: pending,
		}})

		if triggerErr != nil {
			return nil, fmt.Errorf("failed to trigger pending durable runs for dag step: %w", triggerErr)
		}

		tasks = createdTasks
		dags = createdDags
	}

	if len(dags) > 0 || len(tasks) > 0 {
		if sigErr := d.triggerWriter.SignalCreated(ctx, tenantId, tasks, dags); sigErr != nil {
			d.l.Error().Err(sigErr).Msg("failed to signal created tasks/dags for dag step trigger")
		}
	}

	if len(ingestionResult.TriggerRunsResult.Entries) == 0 {
		return nil, fmt.Errorf("no entries returned from durable event ingestion")
	}

	entry := ingestionResult.TriggerRunsResult.Entries[0]

	if entry.ChildNeedsReplay {
		if err := d.replayDAGStepChild(ctx, tenantId, entry.WorkflowRunExternalId); err != nil {
			return nil, err
		}
	}

	return &operator.DAGStepTriggerResult{
		NodeId:                entry.NodeId,
		BranchId:              entry.BranchId,
		WorkflowRunExternalId: entry.WorkflowRunExternalId,
		IsSatisfied:           entry.IsSatisfied,
		ResultPayload:         entry.ResultPayload,
		IsFailure:             entry.ChildTaskIsFailure,
		ErrorMessage:          entry.ChildTaskErrorMessage,
		ReExecuted:            entry.ReExecuted,
	}, nil
}

// CancelDAGChildren cancels already-triggered DAG children on the operator's behalf, since it
// has no direct message-queue access; mirrors the admin CancelTasks path.
func (d *DispatcherServiceImpl) CancelDAGChildren(ctx context.Context, tenantId uuid.UUID, taskExternalIds []uuid.UUID) error {
	if len(taskExternalIds) == 0 {
		return nil
	}

	tasks, err := d.repo.Tasks().FlattenExternalIds(ctx, tenantId, taskExternalIds)
	if err != nil {
		return fmt.Errorf("failed to look up dag children to cancel: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	tasksToCancel := make([]v1.TaskIdInsertedAtRetryCount, 0, len(tasks))

	for _, task := range tasks {
		tasksToCancel = append(tasksToCancel, v1.TaskIdInsertedAtRetryCount{
			Id:         task.ID,
			InsertedAt: task.InsertedAt,
			RetryCount: task.RetryCount,
		})
	}

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCancelTasks,
		false,
		true,
		tasktypes.CancelTasksPayload{Tasks: tasksToCancel},
	)

	if err != nil {
		return fmt.Errorf("failed to create cancel message for dag children: %w", err)
	}

	return d.mq.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)
}
