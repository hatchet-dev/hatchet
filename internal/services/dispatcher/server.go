package dispatcher

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	telemetry_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool

	sendMu sync.Mutex
}

func (worker *subscribedWorker) StartStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunForEngineRow,
	stepRunData *dbsqlc.GetStepRunDataForEngineRow,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if stepRunData.Input != nil {
		inputBytes = stepRunData.Input
	}

	stepName := stepRun.StepReadableId.String

	action := &contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.ActionId,
		ActionPayload: string(inputBytes),
		StepName:      stepName,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
		// NOTE: This is the default because this method is unused
		Priority: 1,
	}

	if stepRunData.AdditionalMetadata != nil {
		metadataStr := string(stepRunData.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if stepRunData.ChildIndex.Valid {
		action.ChildWorkflowIndex = &stepRunData.ChildIndex.Int32
	}

	if stepRunData.ChildKey.Valid {
		action.ChildWorkflowKey = &stepRunData.ChildKey.String
	}

	if stepRunData.ParentId.Valid {
		parentId := sqlchelpers.UUIDToStr(stepRunData.ParentId)
		action.ParentWorkflowRunId = &parentId
	}

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
}

func (worker *subscribedWorker) StartStepRunFromBulk(
	ctx context.Context,
	tenantId string,
	stepRun *dbsqlc.GetStepRunBulkDataForEngineRow,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run-from-bulk") // nolint:ineffassign
	defer span.End()

	inputBytes := []byte{}

	if stepRun.Input != nil {
		inputBytes = stepRun.Input
	}

	stepName := stepRun.StepReadableId.String

	action := &contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.ActionId,
		ActionPayload: string(inputBytes),
		StepName:      stepName,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
		Priority:      stepRun.Priority,
	}

	if stepRun.AdditionalMetadata != nil {
		metadataStr := string(stepRun.AdditionalMetadata)
		action.AdditionalMetadata = &metadataStr
	}

	if stepRun.ChildIndex.Valid {
		action.ChildWorkflowIndex = &stepRun.ChildIndex.Int32
	}

	if stepRun.ChildKey.Valid {
		action.ChildWorkflowKey = &stepRun.ChildKey.String
	}

	if stepRun.ParentId.Valid {
		parentId := sqlchelpers.UUIDToStr(stepRun.ParentId)
		action.ParentWorkflowRunId = &parentId
	}

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(action)
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

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

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

	worker.sendMu.Lock()
	defer worker.sendMu.Unlock()

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         sqlchelpers.UUIDToStr(stepRun.JobId),
		JobName:       stepRun.JobName,
		JobRunId:      sqlchelpers.UUIDToStr(stepRun.JobRunId),
		StepId:        sqlchelpers.UUIDToStr(stepRun.StepId),
		StepRunId:     sqlchelpers.UUIDToStr(stepRun.SRID),
		ActionType:    contracts.ActionType_CANCEL_STEP_RUN,
		StepName:      stepRun.StepReadableId.String,
		WorkflowRunId: sqlchelpers.UUIDToStr(stepRun.WorkflowRunId),
		RetryCount:    stepRun.SRRetryCount,
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
		WebhookId:    request.WebhookId,
	}

	if request.RuntimeInfo != nil {
		opts.RuntimeInfo = &repository.RuntimeInfo{
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
	worker, err := s.repo.Worker().CreateNewWorker(ctx, tenantId, opts)

	if err == metered.ErrResourceExhausted {
		return nil, status.Errorf(codes.ResourceExhausted, "resource exhausted: tenant worker limit or concurrency limit exceeded")
	}

	if err != nil {
		s.l.Error().Err(err).Msgf("could not create worker for tenant %s", tenantId)
		return nil, err
	}

	workerId := sqlchelpers.UUIDToStr(worker.ID)

	if request.Labels != nil {
		_, err = s.upsertLabels(ctx, worker.ID, request.Labels)

		if err != nil {
			return nil, err
		}
	}

	// return the worker id to the worker
	return &contracts.WorkerRegisterResponse{
		TenantId:   tenantId,
		WorkerId:   workerId,
		WorkerName: worker.Name,
	}, nil
}

func (s *DispatcherImpl) UpsertWorkerLabels(ctx context.Context, request *contracts.UpsertWorkerLabelsRequest) (*contracts.UpsertWorkerLabelsResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	_, err := s.upsertLabels(ctx, sqlchelpers.UUIDFromStr(request.WorkerId), request.Labels)

	if err != nil {
		return nil, err
	}

	return &contracts.UpsertWorkerLabelsResponse{
		TenantId: sqlchelpers.UUIDToStr(tenant.ID),
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) upsertLabels(ctx context.Context, workerId pgtype.UUID, request map[string]*contracts.WorkerLabels) ([]*dbsqlc.WorkerLabel, error) {
	affinities := make([]repository.UpsertWorkerLabelOpts, 0, len(request))

	for key, config := range request {

		err := s.v.Validate(config)

		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "Invalid affinity config: %s", err.Error())
		}

		affinities = append(affinities, repository.UpsertWorkerLabelOpts{
			Key:      key,
			IntValue: config.IntValue,
			StrValue: config.StrValue,
		})
	}

	res, err := s.repo.Worker().UpsertWorkerLabels(ctx, workerId, affinities)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not upsert worker affinities for worker %s", sqlchelpers.UUIDToStr(workerId))
		return nil, err
	}

	return res, nil
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
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

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
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	sessionEstablished := time.Now().UTC()

	_, err = s.repo.Worker().UpdateWorkerActiveStatus(ctx, tenantId, request.WorkerId, true, sessionEstablished)

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

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				s.l.Error().Err(err).Msgf("could not update worker %s active status to false due to worker stream closing (session established %s)", request.WorkerId, sessionEstablished.String())
				return err
			}

			return nil
		case <-ctx.Done():
			s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			_, err = s.repo.Worker().UpdateWorkerActiveStatus(ctx, tenantId, request.WorkerId, false, sessionEstablished)

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
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "could not get worker")
		if errors.Is(err, pgx.ErrNoRows) {
			s.l.Error().Msgf("worker %s not found", req.WorkerId)
			return nil, err
		}

		return nil, err
	}

	// if the worker is not active, the listener should reconnect
	if worker.LastListenerEstablished.Valid && !worker.IsActive {
		span.RecordError(err)
		span.SetStatus(telemetry_codes.Error, "worker stream is not active")
		return nil, status.Errorf(codes.FailedPrecondition, "Heartbeat rejected: worker stream is not active: %s", req.WorkerId)
	}

	err = s.repo.Worker().UpdateWorkerHeartbeat(ctx, tenantId, req.WorkerId, heartbeatAt)

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
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return s.releaseSlotV0(ctx, tenant, req)
	case dbsqlc.TenantMajorEngineVersionV1:
		return s.releaseSlotV1(ctx, tenant, req)
	default:
		return nil, status.Errorf(codes.Unimplemented, "ReleaseSlot is not implemented in engine version %s", string(tenant.Version))
	}
}

