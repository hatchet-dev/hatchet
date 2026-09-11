package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// This file holds the operator side of the contract (ActionHandler, Operator) and the
// engine-internal surface that only in-process operators get (TaskEventWriter and the
// SharedOperator helpers). The hosting contract itself is in host.go.

// eventReportTimeout bounds a single result-reporting call. Reporting uses a detached
// context (like the worker SDK) so a cancelled/timed-out task delivery still reports its
// outcome.
const eventReportTimeout = 30 * time.Second

// ActionHandler receives assigned actions. It must not block for long: in process it runs on
// the dispatcher's delivery goroutine and its error requeues the task; over gRPC it runs on
// the session's deliver loop and its error is reported as a retryable failure.
type ActionHandler interface {
	HandleAction(ctx context.Context, action *contracts.AssignedAction) error
}

// Operator is what a hosted operator implements. The host opens the session with the operator
// as its handler and calls Start once; the operator keeps the session for the rest of its
// life. Drain stops new work and waits for in-flight work, bounded by ctx; the host pauses the
// worker before Drain and closes the session after it.
type Operator interface {
	ActionHandler

	Start(ctx context.Context, s Session) error
	Drain(ctx context.Context)
}

type DAGStepTriggerRequest struct {
	ParentTaskExternalId uuid.UUID
	InvocationCount      int32
	WorkflowName         string
	// WorkflowVersionId pins triggering to the DAG's original version.
	WorkflowVersionId   uuid.UUID
	ActionId            string
	ChildIndex          int32
	Input               string
	AdditionalMetadata  []byte
	DagParentTaskRunIds []uuid.UUID
	IsSkipped           bool
	IsCancelled         bool
	DesiredWorkerLabels []*sqlcv1.GetDesiredLabelsRow

	// ParentReExecuted forces the step to re-run during a replay when any of its parents
	// re-executed this invocation.
	ParentReExecuted bool
}

type DAGStepTriggerResult struct {
	NodeId                int64
	BranchId              int64
	WorkflowRunExternalId uuid.UUID

	IsSatisfied   bool
	ResultPayload []byte
	IsFailure     bool
	ErrorMessage  *string

	// ReExecuted is true when the step actually runs this invocation rather than being
	// satisfied from the log.
	ReExecuted bool
}

// TaskEventWriter is the engine-internal surface for engine-internal operators (the DAG
// operator): calls that only exist inside the engine and are never available over gRPC. The
// dispatcher implements it. Everything an operator needs that both hosts offer (events,
// durable invocations, the action set) is on Session instead. Every call names its tenant
// explicitly: nothing here reads the tenant the gRPC auth middleware puts on a request context.
type TaskEventWriter interface {
	// CancelTaskEventCustom reports a cancelled task with a custom cancellation reason. It is
	// the engine-internal writer behind SendCancelledWithMessage, distinct from the CANCELLED
	// step action event every host offers through Session.SendStepActionEvent, and it is not
	// on the gRPC surface: an out-of-process operator has no equivalent.
	CancelTaskEventCustom(ctx context.Context, tenantId uuid.UUID, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)

	TriggerDAGStep(ctx context.Context, tenantId uuid.UUID, req *DAGStepTriggerRequest) (*DAGStepTriggerResult, error)

	// CancelDAGChildren cancels already-triggered children when the orchestrator is cancelled.
	CancelDAGChildren(ctx context.Context, tenantId uuid.UUID, taskExternalIds []uuid.UUID) error
}

// SharedOperator is the state an engine-internal operator shares: its config, the session the
// host opened for it, the engine-internal writer, and the bookkeeping for in-flight work. The
// session arrives with Start; the event senders, the action set and the durable channels go
// through it, so the operator's lifecycle is the host's.
type SharedOperator[T any] struct {
	operatorConfig  T
	taskEventWriter TaskEventWriter
	l               *zerolog.Logger
	tasks           sync.WaitGroup
	mu              sync.Mutex
	session         Session
	operatorId      uuid.UUID
	tenantId        uuid.UUID
	shutdown        bool

	inFlight map[string]context.CancelFunc

	// lastActions is the action set the operator last advertised, so UpdateWorkerActions
	// sends only the difference. Only the goroutine that refreshes actions touches it.
	lastActions map[string]struct{}
}

// NewSharedOperator constructs the shared operator state from the operator row.
func NewSharedOperator[T any](operator *sqlcv1.V1Operator, l *zerolog.Logger, taskEventWriter TaskEventWriter, t T) (*SharedOperator[T], error) {
	err := json.Unmarshal(operator.Config, &t)

	if err != nil {
		return nil, err
	}

	return &SharedOperator[T]{
		operatorConfig:  t,
		l:               l,
		taskEventWriter: taskEventWriter,
		operatorId:      operator.ID,
		tenantId:        operator.TenantID,
		lastActions:     map[string]struct{}{},
	}, nil
}

