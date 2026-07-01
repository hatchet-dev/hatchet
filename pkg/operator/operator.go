package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
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

	WorkerId() uuid.UUID

	// Cleanup tears down a single operator: it pauses the operator's worker and then drains.
	Cleanup()

	// Drain stops accepting new tasks and waits for in-flight ones, without pausing the
	// worker. Used for bulk teardown, where the caller pauses all workers in one query
	// instead of one update per operator.
	Drain()
}

type DAGStepTriggerRequest struct {
	ParentTaskExternalId uuid.UUID
	InvocationCount      int32
	WorkflowName         string
	ActionId             string
	ChildIndex           int32
	Input                string
	DagParentTaskRunIds  []uuid.UUID
	IsSkipped            bool
	IsCancelled          bool
}

type DAGStepTriggerResult struct {
	NodeId                int64
	BranchId              int64
	WorkflowRunExternalId uuid.UUID
}

type TaskEventWriter interface {
	SendStepActionEvent(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)

	CancelTaskEvent(ctx context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)

	// RegisterDurableTask opens a channel-based durable-task session: the operator (acting as
	// a durable worker) writes requests to the returned channel and reads responses from it.
	RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error)

	TriggerDAGStep(ctx context.Context, tenantId uuid.UUID, req *DAGStepTriggerRequest) (*DAGStepTriggerResult, error)
}

type SharedOperator[T any] struct {
	operatorConfig  T
	repo            repository.Repository
	taskEventWriter TaskEventWriter
	l               *zerolog.Logger
	tasks           sync.WaitGroup
	mu              sync.Mutex
	workerId        uuid.UUID
	tenantId        uuid.UUID
	shutdown        bool
}

// NewSharedOperator constructs the shared operator state.
func NewSharedOperator[T any](operator *sqlcv1.V1Operator, l *zerolog.Logger, repo repository.Repository, taskEventWriter TaskEventWriter, workerId uuid.UUID, t T) (*SharedOperator[T], error) {
	err := json.Unmarshal(operator.Config, &t)

	if err != nil {
		return nil, err
	}

	return &SharedOperator[T]{
		operatorConfig:  t,
		l:               l,
		repo:            repo,
		taskEventWriter: taskEventWriter,
		workerId:        workerId,
		tenantId:        operator.TenantID,
	}, nil
}

func (s *SharedOperator[T]) Config() T {
	return s.operatorConfig
}

func (s *SharedOperator[T]) Logger() *zerolog.Logger {
	return s.l
}

func (s *SharedOperator[T]) WorkerId() uuid.UUID {
	return s.workerId
}

func (s *SharedOperator[T]) TenantId() uuid.UUID {
	return s.tenantId
}

func (s *SharedOperator[T]) UpdateWorkerActions(ctx context.Context, actions []string) error {
	return s.repo.Operators().UpdateOperatorWorkerActions(ctx, s.tenantId, s.workerId, actions)
}

func (s *SharedOperator[T]) TriggerDAGStep(ctx context.Context, req *DAGStepTriggerRequest) (*DAGStepTriggerResult, error) {
	if s.taskEventWriter == nil {
		return nil, fmt.Errorf("operator has no task event writer configured")
	}

	ctx = context.WithValue(ctx, tenantContextKey, &sqlcv1.Tenant{ID: s.tenantId})

	return s.taskEventWriter.TriggerDAGStep(ctx, s.tenantId, req)
}

// RegisterDurableTask opens a channel-based durable-task session through the dispatcher,
// injecting the tenant the dispatcher reads off the context (the same key sendStepActionEvent
// uses). Operators that drive durable execution write requests to the returned channel and
// read responses from it.
func (s *SharedOperator[T]) RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error) {
	if s.taskEventWriter == nil {
		return nil, nil, fmt.Errorf("operator has no task event writer configured")
	}

	// the dispatcher reads the tenant off the context (see grpc auth middleware).
	ctx = context.WithValue(ctx, tenantContextKey, &sqlcv1.Tenant{ID: s.tenantId}) // nolint:staticcheck // key must match the dispatcher's

	return s.taskEventWriter.RegisterDurableTask(ctx, externalId)
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

func (s *SharedOperator[T]) SendCancelled(action *contracts.AssignedAction, msg string) error {
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
		EventPayload:      msg,
		RetryCount:        &retryCount,
	}

	ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, tenantContextKey, &sqlcv1.Tenant{ID: s.tenantId}) // nolint:staticcheck

	_, err := s.taskEventWriter.CancelTaskEvent(ctx, event)
	return err
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
	// Stop accepting new tracked tasks before pausing, so no task slips in between the pause
	// and the drain.
	s.beginShutdown()

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

	s.drainTasks()
}

func (s *SharedOperator[T]) Drain() {
	s.beginShutdown()
	s.drainTasks()
}

// beginShutdown stops accepting new tracked tasks. Setting the flag under the mutex (paired
// with the Add in RecordTask) guarantees no WaitGroup.Add races with the Wait in drainTasks.
func (s *SharedOperator[T]) beginShutdown() {
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()
}

// drainTasks waits for in-flight tasks to finish before tearing down. The manager keeps
// heartbeating this worker during the drain so it stays registered until its tasks complete.
func (s *SharedOperator[T]) drainTasks() {
	s.tasks.Wait()
}

func SlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	ac := slices.Clone(a)
	bc := slices.Clone(b)
	slices.Sort(ac)
	slices.Sort(bc)

	return slices.Equal(ac, bc)
}