func (s *DispatcherImpl) releaseSlotV0(ctx context.Context, tenant *dbsqlc.Tenant, req *contracts.ReleaseSlotRequest) (*contracts.ReleaseSlotResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	if req.StepRunId == "" {
		return nil, fmt.Errorf("step run id is required")
	}

	err := s.repo.StepRun().ReleaseStepRunSemaphore(ctx, tenantId, req.StepRunId, true)

	if err != nil {
		return nil, err
	}

	return &contracts.ReleaseSlotResponse{}, nil
}

func (s *DispatcherImpl) SubscribeToWorkflowEvents(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return s.subscribeToWorkflowEventsV0(request, stream)
	case dbsqlc.TenantMajorEngineVersionV1:
		return s.subscribeToWorkflowEventsV1(request, stream)
	default:
		return status.Errorf(codes.Unimplemented, "SubscribeToWorkflowEvents is not implemented in engine version %s", string(tenant.Version))
	}
}

func (s *DispatcherImpl) subscribeToWorkflowEventsV0(request *contracts.SubscribeToWorkflowEventsRequest, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	if request.WorkflowRunId != nil {
		return s.subscribeToWorkflowEventsByWorkflowRunId(*request.WorkflowRunId, stream)
	} else if request.AdditionalMetaKey != nil && request.AdditionalMetaValue != nil {
		return s.subscribeToWorkflowEventsByAdditionalMeta(*request.AdditionalMetaKey, *request.AdditionalMetaValue, stream)
	}

	return status.Errorf(codes.InvalidArgument, "either workflow run id or additional meta key-value must be provided")
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowEventsByAdditionalMeta(key string, value string, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	wg := sync.WaitGroup{}

	// Keep track of active workflow run IDs
	activeRunIds := make(map[string]struct{})
	var mu sync.Mutex     // Mutex to protect activeRunIds
	var sendMu sync.Mutex // Mutex to protect sending messages

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		e, err := s.tenantTaskToWorkflowEventByAdditionalMeta(
			task, tenantId, key, value,
			func(e *contracts.WorkflowEvent) (bool, error) {
				mu.Lock()
				defer mu.Unlock()

				if e.WorkflowRunId == "" {
					return false, nil
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
					return true, nil
				}

				return false, nil
			})

		if err != nil {
			s.l.Error().Err(err).Msgf("could not convert task to workflow event")
			return nil
		} else if e == nil || e.WorkflowRunId == "" {
			return nil
		}

		// send the task to the client
		sendMu.Lock()
		err = stream.Send(e)
		sendMu.Unlock()

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
	cleanupQueue, err := s.sharedReader.Subscribe(tenantId, f)

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

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowEventsByWorkflowRunId(workflowRunId string, stream contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	tenant := stream.Context().Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	s.l.Debug().Msgf("Received subscribe request for workflow: %s", workflowRunId)

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// if the workflow run is in a final state, hang up the connection
	workflowRun, err := s.repo.WorkflowRun().GetWorkflowRunById(ctx, tenantId, workflowRunId)

	if err != nil {
		if errors.Is(err, repository.ErrWorkflowRunNotFound) {
			return status.Errorf(codes.NotFound, "workflow run %s not found", workflowRunId)
		}

		return err
	}

	if repository.IsFinalWorkflowRunStatus(workflowRun.WorkflowRun.Status) {
		return nil
	}

	wg := sync.WaitGroup{}

	sendMu := sync.Mutex{}

	f := func(task *msgqueue.Message) error {
		wg.Add(1)
		defer wg.Done()

		e, err := s.tenantTaskToWorkflowEventByRunId(task, tenantId, workflowRunId)

		if err != nil {
			s.l.Error().Err(err).Msgf("could not convert task to workflow event")
			return nil
		} else if e == nil {
			return nil
		}

		// send the task to the client
		sendMu.Lock()
		err = stream.Send(e)
		sendMu.Unlock()

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
	cleanupQueue, err := s.sharedReader.Subscribe(tenantId, f)

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
			result.Error = repository.StringPtr("Error is too large to send over the Hatchet stream.")
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
	tenant := server.Context().Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return s.subscribeToWorkflowRunsV0(server)
	case dbsqlc.TenantMajorEngineVersionV1:
		return s.subscribeToWorkflowRunsV1(server)
	default:
		return status.Errorf(codes.Unimplemented, "SubscribeToWorkflowRuns is not implemented in engine version %s", string(tenant.Version))
	}
}

// SubscribeToWorkflowEvents registers workflow events with the dispatcher
func (s *DispatcherImpl) subscribeToWorkflowRunsV0(server contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
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
		results := s.cleanResults(e.Results)

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

			if _, parseErr := uuid.Parse(req.WorkflowRunId); parseErr != nil {
				s.l.Warn().Err(parseErr).Msg("invalid workflow run id")
				continue
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
	cleanupQueue, err := s.sharedReader.Subscribe(tenantId, f)

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
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return s.sendStepActionEventV0(ctx, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return s.sendStepActionEventV1(ctx, request)
	default:
		return nil, status.Errorf(codes.Unimplemented, "SendStepActionEvent is not implemented in engine version %s", string(tenant.Version))
	}
}

func (s *DispatcherImpl) sendStepActionEventV0(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		return s.handleStepRunStarted(ctx, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_ACKNOWLEDGED:
		return s.handleStepRunAcked(ctx, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.handleStepRunCompleted(ctx, request)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		return s.handleStepRunFailed(ctx, request)
	}

	return nil, fmt.Errorf("unknown event type %s", request.EventType)
}

func (s *DispatcherImpl) SendGroupKeyActionEvent(ctx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := ctx.Value("tenant").(*dbsqlc.Tenant)

	if tenant.Version == dbsqlc.TenantMajorEngineVersionV1 {
		return nil, status.Errorf(codes.Unimplemented, "SendGroupKeyActionEvent is not implemented in engine version v1")
	}

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

	switch tenant.Version {
	case dbsqlc.TenantMajorEngineVersionV0:
		return d.refreshTimeoutV0(ctx, tenant, request)
	case dbsqlc.TenantMajorEngineVersionV1:
		return d.refreshTimeoutV1(ctx, tenant, request)
	default:
		return nil, status.Errorf(codes.Unimplemented, "RefreshTimeout is not implemented in engine version %s", string(tenant.Version))
	}
}

func (d *DispatcherImpl) refreshTimeoutV0(ctx context.Context, tenant *dbsqlc.Tenant, request *contracts.RefreshTimeoutRequest) (*contracts.RefreshTimeoutResponse, error) {
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	opts := repository.RefreshTimeoutBy{
		IncrementTimeoutBy: request.IncrementTimeoutBy,
	}

	if apiErrors, err := d.v.ValidateAPI(opts); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request: %s", apiErrors.String())
	}

	pgTimeoutAt, err := d.repo.StepRun().RefreshTimeoutBy(ctx, tenantId, request.StepRunId, opts)

	if err != nil {
		return nil, err
	}

	timeoutAt := &timestamppb.Timestamp{
		Seconds: pgTimeoutAt.Time.Unix(),
		Nanos:   int32(pgTimeoutAt.Time.Nanosecond()), // nolint:gosec
	}

	return &contracts.RefreshTimeoutResponse{
		TimeoutAt: timeoutAt,
	}, nil
}

func (s *DispatcherImpl) handleStepRunStarted(inputCtx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.l.Debug().Msgf("Received step started event for step run %s", request.StepRunId)

	startedAt := request.EventTimestamp.AsTime()

	sr, err := s.repo.StepRun().GetStepRunForEngine(ctx, tenantId, request.StepRunId)

	if err != nil {
		return nil, err
	}

	err = s.repo.StepRun().StepRunStarted(ctx, tenantId, sqlchelpers.UUIDToStr(sr.WorkflowRunId), request.StepRunId, startedAt)

	if err != nil {
		return nil, fmt.Errorf("could not mark step run started: %w", err)
	}

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskPayload{
		StepRunId:     request.StepRunId,
		StartedAt:     startedAt.Format(time.RFC3339),
		WorkflowRunId: sqlchelpers.UUIDToStr(sr.WorkflowRunId),
		StepRetries:   &sr.StepRetries,
		RetryCount:    &sr.SRRetryCount,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskMetadata{
		TenantId: tenantId,
	})

	// we send the event directly to the tenant's event queue
	err = s.mq.AddMessage(ctx, msgqueue.TenantEventConsumerQueue(tenantId), &msgqueue.Message{
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

func (s *DispatcherImpl) handleStepRunAcked(inputCtx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.l.Debug().Msgf("Received step ack event for step run %s", request.StepRunId)

	startedAt := request.EventTimestamp.AsTime()

	sr, err := s.repo.StepRun().GetStepRunForEngine(ctx, tenantId, request.StepRunId)

	if err != nil {
		return nil, err
	}

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskPayload{
		StepRunId:     request.StepRunId,
		StartedAt:     startedAt.Format(time.RFC3339),
		WorkflowRunId: sqlchelpers.UUIDToStr(sr.WorkflowRunId),
		StepRetries:   &sr.StepRetries,
		RetryCount:    &sr.SRRetryCount,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err = s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
		ID:       "step-run-acked",
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

func (s *DispatcherImpl) handleStepRunCompleted(inputCtx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.repo.StepRun().ReleaseStepRunSemaphore(ctx, tenantId, request.StepRunId, false)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not release semaphore for step run %s", request.StepRunId)
		return nil, err
	}

	s.l.Debug().Msgf("Received step completed event for step run %s", request.StepRunId)

	// verify that the event payload can be unmarshalled into a map type
	if request.EventPayload != "" {
		res := make(map[string]interface{})

		if err := json.Unmarshal([]byte(request.EventPayload), &res); err != nil {
			// if the payload starts with a [, then it is an array which we don't currently support
			if request.EventPayload[0] == '[' {
				return nil, status.Errorf(codes.InvalidArgument, "Return value is an array, which is not supported")
			}

			// if the payload starts with a \", then it is a string which we don't currently support
			if request.EventPayload[0] == '"' {
				return nil, status.Errorf(codes.InvalidArgument, "Return value is a string, which is not supported")
			}

			return nil, status.Errorf(codes.InvalidArgument, "Return value is not a valid JSON object")
		}
	}

	finishedAt := request.EventTimestamp.AsTime()

	meta, err := s.repo.StepRun().GetStepRunMetaForEngine(ctx, tenantId, request.StepRunId)

	if err != nil {
		return nil, err
	}

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskPayload{
		WorkflowRunId:  sqlchelpers.UUIDToStr(meta.WorkflowRunId),
		StepRunId:      request.StepRunId,
		FinishedAt:     finishedAt.Format(time.RFC3339),
		StepOutputData: request.EventPayload,
		StepRetries:    &meta.Retries,
		RetryCount:     &meta.RetryCount,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err = s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
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

func (s *DispatcherImpl) handleStepRunFailed(inputCtx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := s.repo.StepRun().ReleaseStepRunSemaphore(ctx, tenantId, request.StepRunId, false)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not release semaphore for step run %s", request.StepRunId)
		return nil, err
	}

	s.l.Debug().Msgf("Received step failed event for step run %s", request.StepRunId)

	failedAt := request.EventTimestamp.AsTime()

	meta, err := s.repo.StepRun().GetStepRunMetaForEngine(ctx, tenantId, request.StepRunId)

	if err != nil {
		return nil, err
	}

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskPayload{
		WorkflowRunId: sqlchelpers.UUIDToStr(meta.WorkflowRunId),
		StepRunId:     request.StepRunId,
		FailedAt:      failedAt.Format(time.RFC3339),
		Error:         request.EventPayload,
		StepRetries:   &meta.Retries,
		RetryCount:    &meta.RetryCount,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskMetadata{
		TenantId: tenantId,
	})

	// send the event to the jobs queue
	err = s.mq.AddMessage(ctx, msgqueue.JOB_PROCESSING_QUEUE, &msgqueue.Message{
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

func (s *DispatcherImpl) handleGetGroupKeyRunStarted(inputCtx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func (s *DispatcherImpl) handleGetGroupKeyRunCompleted(inputCtx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func (s *DispatcherImpl) handleGetGroupKeyRunFailed(inputCtx context.Context, request *contracts.GroupKeyActionEvent) (*contracts.ActionEventResponse, error) {
	tenant := inputCtx.Value("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// run the rest on a separate context to always send to job controller
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

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

func (s *DispatcherImpl) taskToWorkflowEvent(task *msgqueue.Message, tenantId string, filter func(task *contracts.WorkflowEvent) (*bool, error), hangupFunc func(task *contracts.WorkflowEvent) (bool, error)) (*contracts.WorkflowEvent, error) {
	workflowEvent := &contracts.WorkflowEvent{}

	var stepRunId string

	switch task.ID {
	case "step-run-started":
		payload, err := UnmarshalPayload[tasktypes.StepRunStartedTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED
		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "step-run-finished":
		payload, err := UnmarshalPayload[tasktypes.StepRunFinishedTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		stepRunId = task.Payload["step_run_id"].(string)
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
		workflowEvent.EventPayload = payload.StepOutputData

		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "step-run-failed":
		payload, err := UnmarshalPayload[tasktypes.StepRunFailedTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		stepRunId = payload.StepRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED
		workflowEvent.EventPayload = payload.Error

		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "step-run-cancelled":
		payload, err := UnmarshalPayload[tasktypes.StepRunCancelledTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		stepRunId = payload.StepRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED

		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "step-run-timed-out":
		payload, err := UnmarshalPayload[tasktypes.StepRunTimedOutTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		stepRunId = payload.StepRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_TIMED_OUT

		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "step-run-stream-event":
		payload, err := UnmarshalPayload[tasktypes.StepRunStreamEventTaskPayload](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowEvent.WorkflowRunId = payload.WorkflowRunId
		stepRunId = payload.StepRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_STEP_RUN
		workflowEvent.ResourceId = stepRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM

		workflowEvent.StepRetries = payload.StepRetries
		workflowEvent.RetryCount = payload.RetryCount
	case "workflow-run-finished":
		payload, err := UnmarshalPayload[tasktypes.WorkflowRunFinishedTask](task.Payload)
		if err != nil {
			return nil, err
		}
		workflowRunId := payload.WorkflowRunId
		workflowEvent.ResourceType = contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN
		workflowEvent.ResourceId = workflowRunId
		workflowEvent.WorkflowRunId = workflowRunId
		workflowEvent.EventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
	}

	match, err := filter(workflowEvent)

	if err != nil {
		return nil, err
	}

	if match != nil && !*match {
		// if not a match, we don't return it
		return nil, nil
	}

	hangup, err := hangupFunc(workflowEvent)

	if err != nil {
		return nil, err
	}

	if hangup {
		workflowEvent.Hangup = true
		return workflowEvent, nil
	}

	if workflowEvent.ResourceType == contracts.ResourceType_RESOURCE_TYPE_STEP_RUN {
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
	}

	return workflowEvent, nil
}

func (s *DispatcherImpl) tenantTaskToWorkflowEventByRunId(task *msgqueue.Message, tenantId string, workflowRunIds ...string) (*contracts.WorkflowEvent, error) {

	workflowEvent, err := s.taskToWorkflowEvent(task, tenantId,
		func(e *contracts.WorkflowEvent) (*bool, error) {
			hit := contains(workflowRunIds, e.WorkflowRunId)
			return &hit, nil
		},
		func(e *contracts.WorkflowEvent) (bool, error) {
			// hangup on complete
			return e.ResourceType == contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN &&
				e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED, nil
		},
	)

	if err != nil {
		return nil, err
	}

	return workflowEvent, nil
}

func tinyHash(key, value string) string {
	// Combine key and value
	combined := fmt.Sprintf("%s:%s", key, value)

	// Create SHA-256 hash
	hash := sha256.Sum256([]byte(combined))

	// Take first 8 bytes of the hash
	shortHash := hash[:8]

	// Encode to base64
	encoded := base64.URLEncoding.EncodeToString(shortHash)

	// Remove padding
	return encoded[:11]
}

func (s *DispatcherImpl) tenantTaskToWorkflowEventByAdditionalMeta(task *msgqueue.Message, tenantId string, key string, value string, hangup func(e *contracts.WorkflowEvent) (bool, error)) (*contracts.WorkflowEvent, error) {
	workflowEvent, err := s.taskToWorkflowEvent(
		task,
		tenantId,
		func(e *contracts.WorkflowEvent) (*bool, error) {
			return cache.MakeCacheable[bool](
				s.cache,
				fmt.Sprintf("wfram-%s-%s-%s", tenantId, e.WorkflowRunId, tinyHash(key, value)),
				func() (*bool, error) {

					if e.WorkflowRunId == "" {
						return nil, nil
					}

					am, err := s.repo.WorkflowRun().GetWorkflowRunAdditionalMeta(context.Background(), tenantId, e.WorkflowRunId)

					if err != nil {
						return nil, err
					}

					if am.AdditionalMetadata == nil {
						f := false
						return &f, nil
					}

					var additionalMetaMap map[string]interface{}
					err = json.Unmarshal(am.AdditionalMetadata, &additionalMetaMap)
					if err != nil {
						return nil, err
					}

					if v, ok := (additionalMetaMap)[key]; ok && v == value {
						t := true
						return &t, nil
					}

					f := false
					return &f, nil

				},
			)
		},
		hangup,
	)

	if err != nil {
		return nil, err
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
	finalWorkflowRuns := make([]*dbsqlc.ListWorkflowRunsRow, 0)

	for _, workflowRun := range workflowRuns {
		wrCopy := workflowRun

		if !repository.IsFinalWorkflowRunStatus(wrCopy.WorkflowRun.Status) {
			continue
		}

		finalWorkflowRuns = append(finalWorkflowRuns, wrCopy)
	}

	res := make([]*contracts.WorkflowRunEvent, 0)

	// get step run results for each workflow run
	mappedStepRunResults, err := s.getStepResultsForWorkflowRun(tenantId, finalWorkflowRuns)

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

func (s *DispatcherImpl) getStepResultsForWorkflowRun(tenantId string, workflowRuns []*dbsqlc.ListWorkflowRunsRow) (map[string][]*contracts.StepRunResult, error) {
	workflowRunIds := make([]string, 0)
	workflowRunToOnFailureJobIds := make(map[string]string)

	for _, workflowRun := range workflowRuns {
		workflowRunIds = append(workflowRunIds, sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.ID))

		if workflowRun.WorkflowVersion.OnFailureJobId.Valid {
			workflowRunToOnFailureJobIds[sqlchelpers.UUIDToStr(workflowRun.WorkflowRun.ID)] = sqlchelpers.UUIDToStr(workflowRun.WorkflowVersion.OnFailureJobId)
		}
	}

	stepRuns, err := s.repo.StepRun().ListStepRuns(context.Background(), tenantId, &repository.ListStepRunsOpts{
		WorkflowRunIds: workflowRunIds,
	})

	if err != nil {
		return nil, err
	}

	res := make(map[string][]*contracts.StepRunResult)

	for _, stepRun := range stepRuns {

		data, err := s.repo.StepRun().GetStepRunDataForEngine(context.Background(), tenantId, sqlchelpers.UUIDToStr(stepRun.SRID))

		if err != nil {
			return nil, fmt.Errorf("could not get step run data for %s, %e", sqlchelpers.UUIDToStr(stepRun.SRID), err)
		}

		workflowRunId := sqlchelpers.UUIDToStr(stepRun.WorkflowRunId)
		jobId := sqlchelpers.UUIDToStr(stepRun.JobId)

		hasError := data.Error.Valid || stepRun.SRCancelledReason.Valid
		isOnFailureJob := workflowRunToOnFailureJobIds[workflowRunId] == jobId

		if hasError && isOnFailureJob {
			continue
		}

		resStepRun := &contracts.StepRunResult{
			StepRunId:      sqlchelpers.UUIDToStr(stepRun.SRID),
			StepReadableId: stepRun.StepReadableId.String,
			JobRunId:       sqlchelpers.UUIDToStr(stepRun.JobRunId),
		}

		if data.Error.Valid {
			resStepRun.Error = &data.Error.String
		}

		if stepRun.SRCancelledReason.Valid {
			errString := fmt.Sprintf("this step run was cancelled due to %s", stepRun.SRCancelledReason.String)
			resStepRun.Error = &errString
		}

		if data.Output != nil {
			resStepRun.Output = repository.StringPtr(string(data.Output))
		}

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
