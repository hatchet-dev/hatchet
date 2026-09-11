// Package operatortest holds a sample contract operator for tests. Echo is written against
// pkg/operator only, so the same handler is opened through the in-process host in unit tests
// and through the gRPC host in the operator end-to-end suite; that both work is the rule the
// contract exists for.
package operatortest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// reportTimeout bounds one event report; the action's own context may be gone by then.
const reportTimeout = 30 * time.Second

// Echo completes every START_STEP_RUN it is handed with the task's input as its output. It
// reports STARTED at once and COMPLETED from a goroutine, the way an operator whose work takes
// time would, and tracks the goroutines so Drain can wait for them.
type Echo struct {
	mu      sync.Mutex
	session operator.Session
	tasks   sync.WaitGroup

	// Handled counts the start actions received, for tests.
	handled int
}

// Start implements operator.Operator: the session is what the echo reports through.
func (e *Echo) Start(_ context.Context, s operator.Session) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.session = s

	return nil
}

// HandleAction implements operator.ActionHandler.
func (e *Echo) HandleAction(_ context.Context, action *contracts.AssignedAction) error {
	if action.ActionType != contracts.ActionType_START_STEP_RUN {
		return nil
	}

	e.mu.Lock()
	s := e.session
	e.handled++
	e.mu.Unlock()

	if s == nil {
		return fmt.Errorf("echo operator received an action before it was started")
	}

	e.tasks.Add(1)

	go func() {
		defer e.tasks.Done()

		_ = e.report(s, action, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, "")

		output, err := echoOutput(action)

		if err != nil {
			_ = e.report(s, action, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, err.Error())
			return
		}

		_ = e.report(s, action, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, output)
	}()

	return nil
}

// Drain implements operator.Operator: it waits for every report in flight, or for ctx.
func (e *Echo) Drain(ctx context.Context) {
	done := make(chan struct{})

	go func() {
		e.tasks.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}
}

// Handled is the number of start actions received.
func (e *Echo) Handled() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.handled
}

func (e *Echo) report(s operator.Session, action *contracts.AssignedAction, eventType contracts.StepActionEventType, payload string) error {
	ctx, cancel := context.WithTimeout(context.Background(), reportTimeout)
	defer cancel()

	retryCount := action.RetryCount

	return s.SendStepActionEvent(ctx, &contracts.StepActionEvent{
		JobId:             action.JobId,
		JobRunId:          action.JobRunId,
		TaskId:            action.TaskId,
		TaskRunExternalId: action.TaskRunExternalId,
		ActionId:          action.ActionId,
		EventTimestamp:    timestamppb.Now(),
		EventType:         eventType,
		EventPayload:      payload,
		RetryCount:        &retryCount,
	})
}

// echoOutput is the task's input, as the engine wraps it in the action payload, or an empty
// object when the payload carries none.
func echoOutput(action *contracts.AssignedAction) (string, error) {
	var payload struct {
		Input json.RawMessage `json:"input"`
	}

	if action.ActionPayload == "" {
		return "{}", nil
	}

	if err := json.Unmarshal([]byte(action.ActionPayload), &payload); err != nil {
		return "", fmt.Errorf("echo: could not decode the action payload: %w", err)
	}

	if len(payload.Input) == 0 {
		return "{}", nil
	}

	return string(payload.Input), nil
}
