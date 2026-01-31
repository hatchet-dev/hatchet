package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	telemetry_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

func (s *DispatcherImpl) Register(ctx context.Context, request *contracts.WorkerRegisterRequest) (*contracts.WorkerRegisterResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	s.l.Debug().Msgf("Received register request from ID %s with actions %v", request.WorkerName, request.Actions)

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
	worker, err := s.repov1.Workers().CreateNewWorker(ctx, tenantId, opts)

	if err == v1.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: tenant worker limit or concurrency limit exceeded")
	}

	if err != nil {
		s.l.Error().Err(err).Msgf("could not create worker for tenant %s", tenantId)
		return nil, err
	}

	workerId := worker.ID.String()

	if request.Labels != nil {
		_, err = s.upsertLabels(ctx, worker.ID, request.Labels)

		if err != nil {
			return nil, err
		}
	}

	// return the worker id to the worker
	return &contracts.WorkerRegisterResponse{
		TenantId:   tenantId.String(),
		WorkerId:   workerId,
		WorkerName: worker.Name,
	}, nil
}

func (s *DispatcherImpl) UpsertWorkerLabels(ctx context.Context, request *contracts.UpsertWorkerLabelsRequest) (*contracts.UpsertWorkerLabelsResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	_, err := s.upsertLabels(ctx, uuid.MustParse(request.WorkerId), request.Labels)

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
		s.l.Error().Err(err).Msgf("could not upsert worker affinities for worker %s", workerId.String())
		return nil, err
	}

	return res, nil
}

// Subscribe handles a subscribe request from a client
func (s *DispatcherImpl) Listen(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenServer) error {
	tenant := stream.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	sessionId := uuid.New().String()
	workerId := uuid.MustParse(request.WorkerId)

	s.l.Debug().Msgf("Received subscribe request from ID: %s", request.WorkerId)

	ctx := stream.Context()

	worker, err := s.repov1.Workers().GetWorkerForEngine(ctx, tenantId, workerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := worker.DispatcherId == nil || *worker.DispatcherId == uuid.Nil || *worker.DispatcherId != s.dispatcherId

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	fin := make(chan bool)

	s.workers.Add(workerId, sessionId, newSubscribedWorker(stream, fin, workerId, 20, s.pubBuffer))

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
				s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)
				return
			case <-fin:
				s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)
				return
			case <-timer.C:
				if now := time.Now().UTC(); lastHeartbeat.Add(4 * time.Second).Before(now) {
					s.l.Debug().Msgf("updating worker %s heartbeat", request.WorkerId)

					_, err := s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
						LastHeartbeatAt: &now,
						IsActive:        v1.BoolPtr(true),
					})

					if err != nil {
						if errors.Is(err, pgx.ErrNoRows) {
							return
						}

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
	tenant := stream.Context().Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID
	sessionId := uuid.New().String()
	workerId, err := uuid.Parse(request.WorkerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("invalid worker ID format: %s", request.WorkerId)
		return status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", request.WorkerId)
	}

	ctx := stream.Context()

	s.l.Debug().Msgf("Received subscribe request from ID: %s", request.WorkerId)

	worker, err := s.repov1.Workers().GetWorkerForEngine(ctx, tenantId, workerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	shouldUpdateDispatcherId := worker.DispatcherId == nil || *worker.DispatcherId == uuid.Nil || *worker.DispatcherId != s.dispatcherId

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if shouldUpdateDispatcherId {
		_, err = s.repov1.Workers().UpdateWorker(ctx, tenantId, workerId, &v1.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
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

		s.l.Error().Err(err).Msgf("could not update worker %s active status to true (session established %s, last session established %s)", request.WorkerId, sessionEstablished.String(), lastSessionEstablished)
		return err
	}

	fin := make(chan bool)

	s.workers.Add(workerId, sessionId, newSubscribedWorker(stream, fin, workerId, s.defaultMaxWorkerBacklogSize, s.pubBuffer))

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
			s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)

			_, err = s.repov1.Workers().UpdateWorkerActiveStatus(ctx, tenantId, workerId, false, sessionEstablished)

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				s.l.Error().Err(err).Msgf("could not update worker %s active status to false due to worker stream closing (session established %s)", request.WorkerId, sessionEstablished.String())
				return err
			}

			return nil
		case <-ctx.Done():
			s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			_, err = s.repov1.Workers().UpdateWorkerActiveStatus(ctx, tenantId, workerId, false, sessionEstablished)

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				s.l.Error().Err(err).Msgf("could not update worker %s active status due to worker disconnecting (session established %s)", request.WorkerId, sessionEstablished.String())
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
		s.l.Error().Err(err).Msgf("invalid worker ID format: %s", req.WorkerId)
		return nil, status.Errorf(codes.InvalidArgument, "invalid worker ID format: %s", req.WorkerId)
	}

	heartbeatAt := time.Now().UTC()

	s.l.Debug().Msgf("Received heartbeat request from ID: %s", req.WorkerId)

	// if heartbeat time is greater than expected heartbeat interval, show a warning
	if req.HeartbeatAt.AsTime().Before(heartbeatAt.Add(-1 * HeartbeatInterval)) {
		s.l.Warn().Msgf("heartbeat time is greater than expected heartbeat interval")
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
			s.l.Error().Msgf("could not update worker heartbeat: worker %s not found", req.WorkerId)
			return nil, err
		}

		return nil, err
	}

	return &contracts.HeartbeatResponse{}, nil
}

func (s *DispatcherImpl) ReleaseSlot(ctx context.Context, req *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)

	return s.releaseSlotV1(ctx, tenant, req)
}

func (s *DispatcherImpl) SubscribeToWorkflowEvents(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
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

type sendTimeFilter struct {
	mu sync.Mutex
}

func (s *sendTimeFilter) canSend() bool {
	if !s.mu.TryLock() {
		return false
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		s.mu.Unlock()
	}()

	return true
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