// Start records the session the host opened. Operators that embed SharedOperator call it from
// their own Start.
func (s *SharedOperator[T]) Start(_ context.Context, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.session = session

	return nil
}

// Session is the session the host opened, or nil before Start.
func (s *SharedOperator[T]) Session() Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.session
}

func (s *SharedOperator[T]) Config() T {
	return s.operatorConfig
}

func (s *SharedOperator[T]) Logger() *zerolog.Logger {
	return s.l
}

// WorkerId is the worker the session backs, or uuid.Nil before Start.
func (s *SharedOperator[T]) WorkerId() uuid.UUID {
	session := s.Session()

	if session == nil {
		return uuid.Nil
	}

	return session.Registration().WorkerId
}

func (s *SharedOperator[T]) TenantId() uuid.UUID {
	return s.tenantId
}

func (s *SharedOperator[T]) OperatorId() uuid.UUID {
	return s.operatorId
}

// UpdateWorkerActions makes actions the worker's action set: the difference from the set last
// advertised goes to the session as adds and removes, followed by a flush. It reports whether
// anything changed. A delta that fails leaves the advertised set as it was, so the next call
// repeats it; ids the engine already has are ignored by it.
func (s *SharedOperator[T]) UpdateWorkerActions(ctx context.Context, actions []string) (bool, error) {
	session := s.Session()

	if session == nil {
		return false, fmt.Errorf("operator has no session yet")
	}

	desired := make(map[string]struct{}, len(actions))
	add := make([]string, 0)

	for _, id := range actions {
		if _, ok := desired[id]; ok {
			continue
		}

		desired[id] = struct{}{}

		if _, ok := s.lastActions[id]; !ok {
			add = append(add, id)
		}
	}

	remove := make([]string, 0)

	for id := range s.lastActions {
		if _, ok := desired[id]; !ok {
			remove = append(remove, id)
		}
	}

	if len(add) == 0 && len(remove) == 0 {
		return false, nil
	}

	if len(add) > 0 {
		if err := session.AddActions(ctx, add); err != nil {
			return false, err
		}
	}

	if len(remove) > 0 {
		if err := session.RemoveActions(ctx, remove); err != nil {
			return false, err
		}
	}

	if err := session.Flush(ctx); err != nil {
		return false, err
	}

	s.lastActions = desired

	return true, nil
}

func (s *SharedOperator[T]) TriggerDAGStep(ctx context.Context, req *DAGStepTriggerRequest) (*DAGStepTriggerResult, error) {
	if s.taskEventWriter == nil {
		return nil, fmt.Errorf("operator has no task event writer configured")
	}

	return s.taskEventWriter.TriggerDAGStep(ctx, s.tenantId, req)
}

func (s *SharedOperator[T]) CancelDAGChildren(ctx context.Context, taskExternalIds []uuid.UUID) error {
	if s.taskEventWriter == nil {
		return fmt.Errorf("operator has no task event writer configured")
	}

	return s.taskEventWriter.CancelDAGChildren(ctx, s.tenantId, taskExternalIds)
}

// OpenDurable opens one durable invocation's pipe through the session: the host does the
// register-worker handshake and holds what the engine sends before its ack, so the operator
// only ever reads invocation traffic, an entry never ahead of the ack that names it.
func (s *SharedOperator[T]) OpenDurable(ctx context.Context, taskExternalId uuid.UUID, invocation int32) (DurableChannel, error) {
	session := s.Session()

	if session == nil {
		return nil, fmt.Errorf("operator has no session yet")
	}

	return session.OpenDurable(ctx, taskExternalId, invocation)
}

// SendStarted reports that the operator has started processing the assigned action.
func (s *SharedOperator[T]) SendStarted(action *contracts.AssignedAction) error {
	return s.SendStartedAt(action, time.Now())
}

func (s *SharedOperator[T]) SendStartedAt(action *contracts.AssignedAction, at time.Time) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, "", nil, at)
}

// SendCompleted reports a successful result. output should be the task's JSON output.
func (s *SharedOperator[T]) SendCompleted(action *contracts.AssignedAction, output []byte) error {
	s.mu.Lock()
	delete(s.inFlight, action.TaskRunExternalId)
	s.mu.Unlock()

	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, string(output), nil)
}

