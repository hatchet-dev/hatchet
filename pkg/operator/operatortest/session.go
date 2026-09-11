package operatortest

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// Session is a recording operator.Session for tests of operators and hosts' callers: events,
// deltas, flushes, pauses and the close are recorded in order; durable channels and workflows
// are not supported.
type Session struct {
	mu  sync.Mutex
	reg operator.Registration

	events  []*contracts.StepActionEvent
	added   [][]string
	removed [][]string
	flushes int
	ops     []string
	closed  bool

	// PauseErr, when set, fails Pause.
	PauseErr error
}

// NewSession builds a session registered for the given tenant, operator and a fresh worker.
func NewSession(tenantId, operatorId uuid.UUID) *Session {
	return &Session{reg: operator.Registration{TenantId: tenantId, OperatorId: operatorId, WorkerId: uuid.New()}}
}

func (s *Session) Registration() operator.Registration {
	return s.reg
}

func (s *Session) AddActions(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.added = append(s.added, append([]string(nil), ids...))

	return nil
}

func (s *Session) RemoveActions(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.removed = append(s.removed, append([]string(nil), ids...))

	return nil
}

func (s *Session) Flush(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.flushes++

	return nil
}

func (s *Session) PutWorkflow(context.Context, *v1.CreateWorkflowVersionRequest) ([]string, error) {
	return nil, operator.ErrNotSupported
}

func (s *Session) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, ev)

	return nil
}

func (s *Session) OpenDurable(context.Context, uuid.UUID, int32) (operator.DurableChannel, error) {
	return nil, operator.ErrNotSupported
}

func (s *Session) Pause(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.PauseErr != nil {
		return s.PauseErr
	}

	s.ops = append(s.ops, "pause")

	return nil
}

func (s *Session) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.ops = append(s.ops, "close")

	return nil
}

// Events returns the step action events reported so far, in order.
func (s *Session) Events() []*contracts.StepActionEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]*contracts.StepActionEvent(nil), s.events...)
}

// Added returns the add deltas issued so far, in order.
func (s *Session) Added() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][]string(nil), s.added...)
}

// Removed returns the remove deltas issued so far, in order.
func (s *Session) Removed() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([][]string(nil), s.removed...)
}

// Flushes is the number of flushes so far.
func (s *Session) Flushes() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.flushes
}

// Ops returns the lifecycle calls (pause, close) in order.
func (s *Session) Ops() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.ops...)
}

// Closed reports whether Close was called.
func (s *Session) Closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}
