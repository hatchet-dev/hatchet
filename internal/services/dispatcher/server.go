package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	telemetry_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

func (s *DispatcherImpl) Register(ctx context.Context, request *contracts.WorkerRegisterRequest) (*contracts.WorkerRegisterResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	s.l.Debug().Ctx(ctx).Msgf("Received register request from ID %s with actions %v", request.WorkerName, request.Actions)

	svcs := request.Services

	if len(svcs) == 0 {
		svcs = []string{"default"}
	}

	opts := &v1.CreateWorkerOpts{
		DispatcherId: s.dispatcherId,
		Name:         request.WorkerName,
		Actions:      request.Actions,
		Services:     svcs,
	}

	if request.RuntimeInfo != nil {
		opts.RuntimeInfo = &v1.RuntimeInfo{
			SdkVersion:      request.RuntimeInfo.SdkVersion,
			Language:        request.RuntimeInfo.Language,
			LanguageVersion: request.RuntimeInfo.LanguageVersion,
			Os:              request.RuntimeInfo.Os,
			Extra:           request.RuntimeInfo.Extra,
		}
	}

	if len(request.SlotConfig) > 0 {
		opts.SlotConfig = request.SlotConfig
	} else {
		// default to 100 slots
		opts.SlotConfig = map[string]int32{v1.SlotTypeDefault: 100}
	}

	// fixme: deprecated remove in a future release feb6 2026
	if request.Slots != nil {
		if len(request.SlotConfig) > 0 {
			return nil, status.Errorf(codes.InvalidArgument, "either slot_config or slots (deprecated) must be provided, not both")
		}

		opts.SlotConfig = map[string]int32{v1.SlotTypeDefault: *request.Slots}
	}

	if apiErrors, err := s.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	// create a worker in the database
	worker, err := s.repov1.Workers().CreateNewWorker(ctx, tenantId, opts)

	if err == v1.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: tenant worker limit or concurrency limit exceeded")
	}

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not create worker for tenant %s", tenantId)
		return nil, err
	}

	workerId := worker.ID.String()

	if request.Labels != nil {
		_, err = s.upsertLabels(ctx, worker.ID, request.Labels)

		if err != nil {
			return nil, err
		}
	}

	s.analytics.Count(ctx, analytics.Worker, analytics.Register, analytics.Props(
		"worker_name", request.WorkerName,
		"runtime_language", strings.ToLower(request.GetRuntimeInfo().GetLanguage().String()),
		"runtime_sdk_version", request.GetRuntimeInfo().GetSdkVersion(),
		"runtime_language_version", request.GetRuntimeInfo().GetLanguageVersion(),
		"runtime_os", request.GetRuntimeInfo().GetOs(),
		"runtime_extra", request.GetRuntimeInfo().GetExtra(),
		"has_labels", len(request.Labels) > 0,
		"has_webhook_id", request.WebhookId != nil,
		"has_runtime_info", request.RuntimeInfo != nil,
		"has_slot_config", len(request.SlotConfig) > 0,
		"has_custom_slots", request.Slots != nil,
		"has_services", len(request.Services) > 0,
	))

	return &contracts.WorkerRegisterResponse{
		TenantId:   tenantId.String(),
		WorkerId:   workerId,
		WorkerName: worker.Name,
	}, nil
}

func (s *DispatcherImpl) UpsertWorkerLabels(ctx context.Context, request *contracts.UpsertWorkerLabelsRequest) (*contracts.UpsertWorkerLabelsResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	s.analytics.Count(ctx, analytics.Worker, analytics.Create)
	workerId, err := uuid.Parse(request.WorkerId)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", request.WorkerId)
	}

	_, err = s.upsertLabels(ctx, workerId, request.Labels)

	if err != nil {
		return nil, err
	}

	return &contracts.UpsertWorkerLabelsResponse{
		TenantId: tenant.ID.String(),
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) upsertLabels(ctx context.Context, workerId uuid.UUID, request map[string]*contracts.WorkerLabels) ([]*sqlcv1.WorkerLabel, error) {
	affinities := make([]v1.UpsertWorkerLabelOpts, 0, len(request))

	for key, config := range request {

		err := s.v.Validate(config)

		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid affinity config: %s", err.Error())
		}

		affinities = append(affinities, v1.UpsertWorkerLabelOpts{
			Key:      key,
			IntValue: config.IntValue,
			StrValue: config.StrValue,
		})
	}

	res, err := s.repov1.Workers().UpsertWorkerLabels(ctx, workerId, affinities)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not upsert worker affinities for worker %s", workerId.String())
		return nil, err
	}

	return res, nil
}

// Subscribe handles a subscribe request from a client
func (s *DispatcherImpl) Listen(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenServer) error {
	ctx := stream.Context()
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	s.analytics.Count(ctx, analytics.Worker, analytics.Listen)
	sessionId := uuid.New().String()
	workerId, err := uuid.Parse(request.WorkerId)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("invalid worker ID format: %s", request.WorkerId)
		return status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", request.WorkerId)
	}

	s.l.Debug().Ctx(ctx).Msgf("Received subscribe request from ID: %s", request.WorkerId)

	worker, err := s.repov1.Workers().GetWorkerForEngine(ctx, tenantId, workerId)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := worker.DispatcherId == nil || (worker.DispatcherId != nil && *worker.DispatcherId != s.dispatcherId)

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	fin := make(chan bool)

	s.workers.Add(workerId, sessionId, newSubscribedWorker(stream, fin, workerId, s.defaultMaxWorkerLockAcquisitionTime, s.pubBuffer))

	defer func() {
		// non-blocking send
		select {
		case fin <- true:
		default:
		}

		s.workers.DeleteForSession(workerId, sessionId)
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
				s.l.Debug().Ctx(ctx).Msgf("worker id %s has disconnected", request.WorkerId)
				return
			case <-fin:
				s.l.Debug().Ctx(ctx).Msgf("closing stream for worker id: %s", request.WorkerId)
				return
			case <-timer.C:
				if now := time.Now().UTC(); lastHeartbeat.Add(4 * time.Second).Before(now) {
					s.l.Debug().Ctx(ctx).Msgf("updating worker %s heartbeat", request.WorkerId)

					_, err := s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
						LastHeartbeatAt: &now,
						IsActive:        v1.BoolPtr(true),
					})

					if err != nil {
						if errors.Is(err, pgx.ErrNoRows) {
							return
						}

						s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s heartbeat", request.WorkerId)
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
			s.l.Debug().Ctx(ctx).Msgf("closing stream for worker id: %s", request.WorkerId)
			return nil
		case <-ctx.Done():
			s.l.Debug().Ctx(ctx).Msgf("worker id %s has disconnected", request.WorkerId)
			return nil
		}
	}
}