// SendCancelled reports a cancelled task
func (s *SharedOperator[T]) SendCancelled(action *contracts.AssignedAction) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, "cancelled", nil)
}

// SendFailed reports a failure with the given error message. shouldNotRetry, when true,
// prevents the task from being retried.
func (s *SharedOperator[T]) SendFailed(action *contracts.AssignedAction, errMsg string, shouldNotRetry bool) error {
	return s.sendStepActionEvent(action, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, errMsg, &shouldNotRetry)
}

// SendCancelledWithMessage reports a cancelled task with a custom cancellation reason through
// the engine-internal writer rather than the generic step action event path, so the reason
// reaches the run's events verbatim. There is no such RPC: an operator hosted over gRPC
// reports a plain CANCELLED event instead.
func (s *SharedOperator[T]) SendCancelledWithMessage(action *contracts.AssignedAction, msg string) error {
	if s.taskEventWriter == nil {
		return fmt.Errorf("operator has no task event writer configured")
	}

	event := s.buildEvent(action, msg, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)
	defer cancel()

	_, err := s.taskEventWriter.CancelTaskEventCustom(ctx, s.tenantId, event)
	return err
}

// buildEvent is the StepActionEvent the engine expects for action, with the worker filled by
// the session.
func (s *SharedOperator[T]) buildEvent(action *contracts.AssignedAction, payload string, ts time.Time) *contracts.StepActionEvent {
	retryCount := action.RetryCount

	return &contracts.StepActionEvent{
		WorkerId:          s.WorkerId().String(),
		JobId:             action.JobId,
		JobRunId:          action.JobRunId,
		TaskId:            action.TaskId,
		TaskRunExternalId: action.TaskRunExternalId,
		ActionId:          action.ActionId,
		EventTimestamp:    timestamppb.New(ts),
		EventPayload:      payload,
		RetryCount:        &retryCount,
	}
}

// sendStepActionEvent builds a StepActionEvent from the assigned action and reports it through
// the session. It uses a detached, time-bounded context (the caller's request context may
// already be cancelled by the time we report).
func (s *SharedOperator[T]) sendStepActionEvent(action *contracts.AssignedAction, eventType contracts.StepActionEventType, payload string, shouldNotRetry *bool, eventTS ...time.Time) error {
	session := s.Session()

	if session == nil {
		return fmt.Errorf("operator has no session yet")
	}

	ts := time.Now()
	if len(eventTS) > 0 {
		ts = eventTS[0]
	}

	event := s.buildEvent(action, payload, ts)
	event.EventType = eventType
	event.ShouldNotRetry = shouldNotRetry

	ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)
	defer cancel()

	return session.SendStepActionEvent(ctx, event)
}

// RecordTask registers an in-flight task and returns a release function that the caller
// must invoke (typically via defer) when the task finishes. Drain blocks until every
// recorded task has been released.
//
// If the operator is already draining, the returned release is a no-op and the task is not
// tracked: callers should avoid starting new work once Drain has begun, but in-flight work
// recorded before it is always awaited.
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

func (s *SharedOperator[T]) RegisterCancellableContext(ctx context.Context, taskRunExternalId string) (context.Context, func()) {
	cctx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	if s.inFlight == nil {
		s.inFlight = make(map[string]context.CancelFunc)
	}
	s.inFlight[taskRunExternalId] = cancel
	s.mu.Unlock()

	return cctx, func() {
		s.mu.Lock()
		delete(s.inFlight, taskRunExternalId)
		s.mu.Unlock()

		cancel()
	}
}

func (s *SharedOperator[T]) CancelTask(taskRunExternalId string) bool {
	s.mu.Lock()
	cancel, ok := s.inFlight[taskRunExternalId]

	if ok {
		delete(s.inFlight, taskRunExternalId)
	}

	s.mu.Unlock()

	if ok {
		cancel()
	}

	// if we didn't find it in the map, that means the task has either completed, or already been cancelled
	return ok
}

// Drain stops accepting new tracked tasks and waits for the in-flight ones, or for ctx. The
// host pauses the worker before calling it, so nothing new arrives while it waits.
func (s *SharedOperator[T]) Drain(ctx context.Context) {
	s.beginShutdown()

	done := make(chan struct{})

	go func() {
		s.tasks.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		if s.l != nil {
			s.l.Warn().Ctx(ctx).Msg("operator drain ended before every in-flight task finished")
		}
	}
}

// beginShutdown stops accepting new tracked tasks. Setting the flag under the mutex (paired
// with the Add in RecordTask) guarantees no WaitGroup.Add races with the Wait in Drain.
func (s *SharedOperator[T]) beginShutdown() {
	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()
}
