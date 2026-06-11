package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// tenantContextKey is the context key the dispatcher's SendStepActionEvent reads the tenant
// from. It must match the key set by the gRPC auth middleware
// (internal/services/grpc/middleware/auth.go).
const tenantContextKey = "tenant"

// eventReportTimeout bounds a single result-reporting call. Reporting uses a detached
// context (like the worker SDK) so a cancelled/timed-out task delivery still reports its
// outcome.
const eventReportTimeout = 30 * time.Second

type Operator interface {
	HandleAction(ctx context.Context, action *contracts.AssignedAction) error

	HandleDurableTaskRequest(ctx context.Context, req *v1contracts.DurableTaskRequest) error

	// TODO: unify interface with the TaskEventWriter below, these are basically doing the same thing, except the
	// durable task handler requires a callback stream (so we need to send it directly to the durableInvocations)
	RegisterDurableTaskHandler(func(ctx context.Context, req *v1contracts.DurableTaskResponse) error)

	WorkerId() uuid.UUID

	Cleanup()
}

type TaskEventWriter interface {
	SendStepActionEvent(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)
}

type SharedOperator[T any] struct {
	operatorConfig  T
	repo            repository.Repository
	taskEventWriter TaskEventWriter
	l               *zerolog.Logger
	cancel          func()
	tasks           sync.WaitGroup
	mu              sync.Mutex
	workerId        uuid.UUID
	tenantId        uuid.UUID
	shutdown        bool
}

func NewSharedOperator[T any](operator *sqlcv1.V1Operator, l *zerolog.Logger, repo repository.Repository, taskEventWriter TaskEventWriter, workerId uuid.UUID, t T) (*SharedOperator[T], error) {
	err := json.Unmarshal(operator.Config, &t)

	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background()) // #nosec G118 -- cancel is stored on the struct and invoked in Cleanup

	s := &SharedOperator[T]{
		operatorConfig:  t,
		l:               l,
		repo:            repo,
		taskEventWriter: taskEventWriter,
		workerId:        workerId,
		tenantId:        operator.TenantID,
		cancel:          cancel,
	}

	go s.startHeartbeatTicker(ctx)

	return s, nil
}

func (s *SharedOperator[T]) Config() T {
	return s.operatorConfig
}

func (s *SharedOperator[T]) Logger() *zerolog.Logger {
	return s.l
}

func (s *SharedOperator[T]) startHeartbeatTicker(ctx context.Context) {
	t := time.NewTicker(4 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := s.repo.Workers().UpdateWorkerHeartbeat(ctx, s.tenantId, s.workerId, time.Now().UTC())

			if err != nil {
				s.l.Error().Err(err).Msgf("could not update worker heartbeat")
			}
		}
	}
}

func (s *SharedOperator[T]) WorkerId() uuid.UUID {
	return s.workerId
}

func (s *SharedOperator[T]) UpdateWorkerActions(ctx context.Context, actions []string) error {
	return s.repo.Operators().UpdateOperatorWorkerActions(ctx, s.tenantId, s.workerId, actions)
}

// SendStarted reports that the operator has started processing the assigned action.
func (s *SharedOperator[T]) SendStarted(action *contracts.AssignedAction) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, "", nil)
}

// SendCompleted reports a successful result. output should be the task's JSON output.
func (s *SharedOperator[T]) SendCompleted(action *contracts.AssignedAction, output []byte) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, string(output), nil)
}

// SendFailed reports a failure with the given error message. shouldNotRetry, when true,
// prevents the task from being retried.
func (s *SharedOperator[T]) SendFailed(action *contracts.AssignedAction, errMsg string, shouldNotRetry bool) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, errMsg, &shouldNotRetry)
}

// sendStepActionEvent builds a StepActionEvent from the assigned action and reports it back
// through the dispatcher's TaskEventWriter. It uses a detached, time-bounded context (the
// caller's request context may already be cancelled by the time we report) and injects the
// tenant the dispatcher expects on the context.
func (s *SharedOperator[T]) sendStepActionEvent(action *contracts.AssignedAction, eventType contracts.StepActionEventType, payload string, shouldNotRetry *bool) error {
	if s.taskEventWriter == nil {
		return fmt.Errorf("operator has no task event writer configured")
	}

	retryCount := action.RetryCount

	event := &contracts.StepActionEvent{
		WorkerId:          s.workerId.String(),
		JobId:             action.JobId,
		JobRunId:          action.JobRunId,
		TaskId:            action.TaskId,
		TaskRunExternalId: action.TaskRunExternalId,
		ActionId:          action.ActionId,
		EventTimestamp:    timestamppb.Now(),
		EventType:         eventType,
		EventPayload:      payload,
		RetryCount:        &retryCount,
		ShouldNotRetry:    shouldNotRetry,
	}

	ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)
	defer cancel()

	// the dispatcher reads the tenant off the context (see grpc auth middleware).
	ctx = context.WithValue(ctx, tenantContextKey, &sqlcv1.Tenant{ID: s.tenantId}) // nolint:staticcheck // key must match the dispatcher's

	_, err := s.taskEventWriter.SendStepActionEvent(ctx, event)

	return err
}

// RecordTask registers an in-flight task and returns a release function that the caller
// must invoke (typically via defer) when the task finishes. Cleanup blocks until every
// recorded task has been released.
//
// If the operator is already shutting down, the returned release is a no-op and the task is
// not tracked — callers should generally avoid starting new work once Cleanup has begun,
// but in-flight work recorded before shutdown is always awaited.
func (s *SharedOperator[T]) RecordTask() func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.shutdown {
		return func() {}
	}

	s.tasks.Add(1)

	var once sync.Once

	return func() {
		once.Do(s.tasks.Done)
	}
}

func (s *SharedOperator[T]) Cleanup() {
	// Stop accepting new tracked tasks. Setting this under the mutex (paired with the Add in
	// RecordTask) guarantees no WaitGroup.Add races with the Wait below.
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()

	// Pause the worker so the scheduler stops assigning new tasks to it while we drain the
	// in-flight ones. Uses a detached, bounded context since the operator's own context is
	// about to be cancelled.
	if s.repo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)

		paused := true

		if _, err := s.repo.Workers().UpdateWorker(ctx, s.tenantId, s.workerId, &repository.UpdateWorkerOpts{
			IsPaused: &paused,
		}); err != nil && s.l != nil {
			s.l.Error().Err(err).Msg("could not pause operator worker on shutdown")
		}

		cancel()
	}

	// Wait for in-flight tasks to finish before tearing down. The heartbeat keeps running
	// during the drain so the worker stays registered until its tasks complete.
	s.tasks.Wait()

	s.cancel()
}

func (s *SharedOperator[T]) HandleDurableTaskRequest(ctx context.Context, req *v1contracts.DurableTaskRequest) error {
	panic("durable task handling not yet implemented for http operator")
}

func (s *SharedOperator[T]) RegisterDurableTaskHandler(handler func(ctx context.Context, req *v1contracts.DurableTaskResponse) error) {
	panic("durable task handling not yet implemented for http operator")
}