// ListenV2 is like Listen, but implementation does not include heartbeats. This should only used by SDKs
// against engine version v0.18.1+
func (s *DispatcherImpl) ListenV2(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenV2Server) error {
	ctx := stream.Context()
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	s.analytics.Count(stream.Context(), analytics.Worker, analytics.Listen)
	sessionId := uuid.New().String()
	workerId, err := uuid.Parse(request.WorkerId)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("invalid worker ID format: %s", request.WorkerId)
		return status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", request.WorkerId)
	}

	s.l.Debug().Ctx(ctx).Msgf("Received subscribe request from ID: %s", request.WorkerId)

	worker, err := s.repov1.Workers().GetWorkerForEngine(ctx, tenantId, workerId)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := worker.DispatcherId == nil || (worker.DispatcherId != nil && *worker.DispatcherId != s.dispatcherId)

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	sessionEstablished := time.Now().UTC()

	_, err = s.repov1.Workers().UpdateWorkerActiveStatus(ctx, tenantId, workerId, true, sessionEstablished)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		lastSessionEstablished := "NULL"

		if worker.LastListenerEstablished.Valid {
			lastSessionEstablished = worker.LastListenerEstablished.Time.String()
		}

		s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s active status to true (session established %s, last session established %s)", request.WorkerId, sessionEstablished.String(), lastSessionEstablished)
		return err
	}

	fin := make(chan bool)

	s.workers.Add(workerId, sessionId, newSubscribedWorker(stream, fin, workerId, s.defaultMaxWorkerLockAcquisitionTime, s.pubBuffer))

	defer func() {
		// non-blocking send
		select {
		case fin <- true:
		default:
		}

		s.workers.DeleteForSession(workerId, sessionId)
	}()

	// Keep the connection alive for sending messages
	for {
		select {
		case <-fin:
			s.l.Debug().Ctx(ctx).Msgf("closing stream for worker id: %s", request.WorkerId)

			_, err = s.repov1.Workers().UpdateWorkerActiveStatus(ctx, tenantId, workerId, false, sessionEstablished)

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s active status to false due to worker stream closing (session established %s)", request.WorkerId, sessionEstablished.String())
				return err
			}

			return nil
		case <-ctx.Done():
			s.l.Debug().Ctx(ctx).Msgf("worker id %s has disconnected", request.WorkerId)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			_, err = s.repov1.Workers().UpdateWorkerActiveStatus(ctx, tenantId, workerId, false, sessionEstablished)

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				s.l.Error().Ctx(ctx).Err(err).Msgf("could not update worker %s active status due to worker disconnecting (session established %s)", request.WorkerId, sessionEstablished.String())
				return err
			}

			return nil
		}
	}
}

const HeartbeatInterval = 4 * time.Second

// Heartbeat is used to update the last heartbeat time for a worker
func (s *DispatcherImpl) Heartbeat(ctx context.Context, req *contracts.HeartbeatRequest) (*contracts.HeartbeatResponse, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-worker-heartbeat")
	defer span.End()

	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	workerId, err := uuid.Parse(req.WorkerId)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msgf("invalid worker ID format: %s", req.WorkerId)
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", req.WorkerId)
	}

	heartbeatAt := time.Now().UTC()

	s.l.Debug().Ctx(ctx).Msgf("Received heartbeat request from ID: %s", req.WorkerId)

	// if heartbeat time is greater than expected heartbeat interval, show a warning
	if req.HeartbeatAt.AsTime().Before(heartbeatAt.Add(-1 * HeartbeatInterval)) {
		s.l.Warn().Ctx(ctx).Msgf("heartbeat time is greater than expected heartbeat interval")
	}

	worker, err := s.repov1.Workers().GetWorkerForEngine(ctx, tenantId, workerId)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not get worker")
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "worker not found: %s", req.WorkerId)
		}

		return nil, err
	}

	// if the worker is not active, the listener should reconnect
	if worker.LastListenerEstablished.Valid && !worker.IsActive {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "worker stream is not active")
		return nil, status.Errorf(codes.FailedPrecondition, "Heartbeat rejected: worker stream is not active: %s", req.WorkerId)
	}

	err = s.repov1.Workers().UpdateWorkerHeartbeat(ctx, tenantId, workerId, heartbeatAt)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not update worker heartbeat")
		if errors.Is(err, pgx.ErrNoRows) {
			s.l.Error().Ctx(ctx).Msgf("could not update worker heartbeat: worker %s not found", req.WorkerId)
			return nil, err
		}

		return nil, err
	}

	// if the worker doesn't have a previous heartbeat or hasn't heartbeat in 30 seconds, notify downstream components that a
	// new worker is available
	if !worker.LastHeartbeatAt.Valid || worker.LastHeartbeatAt.Time.Before(heartbeatAt.Add(-30*time.Second)) {
		if tenant.SchedulerPartitionId.Valid {
			go func() {
				// detached from the request so the notify outlives the handler, but keeps
				// the request's values for tracing
				notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()

				msg, err := tasktypes.NotifyNewWorker(tenantId, worker.ID)

				if err != nil {
					s.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not create message for notifying new worker")
				} else {
					err = s.mqv1.SendMessage(
						notifyCtx,
						msgqueue.QueueTypeFromPartitionIDAndController(tenant.SchedulerPartitionId.String, msgqueue.Scheduler),
						msg,
					)

					if err != nil {
						s.l.Err(err).Ctx(ctx).Str("scheduler_partition_id", tenant.SchedulerPartitionId.String).Msg("could not add message to scheduler partition queue")
					}
				}
			}()
		}
	}

	return &contracts.HeartbeatResponse{}, nil
}

func (s *DispatcherImpl) ReleaseSlot(ctx context.Context, req *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	s.analytics.Count(ctx, analytics.Worker, analytics.Release)
	return s.releaseSlot(ctx, tenant, req)
}

func (s *DispatcherImpl) RestoreEvictedTask(ctx context.Context, req *contracts.RestoreEvictedTaskRequest) (*contracts.RestoreEvictedTaskResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	s.analytics.Count(ctx, analytics.DurableTask, analytics.Restore)

	return s.restoreEvictedTask(ctx, tenant, req)
}

func (s *DispatcherImpl) SubscribeToWorkflowEvents(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	if _, ok := stream.Context().Value("tenant").(*sqlcv1.Tenant); ok {
		s.analytics.Count(stream.Context(), analytics.WorkflowRun, analytics.Subscribe)
	}
	return s.subscribeToWorkflowEventsV1(request, stream)
}

// map of workflow run ids to whether the workflow runs are finished and have sent a message
// that the workflow run is finished
type workflowRunAcks struct {
	acks map[uuid.UUID]bool
	mu   sync.RWMutex
}

func (w *workflowRunAcks) addWorkflowRun(id uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.acks[id] = false
}

func (w *workflowRunAcks) getNonAckdWorkflowRuns() []uuid.UUID {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(w.acks))

	for id := range w.acks {
		if !w.acks[id] {
			ids = append(ids, id)
		}
	}

	return ids
}

func (w *workflowRunAcks) ackWorkflowRun(id uuid.UUID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.acks, id)
}

