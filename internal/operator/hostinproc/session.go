package hostinproc

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// session is one in-process session: a thin adapter from the contract to the engine session.
// Deltas are applied synchronously, in chunks the engine accepts, so Flush has nothing to wait
// for.
type session struct {
	host *Host
	ss   *operatorsvc.Session
	reg  operator.Registration

	mu     sync.Mutex
	closed bool
}

func (s *session) Registration() operator.Registration {
	return s.reg
}

func (s *session) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.closed
}

// AddActions implements operator.Session; the delta is committed when it returns.
func (s *session) AddActions(ctx context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.applyDelta(ctx, ids, nil)
}

// RemoveActions implements operator.Session; the delta is committed when it returns.
func (s *session) RemoveActions(ctx context.Context, ids []string) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.applyDelta(ctx, nil, ids)
}

// applyDelta applies adds then removes, each in chunks of at most MaxActionsPerDelta ids, which
// is the most one engine delta carries. A chunk that fails stops the delta there; the ids
// before it are committed, which is what the caller's next call repeats harmlessly.
func (s *session) applyDelta(ctx context.Context, add, remove []string) error {
	for _, chunk := range chunks(add, operatorsvc.MaxActionsPerDelta) {
		if _, err := s.ss.ApplyDelta(ctx, chunk, nil); err != nil {
			return err
		}
	}

	for _, chunk := range chunks(remove, operatorsvc.MaxActionsPerDelta) {
		if _, err := s.ss.ApplyDelta(ctx, nil, chunk); err != nil {
			return err
		}
	}

	return nil
}

func chunks(ids []string, size int) [][]string {
	if len(ids) == 0 {
		return nil
	}

	out := make([][]string, 0, (len(ids)+size-1)/size)

	for start := 0; start < len(ids); start += size {
		end := min(start+size, len(ids))
		out = append(out, ids[start:end])
	}

	return out
}

// Flush implements operator.Session. Deltas are synchronous here, so there is nothing to wait
// for.
func (s *session) Flush(context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return nil
}

// PutWorkflow implements operator.Session over the admin service with the tenant on the
// context. The action ids come from the steps the engine stored for the new version, which are
// the normalized ids the scheduler matches on; the DAG orchestrator step the engine adds to a
// DAG workflow is not an action of the caller's and is left out.
func (s *session) PutWorkflow(ctx context.Context, wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if s.isClosed() {
		return nil, operator.ErrSessionClosed
	}

	if s.host.workflows == nil {
		return nil, fmt.Errorf("hostinproc: put workflow: %w", operator.ErrNotSupported)
	}

	tenant := s.ss.Tenant()

	resp, err := s.host.workflows.PutWorkflow(operatorsvc.WithTenant(ctx, tenant), wf)

	if err != nil {
		return nil, err
	}

	versionId, err := uuid.Parse(resp.GetId())

	if err != nil {
		return nil, fmt.Errorf("hostinproc: the engine returned workflow version id %q: %w", resp.GetId(), err)
	}

	steps, err := s.host.workflows.ListStepsByWorkflowVersionId(ctx, tenant.ID, versionId)

	if err != nil {
		return nil, fmt.Errorf("hostinproc: could not list the steps of workflow version %s: %w", versionId, err)
	}

	seen := make(map[string]struct{}, len(steps))
	actions := make([]string, 0, len(steps))

	for _, step := range steps {
		if step.IsDagOrchestrator || step.ActionId == "" {
			continue
		}

		if _, ok := seen[step.ActionId]; ok {
			continue
		}

		seen[step.ActionId] = struct{}{}
		actions = append(actions, step.ActionId)
	}

	return actions, nil
}

// SendStepActionEvent implements operator.Session on the tenant-forged context path.
func (s *session) SendStepActionEvent(ctx context.Context, ev *contracts.StepActionEvent) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.ss.SendStepActionEvent(ctx, ev)
}

// OpenDurable implements operator.Session: the engine session does the handshake and the hold.
func (s *session) OpenDurable(ctx context.Context, taskExternalId uuid.UUID, invocation int32) (operator.DurableChannel, error) {
	if s.isClosed() {
		return nil, operator.ErrSessionClosed
	}

	return s.ss.OpenDurable(ctx, taskExternalId, invocation)
}

// Pause implements operator.Session.
func (s *session) Pause(ctx context.Context) error {
	if s.isClosed() {
		return operator.ErrSessionClosed
	}

	return s.ss.Pause(ctx, true)
}

// Close implements operator.Session: pause (unless already paused), release, deactivate fenced
// on the session id, then leave the heartbeat set. Close runs once; later calls return nil.
func (s *session) Close(ctx context.Context) error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}

	s.closed = true
	s.mu.Unlock()

	err := s.ss.Close(ctx)

	s.host.forget(s)

	return err
}