func (w *workflowRunAcks) hasWorkflowRun(id uuid.UUID) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	_, ok := w.acks[id]
	return ok
}

func (d *DispatcherImpl) cleanResults(results []*contracts.StepRunResult) []*contracts.StepRunResult {
	totalSize, sizeOfOutputs, _ := calculateResultsSize(results)

	if totalSize < d.payloadSizeThreshold {
		return results
	}

	if sizeOfOutputs >= d.payloadSizeThreshold {
		return nil
	}

	// otherwise, attempt to clean the results by removing large error fields
	cleanedResults := make([]*contracts.StepRunResult, 0, len(results))

	fieldThreshold := (d.payloadSizeThreshold - sizeOfOutputs) / len(results) // how much overhead we'd have per result or error field, in the worst case

	for _, result := range results {
		if result == nil {
			continue
		}

		// we only try to clean the error field at the moment, as modifying the output is more risky
		if result.Error != nil && len(*result.Error) > fieldThreshold {
			result.Error = v1.StringPtr("Error is too large to send over the Hatchet stream.")
		}

		cleanedResults = append(cleanedResults, result)
	}

	// if we are still over the limit, we just return nil
	if totalSize, _, _ := calculateResultsSize(cleanedResults); totalSize > d.payloadSizeThreshold {
		return nil
	}

	return cleanedResults
}

func calculateResultsSize(results []*contracts.StepRunResult) (totalSize int, sizeOfOutputs int, sizeOfErrors int) {
	for _, result := range results {
		if result != nil && result.Output != nil {
			totalSize += (len(*result.Output))
			sizeOfOutputs += (len(*result.Output))
		}

		if result != nil && result.Error != nil {
			totalSize += (len(*result.Error))
			sizeOfErrors += (len(*result.Error))
		}
	}

	return
}

func (s *DispatcherImpl) SubscribeToWorkflowRuns(server contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	s.analytics.Count(server.Context(), analytics.WorkflowRun, analytics.Subscribe)
	return s.subscribeToWorkflowRunsV1(server)
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
	return s.sendStepActionEventV1(ctx, request)
}

func (s *DispatcherImpl) SendGroupKeyActionEvent(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "SendGroupKeyActionEvent is not implemented in engine version v1")
}

func (s *DispatcherImpl) PutOverridesData(ctx context.Context, request *contracts.OverridesData) (*contracts.OverridesDataResponse, error) {
	return &contracts.OverridesDataResponse{}, nil
}

func (s *DispatcherImpl) Unsubscribe(ctx context.Context, request *contracts.WorkerUnsubscribeRequest) (*contracts.WorkerUnsubscribeResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	s.analytics.Count(ctx, analytics.Worker, analytics.Delete)

	workerId, err := uuid.Parse(request.WorkerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", request.WorkerId)
	}

	// remove the worker from the connection pool
	s.workers.Delete(workerId)

	return &contracts.WorkerUnsubscribeResponse{
		TenantId: tenantId.String(),
		WorkerId: request.WorkerId,
	}, nil
}

func (d *DispatcherImpl) RefreshTimeout(ctx context.Context, request *contracts.RefreshTimeoutRequest) (*contracts.RefreshTimeoutResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	return d.refreshTimeoutV1(ctx, tenant, request)
}

func UnmarshalPayload[T any](payload interface{}) (T, error) {
	var result T

	// Convert the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return result, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Unmarshal JSON into the desired type
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return result, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return result, nil
}

func (s *DispatcherImpl) GetVersion(ctx context.Context, req *contracts.GetVersionRequest) (*contracts.GetVersionResponse, error) {
	return &contracts.GetVersionResponse{
		Version: s.version,
	}, nil
}

type timeoutEvent struct {
	timeoutAt time.Time
	events    []*contracts.WorkflowEvent
}

type StreamEventBuffer struct {
	ctx                       context.Context
	stepRunIdToWorkflowEvents map[uuid.UUID][]*contracts.WorkflowEvent
	stepRunIdToExpectedIndex  map[uuid.UUID]int64
	stepRunIdToLastSeenTime   map[uuid.UUID]time.Time
	stepRunIdToCompletionTime map[uuid.UUID]time.Time
	eventsChan                chan *contracts.WorkflowEvent
	timedOutEventProducer     chan timeoutEvent
	cancel                    context.CancelFunc
	timeoutDuration           time.Duration
	gracePeriod               time.Duration
	mu                        sync.Mutex
}

func NewStreamEventBuffer(timeout time.Duration) *StreamEventBuffer {
	ctx, cancel := context.WithCancel(context.Background())

	buffer := &StreamEventBuffer{
		stepRunIdToWorkflowEvents: make(map[uuid.UUID][]*contracts.WorkflowEvent),
		stepRunIdToExpectedIndex:  make(map[uuid.UUID]int64),
		stepRunIdToLastSeenTime:   make(map[uuid.UUID]time.Time),
		stepRunIdToCompletionTime: make(map[uuid.UUID]time.Time),
		timeoutDuration:           timeout,
		gracePeriod:               2 * time.Second, // Wait 2 seconds after completion for late events
		eventsChan:                make(chan *contracts.WorkflowEvent, 100),
		timedOutEventProducer:     make(chan timeoutEvent, 100),
		ctx:                       ctx,
		cancel:                    cancel,
	}

	go buffer.processTimeoutEvents()
	go buffer.periodicCleanup()

	return buffer
}

func isTerminalEvent(event *contracts.WorkflowEvent) bool {
	if event == nil {
		return false
	}

	return event.ResourceType == contracts.ResourceType_RESOURCE_TYPE_STEP_RUN &&
		(event.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED ||
			event.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED ||
			event.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED)
}

func sortByEventIndex(a, b *contracts.WorkflowEvent) int {
	if a.EventIndex == nil && b.EventIndex == nil {
		if a.EventTimestamp.AsTime().Before(b.EventTimestamp.AsTime()) {
			return -1
		}

		if a.EventTimestamp.AsTime().After(b.EventTimestamp.AsTime()) {
			return 1
		}

		return 0
	}

	if a.EventIndex == nil {
		return -1
	}

	if b.EventIndex == nil {
		return 1
	}

	if *a.EventIndex < *b.EventIndex {
		return -1
	}

	if *a.EventIndex > *b.EventIndex {
		return 1
	}

	return 0
}

func (b *StreamEventBuffer) processTimeoutEvents() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case timedOut := <-b.timedOutEventProducer:
			timer := time.NewTimer(time.Until(timedOut.timeoutAt))

			select {
			case <-b.ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				b.mu.Lock()
				for _, event := range timedOut.events {
					stepRunId := uuid.MustParse(event.ResourceId)

					if bufferedEvents, exists := b.stepRunIdToWorkflowEvents[stepRunId]; exists {
						for _, e := range bufferedEvents {
							select {
							case b.eventsChan <- e:
							case <-b.ctx.Done():
								b.mu.Unlock()
								return
							}
						}

						delete(b.stepRunIdToWorkflowEvents, stepRunId)
						delete(b.stepRunIdToLastSeenTime, stepRunId)
						b.stepRunIdToExpectedIndex[stepRunId] = -1
					}
				}
				b.mu.Unlock()
			}
		}
	}
}

func (b *StreamEventBuffer) Events() <-chan *contracts.WorkflowEvent {
	return b.eventsChan
}

// Close stops the buffer's background goroutines. The channels are deliberately not
// closed: in-flight AddEvent/timeout sends race a close (send on a closed channel
// panics even inside a select), and every sender and consumer already exits via
// context cancellation.
func (b *StreamEventBuffer) Close() {
	b.cancel()
}

func (b *StreamEventBuffer) periodicCleanup() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.mu.Lock()
			now := time.Now()

			for stepRunId, completionTime := range b.stepRunIdToCompletionTime {
				if now.Sub(completionTime) > b.gracePeriod {
					delete(b.stepRunIdToWorkflowEvents, stepRunId)
					delete(b.stepRunIdToExpectedIndex, stepRunId)
					delete(b.stepRunIdToLastSeenTime, stepRunId)
					delete(b.stepRunIdToCompletionTime, stepRunId)
				}
			}

			b.mu.Unlock()
		}
	}
}

func (b *StreamEventBuffer) AddEvent(event *contracts.WorkflowEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	stepRunId := uuid.MustParse(event.ResourceId)
	now := time.Now()

	if event.ResourceType != contracts.ResourceType_RESOURCE_TYPE_STEP_RUN ||
		event.EventType != contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {

		if isTerminalEvent(event) {
			if events, exists := b.stepRunIdToWorkflowEvents[stepRunId]; exists && len(events) > 0 {
				slices.SortFunc(events, sortByEventIndex)

				for _, e := range events {
					select {
					case b.eventsChan <- e:
					case <-b.ctx.Done():
						return
					}
				}

				delete(b.stepRunIdToWorkflowEvents, stepRunId)
				delete(b.stepRunIdToExpectedIndex, stepRunId)
				delete(b.stepRunIdToLastSeenTime, stepRunId)
			}

			b.stepRunIdToCompletionTime[stepRunId] = now
		}

		select {
		case b.eventsChan <- event:
		case <-b.ctx.Done():
			return
		}
		return
	}

	b.stepRunIdToLastSeenTime[stepRunId] = now

	if _, exists := b.stepRunIdToExpectedIndex[stepRunId]; !exists {
		// IMPORTANT: Events are zero-indexed
		b.stepRunIdToExpectedIndex[stepRunId] = 0
	}

	// If EventIndex is nil, don't buffer - just release the event immediately
	if event.EventIndex == nil {
		select {
		case b.eventsChan <- event:
		case <-b.ctx.Done():
			return
		}
		return
	}

	expectedIndex := b.stepRunIdToExpectedIndex[stepRunId]

	// IMPORTANT: if expected index is -1, it means we're starting fresh after a timeout
	if expectedIndex == -1 && event.EventIndex != nil {
		b.stepRunIdToExpectedIndex[stepRunId] = *event.EventIndex
		expectedIndex = *event.EventIndex
	}

	// For stream events: if this event is the next expected one, send it immediately
	// Only buffer if it's out of order
	if *event.EventIndex == expectedIndex {
		if bufferedEvents, exists := b.stepRunIdToWorkflowEvents[stepRunId]; exists && len(bufferedEvents) > 0 {
			b.stepRunIdToWorkflowEvents[stepRunId] = append(bufferedEvents, event)
			slices.SortFunc(b.stepRunIdToWorkflowEvents[stepRunId], sortByEventIndex)

			b.sendReadyEvents(stepRunId)
		} else {
			b.stepRunIdToExpectedIndex[stepRunId] = expectedIndex + 1
			select {
			case b.eventsChan <- event:
			case <-b.ctx.Done():
				return
			}
		}
		return
	}

	if _, exists := b.stepRunIdToWorkflowEvents[stepRunId]; !exists {
		b.stepRunIdToWorkflowEvents[stepRunId] = make([]*contracts.WorkflowEvent, 0)
	}

	b.stepRunIdToWorkflowEvents[stepRunId] = append(b.stepRunIdToWorkflowEvents[stepRunId], event)
	slices.SortFunc(b.stepRunIdToWorkflowEvents[stepRunId], sortByEventIndex)

	b.sendReadyEvents(stepRunId)

	b.scheduleTimeoutIfNeeded(stepRunId, now)
}

func (b *StreamEventBuffer) scheduleTimeoutIfNeeded(stepRunId uuid.UUID, eventTime time.Time) {
	if events, exists := b.stepRunIdToWorkflowEvents[stepRunId]; exists && len(events) > 0 {
		timeoutAt := eventTime.Add(b.timeoutDuration)

		newTimeout := timeoutEvent{
			events:    append([]*contracts.WorkflowEvent{}, events...),
			timeoutAt: timeoutAt,
		}

		select {
		case b.timedOutEventProducer <- newTimeout:
		case <-b.ctx.Done():
			return
		default:
			// If the channel is full, we skip this timeout scheduling
		}
	}
}

func (b *StreamEventBuffer) sendReadyEvents(stepRunId uuid.UUID) {
	events := b.stepRunIdToWorkflowEvents[stepRunId]
	expectedIdx := b.stepRunIdToExpectedIndex[stepRunId]

	for len(events) > 0 && events[0].EventIndex != nil && *events[0].EventIndex == expectedIdx {
		select {
		case b.eventsChan <- events[0]:
		case <-b.ctx.Done():
			return
		}
		events = events[1:]
		expectedIdx++
	}

	if len(events) == 0 {
		// delete rather than store an empty slice: processTimeoutEvents treats a present
		// entry as "events still buffered" and would reset the expected index
		delete(b.stepRunIdToWorkflowEvents, stepRunId)
	} else {
		b.stepRunIdToWorkflowEvents[stepRunId] = events
	}

	b.stepRunIdToExpectedIndex[stepRunId] = expectedIdx
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowRunsV1(server contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	ctx := server.Context()
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	s.l.Debug().Ctx(ctx).Msgf("Received subscribe request for tenant: %s", tenantId)

	acks := &workflowRunAcks{
		acks: make(map[uuid.UUID]bool),
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deregister := s.streamSessions.Register(cancel)
	defer deregister()

	wg := sync.WaitGroup{}
	sendMu := sync.Mutex{}
	ringIndex := 0
	ringMu := sync.Mutex{}

	sendEvent := func(ctx context.Context, e *contracts.WorkflowRunEvent) error {
		_, sendEventSpan := telemetry.NewSpan(ctx, "subscribe_to_workflow_runs_v1.send_event")
		defer sendEventSpan.End()

		results := s.cleanResults(e.Results)

		if results == nil {
			s.l.Warn().Ctx(ctx).Msgf("results size for workflow run %s exceeds 3MB and cannot be reduced", e.WorkflowRunId)
			e.Results = nil
		}

		sendMu.Lock()
		defer sendMu.Unlock()

		workflowRunId, err := uuid.Parse(e.WorkflowRunId)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msgf("could not parse workflow run id %s", e.WorkflowRunId)
			return err
		}

		// only send if it has not been concurrently sent by another process
		shouldSend := acks.hasWorkflowRun(workflowRunId)

		if shouldSend {
			err := server.Send(e)

			if err != nil {
				s.l.Error().Ctx(ctx).Err(err).Msgf("could not send workflow event for run %s", e.WorkflowRunId)
				return err
			}
		}

		acks.ackWorkflowRun(workflowRunId)

		return nil
	}

	iter := func(workflowRunIds []uuid.UUID) error {
		if len(workflowRunIds) == 0 {
			return nil
		}

		iterCtx, iterSpan := telemetry.NewSpan(ctx, "subscribe_to_workflow_runs_v1.iter")
		defer iterSpan.End()

		bufferSize := s.workflowRunBufferSize

		if len(workflowRunIds) > bufferSize {
			ringMu.Lock()

			start := ringIndex % len(workflowRunIds)

			if start+bufferSize <= len(workflowRunIds) {
				workflowRunIds = workflowRunIds[start : start+bufferSize]
				ringIndex = start + bufferSize
			} else {
				end := (start + bufferSize) % len(workflowRunIds)
				workflowRunIds = append(workflowRunIds[start:], workflowRunIds[:end]...)
				ringIndex = end
			}

			if ringIndex >= len(workflowRunIds) {
				ringIndex = 0
			}

			ringMu.Unlock()
		}

		start := time.Now()

		finalizedWorkflowRuns, err := s.repov1.Tasks().ListFinalizedWorkflowRuns(iterCtx, tenantId, workflowRunIds)

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not list finalized workflow runs")
			return err
		}

		events, err := s.taskEventsToWorkflowRunEvent(tenantId, finalizedWorkflowRuns)

		// Release the reference to finalizedWorkflowRuns so GC can reclaim the large
		// payload byte slices while we're sending events (which can be slow due to
		// sendMu serialization). The event data has already been copied to strings.
		finalizedWorkflowRuns = nil // nolint: ineffassign

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msg("could not convert task events to workflow run events")
			return err
		}

		if time.Since(start) > 100*time.Millisecond {
			s.l.Warn().Ctx(ctx).Msgf("list finalized workflow runs for %d workflows took %s", len(workflowRunIds), time.Since(start))
		}

		for _, event := range events {
			err := sendEvent(iterCtx, event)

			if err != nil {
				return err
			}
		}

		return nil
	}

	f := func(msg *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		if matchedWorkflowRunIds, ok := s.isMatchingWorkflowRunV1(msg, acks); ok {
			if err := iter(matchedWorkflowRunIds); err != nil {
				s.l.Error().Ctx(ctx).Err(err).Msg("could not iterate over workflow runs")
			}
		}

		return nil
	}

	// subscribe to the task queue for the tenant
	cleanupQueue, err := s.sharedNonBufferedReaderv1.Subscribe(tenantId, f)

	if err != nil {
		return err
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

				s.l.Error().Ctx(ctx).Err(err).Msg("could not receive message from client")
				return
			}

			workflowRunId, err := uuid.Parse(req.WorkflowRunId)

			if err != nil {
				s.l.Warn().Ctx(ctx).Err(err).Msg("invalid workflow run id")
				continue
			}

			acks.addWorkflowRun(workflowRunId)
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
					s.l.Error().Ctx(ctx).Err(err).Msg("could not iterate over workflow runs")
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

func (s *DispatcherImpl) taskEventsToWorkflowRunEvent(tenantId uuid.UUID, finalizedWorkflowRuns []*v1.ListFinalizedWorkflowRunsResponse) ([]*contracts.WorkflowRunEvent, error) {
	res := make([]*contracts.WorkflowRunEvent, 0)

	for _, wr := range finalizedWorkflowRuns {
		status := contracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED
		stepRunResults := make([]*contracts.StepRunResult, 0)

		for _, event := range wr.OutputEvents {
			res := &contracts.StepRunResult{
				TaskRunExternalId: event.TaskExternalId.String(),
				TaskName:          event.StepReadableID,
				JobRunId:          event.TaskExternalId.String(),
			}

			switch event.EventType {
			case sqlcv1.V1TaskEventTypeCOMPLETED:
				out := string(event.Output)

				res.Output = &out
			case sqlcv1.V1TaskEventTypeFAILED:
				res.Error = &event.ErrorMessage
			case sqlcv1.V1TaskEventTypeCANCELLED:
				//FIXME: this should be more specific for schedule timeouts
				res.Error = &event.ErrorMessage
			}

			stepRunResults = append(stepRunResults, res)
		}

		res = append(res, &contracts.WorkflowRunEvent{
			WorkflowRunId:  wr.WorkflowRunId.String(),
			EventType:      status,
			EventTimestamp: timestamppb.New(time.Now()),
			Results:        stepRunResults,
		})
	}

	return res, nil
}

func (s *DispatcherImpl) sendStepActionEventV1(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	// if there's no retry count, we need to read it from the task, so we can't skip the cache
	skipCache := request.RetryCount == nil
	taskExternalId, err := uuid.Parse(request.TaskRunExternalId)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid task external run id %s: %v", request.TaskRunExternalId, err)
	}

	task, err := s.getSingleTask(ctx, tenant.ID, taskExternalId, skipCache)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid task external run id %s: %v", request.TaskRunExternalId, err)
	}

	retryCount := task.RetryCount

	if request.RetryCount != nil {
		retryCount = *request.RetryCount
	} else {
		s.l.Warn().Ctx(ctx).Msg("retry count is nil, using task's current retry count")
	}

	if request.EventType == contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED {
		if validationErr := v1.ValidateJSONB([]byte(request.EventPayload), "taskOutput"); validationErr != nil {
			request.EventPayload = validationErr.Error()
			request.EventType = contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED
		}
	}

	var durableInvCount int32
	invocationCounts, err := s.repov1.DurableEvents().GetDurableTaskInvocationCounts(ctx, tenant.ID, []v1.IdInsertedAt{
		{ID: task.ID, InsertedAt: task.InsertedAt},
	})
	if err == nil {
		if count, ok := invocationCounts[v1.IdInsertedAt{ID: task.ID, InsertedAt: task.InsertedAt}]; ok && count != nil {
			durableInvCount = *count
		}
	}

	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		return s.handleTaskStarted(ctx, task, retryCount, durableInvCount, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_ACKNOWLEDGED:
		// TODO: IMPLEMENT
		tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
		return &contracts.ActionEventResponse{
			TenantId: tenant.ID.String(),
			WorkerId: request.WorkerId,
		}, nil
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.handleTaskCompleted(ctx, task, retryCount, durableInvCount, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		return s.handleTaskFailed(ctx, task, retryCount, durableInvCount, request)
	}

	return nil, status.Errorf(codes.InvalidArgument, "invalid task external run id %s", request.TaskRunExternalId)
}

func (s *DispatcherImpl) handleTaskStarted(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount, durableInvocationCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	msg, err := tasktypes.MonitoringEventMessageFromActionEvent(
		tenantId,
		task.ID,
		retryCount,
		durableInvocationCount,
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
		TenantId: tenantId.String(),
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleTaskCompleted(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount int32, durableInvocationCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	ctx := inputCtx
	tenant := inputCtx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	// if request.RetryCount == nil {
	// 	return nil, fmt.Errorf("retry count is required in v2")
	// }

	msg, err := tasktypes.CompletedTaskMessage(
		tenantId,
		task.ID,
		task.InsertedAt,
		task.ExternalID,
		task.WorkflowRunID,
		retryCount,
		[]byte(request.EventPayload),
	)

	if err != nil {
		return nil, err
	}

	err = s.mqv1.SendMessage(inputCtx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	resp := &contracts.ActionEventResponse{
		TenantId: tenantId.String(),
		WorkerId: request.WorkerId,
	}

	olapMsg, err := tasktypes.MonitoringEventMessageFromActionEvent(
		tenantId,
		task.ID,
		retryCount,
		durableInvocationCount,
		request,
	)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msg("could not create monitoring event message")
		return resp, nil
	}

	err = s.pubBuffer.Pub(inputCtx, msgqueue.OLAP_QUEUE, olapMsg, false)

	if err != nil {
		s.l.Error().Ctx(ctx).Err(err).Msg("could not publish monitoring event message")
		return resp, nil
	}

	return resp, nil
}

func (s *DispatcherImpl) handleTaskFailed(inputCtx context.Context, task *sqlcv1.FlattenExternalIdsRow, retryCount int32, durableInvocationCount int32, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	shouldNotRetry := false

	if request.ShouldNotRetry != nil {
		shouldNotRetry = *request.ShouldNotRetry
	}

	msg, err := tasktypes.FailedTaskMessage(
		tenantId,
		task.ID,
		task.InsertedAt,
		task.ExternalID,
		task.WorkflowRunID,
		retryCount,
		true,
		request.EventPayload,
		shouldNotRetry,
	)

	if err != nil {
		return nil, err
	}

	err = s.mqv1.SendMessage(inputCtx, msgqueue.TASK_PROCESSING_QUEUE, msg)

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: tenantId.String(),
		WorkerId: request.WorkerId,
	}, nil
}

func (d *DispatcherImpl) getSingleTask(ctx context.Context, tenantId, taskExternalId uuid.UUID, skipCache bool) (*sqlcv1.FlattenExternalIdsRow, error) {
	return d.repov1.Tasks().GetTaskByExternalId(ctx, tenantId, taskExternalId, skipCache)
}

func (d *DispatcherImpl) refreshTimeoutV1(ctx context.Context, tenant *sqlcv1.Tenant, request *contracts.RefreshTimeoutRequest) (*contracts.RefreshTimeoutResponse, error) {
	tenantId := tenant.ID
	taskExternalId, err := uuid.Parse(request.TaskRunExternalId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid task external run id %s: %v", request.TaskRunExternalId, err)
	}

	opts := v1.RefreshTimeoutBy{
		TaskExternalId:     taskExternalId,
		IncrementTimeoutBy: request.IncrementTimeoutBy,
	}

	if apiErrors, validationErr := d.v.ValidateAPI(opts); validationErr != nil {
		return nil, validationErr
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	taskRuntime, err := d.repov1.Tasks().RefreshTimeoutBy(ctx, tenantId, opts)

	if err != nil {
		return nil, err
	}

	workerId := taskRuntime.WorkerID

	// send to the OLAP repository
	msg, err := tasktypes.MonitoringEventMessageFromInternal(
		tenantId,
		tasktypes.CreateMonitoringEventPayload{
			TaskId:         taskRuntime.TaskID,
			RetryCount:     taskRuntime.RetryCount,
			WorkerId:       workerId,
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

func (d *DispatcherImpl) releaseSlot(ctx context.Context, tenant *sqlcv1.Tenant, request *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenantId := tenant.ID
	stepRunId, err := uuid.Parse(request.TaskRunExternalId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid task external run id %s: %v", request.TaskRunExternalId, err)
	}

	releasedSlot, err := d.repov1.Tasks().ReleaseSlot(ctx, tenantId, stepRunId)

	if err != nil {
		return nil, err
	}

	workerId := releasedSlot.WorkerID

	// send to the OLAP repository
	msg, err := tasktypes.MonitoringEventMessageFromInternal(
		tenantId,
		tasktypes.CreateMonitoringEventPayload{
			TaskId:         releasedSlot.TaskID,
			RetryCount:     releasedSlot.RetryCount,
			WorkerId:       workerId,
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

func (d *DispatcherImpl) restoreEvictedTask(ctx context.Context, tenant *sqlcv1.Tenant, request *contracts.RestoreEvictedTaskRequest) (*contracts.RestoreEvictedTaskResponse, error) {
	tenantId := tenant.ID
	taskExternalId, err := uuid.Parse(request.TaskRunExternalId)
	if err != nil {
		return nil, fmt.Errorf("invalid task_run_external_id: %w", err)
	}

	msg, err := tasktypes.DurableRestoreTaskMessage(tenantId, taskExternalId, "Restore via dispatcher RPC")
	if err != nil {
		return nil, err
	}

	err = d.mqv1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg)
	if err != nil {
		return nil, err
	}

	return &contracts.RestoreEvictedTaskResponse{Requeued: true}, nil
}

func (s *DispatcherImpl) subscribeToWorkflowEventsV1(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	if request.WorkflowRunId != nil {
		workflowRunId, err := uuid.Parse(*request.WorkflowRunId)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid workflow run id %s: %v", *request.WorkflowRunId, err)
		}

		return s.subscribeToWorkflowEventsByWorkflowRunIdV1(workflowRunId, stream)
	} else if request.AdditionalMetaKey != nil && request.AdditionalMetaValue != nil {
		return s.subscribeToWorkflowEventsByAdditionalMetaV1(*request.AdditionalMetaKey, *request.AdditionalMetaValue, stream)
	}

	return status.Errorf(codes.InvalidArgument, "either workflow run id or additional meta key-value must be provided")
}

func (s *DispatcherImpl) subscribeToWorkflowEventsByWorkflowRunIdV1(workflowRunId uuid.UUID, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	deregister := s.streamSessions.Register(cancel)
	defer deregister()

	retries := 0
	foundWorkflowRun := false

	for retries < 10 {
		wr, err := s.repov1.OLAP().ReadWorkflowRun(ctx, workflowRunId)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				retries++
				time.Sleep(1 * time.Second)
				continue
			}

			return err
		}

		if wr == nil || wr.WorkflowRun == nil {
			retries++
			time.Sleep(1 * time.Second)
			continue
		}

		if wr.WorkflowRun.TenantID != tenantId {
			return status.Errorf(codes.NotFound, "workflow run %s not found", workflowRunId)
		}

		if wr.WorkflowRun.ReadableStatus == sqlcv1.V1ReadableStatusOlapCANCELLED ||
			wr.WorkflowRun.ReadableStatus == sqlcv1.V1ReadableStatusOlapCOMPLETED ||
			wr.WorkflowRun.ReadableStatus == sqlcv1.V1ReadableStatusOlapFAILED {
			return nil
		}

		foundWorkflowRun = true
		break
	}

	if !foundWorkflowRun {
		return status.Errorf(codes.NotFound, "workflow run %s not found", workflowRunId)
	}

	wg := sync.WaitGroup{}
	var mu sync.Mutex     // Mutex to protect activeRunIds
	var sendMu sync.Mutex // Mutex to protect sending messages

	streamBuffer := NewStreamEventBuffer(s.streamEventBufferTimeout)
	defer streamBuffer.Close()

	// Handle events from the stream buffer
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-streamBuffer.Events():
				if !ok {
					return
				}

				sendMu.Lock()
				err := stream.Send(event)
				sendMu.Unlock()

				if err != nil {
					s.l.Error().Ctx(ctx).Err(err).Msgf("could not send workflow event to client")
					cancel()
					return
				}

				if event.Hangup {
					cancel()
					return
				}
			}
		}
	}()

	f := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		wg.Add(1)
		defer wg.Done()

		events, err := s.msgsToWorkflowEvent(
			msgId,
			payloads,
			func(events []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error) {
				workflowRunIds := make([]uuid.UUID, 0)
				workflowRunIdsToEvents := make(map[string][]*contracts.WorkflowEvent)

				for _, e := range events {
					wri, err := uuid.Parse(e.WorkflowRunId)

					if err != nil {
						return nil, err
					}

					if wri != workflowRunId {
						continue
					}

					workflowRunIds = append(workflowRunIds, wri)
					workflowRunIdsToEvents[e.WorkflowRunId] = append(workflowRunIdsToEvents[e.WorkflowRunId], e)
				}

				workflowRuns, err := s.listWorkflowRuns(ctx, tenantId, workflowRunIds)

				if err != nil {
					return nil, err
				}

				workflowRunIdsToRow := make(map[uuid.UUID]*listWorkflowRunsResult)

				for _, wr := range workflowRuns {
					workflowRunIdsToRow[wr.WorkflowRunId] = wr
				}

				res := make([]*contracts.WorkflowEvent, 0)

				for _, es := range workflowRunIdsToEvents {
					res = append(res, es...)
				}

				return res, nil
			},
			func(events []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error) {
				mu.Lock()
				defer mu.Unlock()

				res := make([]*contracts.WorkflowEvent, 0)

				for _, e := range events {
					if e.WorkflowRunId == "" {
						continue
					}

					isWorkflowRunCompletedEvent := e.ResourceType == contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN &&
						(e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED || e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED || e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED)

					if isWorkflowRunCompletedEvent {
						e.Hangup = true
					}

					res = append(res, e)
				}

				return res, nil
			})

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msgf("could not convert task to workflow event")
			return nil
		} else if len(events) == 0 {
			return nil
		}

		// send the task to the client
		for _, e := range events {
			streamBuffer.AddEvent(e)
		}

		return nil
	}

	// subscribe to the task queue for the tenant
	cleanupQueue, err := s.sharedBufferedReaderv1.Subscribe(tenantId, f)

	if err != nil {
		return fmt.Errorf("could not subscribe to shared tenant queue: %w", err)
	}

	<-ctx.Done()

	// the consumer goroutine has exited with the context, so close the buffer now:
	// otherwise in-flight f callbacks block sending to the buffer's full channel until
	// this function returns, and waitFor below waits on those same callbacks
	streamBuffer.Close()

	if err := cleanupQueue(); err != nil {
		return fmt.Errorf("could not cleanup queue: %w", err)
	}

	waitFor(&wg, 60*time.Second, s.l)

	return nil
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowEventsByAdditionalMetaV1(key string, value string, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	deregister := s.streamSessions.Register(cancel)
	defer deregister()

	wg := sync.WaitGroup{}

	// Keep track of active workflow run IDs
	activeRunIds := make(map[string]struct{})
	var mu sync.Mutex     // Mutex to protect activeRunIds
	var sendMu sync.Mutex // Mutex to protect sending messages

	f := func(tenantId uuid.UUID, msgId string, payloads [][]byte) error {
		wg.Add(1)
		defer wg.Done()

		events, err := s.msgsToWorkflowEvent(
			msgId,
			payloads,
			func(events []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error) {
				workflowRunIds := make([]uuid.UUID, 0)
				workflowRunIdsToEvents := make(map[uuid.UUID][]*contracts.WorkflowEvent)

				for _, e := range events {
					workflowRunId, err := uuid.Parse(e.WorkflowRunId)

					if err != nil {
						return nil, err
					}

					workflowRunIds = append(workflowRunIds, workflowRunId)
					workflowRunIdsToEvents[workflowRunId] = append(workflowRunIdsToEvents[workflowRunId], e)
				}

				workflowRuns, err := s.listWorkflowRuns(ctx, tenantId, workflowRunIds)

				if err != nil {
					return nil, err
				}

				workflowRunIdsToRow := make(map[uuid.UUID]*listWorkflowRunsResult)

				for _, wr := range workflowRuns {
					workflowRunIdsToRow[wr.WorkflowRunId] = wr
				}

				res := make([]*contracts.WorkflowEvent, 0)

				for workflowRunId, es := range workflowRunIdsToEvents {
					if row, ok := workflowRunIdsToRow[workflowRunId]; ok {
						if v, ok := row.AdditionalMetadata[key]; ok && v == value {
							res = append(res, es...)
						}
					}
				}

				return res, nil
			},
			func(events []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error) {
				mu.Lock()
				defer mu.Unlock()

				res := make([]*contracts.WorkflowEvent, 0)

				for _, e := range events {
					if e.WorkflowRunId == "" {
						continue
					}

					isWorkflowRunCompletedEvent := e.ResourceType == contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN &&
						e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED

					if !isWorkflowRunCompletedEvent {
						// Add the run ID to active runs
						activeRunIds[e.WorkflowRunId] = struct{}{}
					} else {
						// Remove the completed run from active runs
						delete(activeRunIds, e.WorkflowRunId)
					}

					// Only return true to hang up if we've seen at least one run and all runs are completed
					if len(activeRunIds) == 0 {
						e.Hangup = true
					}

					res = append(res, e)
				}

				return res, nil
			})

		if err != nil {
			s.l.Error().Ctx(ctx).Err(err).Msgf("could not convert task to workflow event")
			return nil
		} else if len(events) == 0 {
			return nil
		}

		// send the task to the client
		for _, e := range events {
			sendMu.Lock()
			err = stream.Send(e)
			sendMu.Unlock()

			if err != nil {
				cancel()
				s.l.Error().Ctx(ctx).Err(err).Msgf("could not send workflow event to client")
				return nil
			}

			if e.Hangup {
				cancel()
			}
		}

		return nil
	}

	// subscribe to the task queue for the tenant
	cleanupQueue, err := s.sharedBufferedReaderv1.Subscribe(tenantId, f)

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

type listWorkflowRunsResult struct {
	AdditionalMetadata map[string]interface{}
	WorkflowRunId      uuid.UUID
}

func (s *DispatcherImpl) listWorkflowRuns(ctx context.Context, tenantId uuid.UUID, workflowRunIds []uuid.UUID) ([]*listWorkflowRunsResult, error) {
	// use cache heavily
	res := make([]*listWorkflowRunsResult, 0)
	workflowRunIdsToLookup := make([]uuid.UUID, 0)

	for _, workflowRunId := range workflowRunIds {
		k := fmt.Sprintf("%s-%s", tenantId, workflowRunId)
		if val, ok := s.cache.Get(k); ok {
			if valResult, ok := val.(*listWorkflowRunsResult); ok {
				res = append(res, valResult)
				continue
			}
		}

		workflowRunIdsToLookup = append(workflowRunIdsToLookup, workflowRunId)
	}

	foundWorkflowRuns := make(map[uuid.UUID]*listWorkflowRunsResult)

	flattenedRows, err := s.repov1.Tasks().FlattenExternalIds(ctx, tenantId, workflowRunIdsToLookup)

	if err != nil {
		return nil, err
	}

	for _, row := range flattenedRows {
		workflowRunId := row.WorkflowRunID
		if _, ok := foundWorkflowRuns[workflowRunId]; ok {
			continue
		}

		result := &listWorkflowRunsResult{
			WorkflowRunId: workflowRunId,
		}

		if len(row.AdditionalMetadata) > 0 {
			var additionalMetaMap map[string]interface{}
			err = json.Unmarshal(row.AdditionalMetadata, &additionalMetaMap)
			if err != nil {
				return nil, err
			}

			result.AdditionalMetadata = additionalMetaMap
		}

		res = append(res, result)
	}

	return res, nil
}

func (s *DispatcherImpl) msgsToWorkflowEvent(msgId string, payloads [][]byte, filter func(tasks []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error), hangupFunc func(tasks []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error)) ([]*contracts.WorkflowEvent, error) {
	workflowEvents := []*contracts.WorkflowEvent{}

	switch msgId {
	case "created-task":
		payloads := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

		for _, payload := range payloads {
			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.WorkflowRunID.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
				ResourceId:     payload.ExternalID.String(),
				EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED,
				EventTimestamp: timestamppb.New(payload.InsertedAt.Time),
				RetryCount:     &payload.RetryCount,
			})
		}
	case "task-completed":
		payloads := msgqueue.JSONConvert[tasktypes.CompletedTaskPayload](payloads)

		for _, payload := range payloads {
			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.WorkflowRunId.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
				ResourceId:     payload.ExternalId.String(),
				EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
				EventTimestamp: timestamppb.New(time.Now()),
				RetryCount:     &payload.RetryCount,
				EventPayload:   string(payload.Output),
			})
		}
	case "task-failed":
		payloads := msgqueue.JSONConvert[tasktypes.FailedTaskPayload](payloads)

		for _, payload := range payloads {
			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.WorkflowRunId.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
				ResourceId:     payload.ExternalId.String(),
				EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED,
				EventTimestamp: timestamppb.New(time.Now()),
				RetryCount:     &payload.RetryCount,
				EventPayload:   payload.ErrorMsg,
			})
		}
	case "task-cancelled":
		payloads := msgqueue.JSONConvert[tasktypes.CancelledTaskPayload](payloads)

		for _, payload := range payloads {
			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.WorkflowRunId.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
				ResourceId:     payload.ExternalId.String(),
				EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED,
				EventTimestamp: timestamppb.New(time.Now()),
				RetryCount:     &payload.RetryCount,
			})
		}
	case "task-stream-event":
		payloads := msgqueue.JSONConvert[tasktypes.StreamEventPayload](payloads)

		for _, payload := range payloads {
			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.WorkflowRunId.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
				ResourceId:     payload.TaskRunId.String(),
				EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
				EventTimestamp: timestamppb.New(payload.CreatedAt),
				EventPayload:   string(payload.Payload),
				EventIndex:     payload.EventIndex,
			})
		}
	case "workflow-run-finished":
		payloads := msgqueue.JSONConvert[tasktypes.NotifyFinalizedPayload](payloads)

		for _, payload := range payloads {
			eventType := contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED

			switch payload.Status {
			case sqlcv1.V1ReadableStatusOlapCANCELLED:
				eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED
			case sqlcv1.V1ReadableStatusOlapFAILED:
				eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED
			case sqlcv1.V1ReadableStatusOlapCOMPLETED:
				eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
			}

			workflowEvents = append(workflowEvents, &contracts.WorkflowEvent{
				WorkflowRunId:  payload.ExternalId.String(),
				ResourceType:   contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN,
				ResourceId:     payload.ExternalId.String(),
				EventType:      eventType,
				EventTimestamp: timestamppb.New(time.Now()),
			})
		}
	}

	matches, err := filter(workflowEvents)

	if err != nil {
		return nil, err
	}

	matches, err = hangupFunc(matches)

	if err != nil {
		return nil, err
	}

	// order matches
	slices.SortFunc(matches, func(a, b *contracts.WorkflowEvent) int {
		// anything with a hangup should be last
		if a.Hangup && !b.Hangup {
			return 1
		} else if !a.Hangup && b.Hangup {
			return -1
		}

		return sortByEventIndex(a, b)
	})

	return matches, nil
}

func (s *DispatcherImpl) isMatchingWorkflowRunV1(msg *msgqueue.Message, acks *workflowRunAcks) ([]uuid.UUID, bool) {
	switch msg.ID {
	case "workflow-run-finished":
		payloads := msgqueue.JSONConvert[tasktypes.NotifyFinalizedPayload](msg.Payloads)
		res := make([]uuid.UUID, 0)

		for _, payload := range payloads {
			if acks.hasWorkflowRun(payload.ExternalId) {
				res = append(res, payload.ExternalId)
			}
		}

		if len(res) == 0 {
			return nil, false
		}

		return res, true
	case "workflow-run-finished-candidate":
		payloads := msgqueue.JSONConvert[tasktypes.CandidateFinalizedPayload](msg.Payloads)
		res := make([]uuid.UUID, 0)

		for _, payload := range payloads {
			if acks.hasWorkflowRun(payload.WorkflowRunId) {
				res = append(res, payload.WorkflowRunId)
			}
		}

		if len(res) == 0 {
			return nil, false
		}

		return res, true
	default:
		return nil, false
	}
}
