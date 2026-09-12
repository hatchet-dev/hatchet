package serverlessoperator

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

// eventReportTimeout bounds one step event report. Reports use a detached context, like the
// SDK and SharedOperator do, so a cancelled or timed-out delivery still reports its outcome.
const eventReportTimeout = 30 * time.Second

// eventSender builds the same StepActionEvents as pkg/operator.SharedOperator and sends them
// through a session.
type eventSender struct {
	session operator.Session
}

func newStepEvent(workerId string, action *contracts.AssignedAction, eventType contracts.StepActionEventType, payload string, shouldNotRetry *bool) *contracts.StepActionEvent {
	retryCount := action.RetryCount

	return &contracts.StepActionEvent{
		WorkerId:          workerId,
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
}

func (s *eventSender) send(action *contracts.AssignedAction, eventType contracts.StepActionEventType, payload string, shouldNotRetry *bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), eventReportTimeout)
	defer cancel()

	workerId := s.session.Registration().WorkerId.String()

	return s.session.SendStepActionEvent(ctx, newStepEvent(workerId, action, eventType, payload, shouldNotRetry))
}

func (s *eventSender) started(action *contracts.AssignedAction) error {
	return s.send(action, contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, "", nil)
}

func (s *eventSender) completed(action *contracts.AssignedAction, output []byte) error {
	return s.send(action, contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED, string(output), nil)
}

func (s *eventSender) failed(action *contracts.AssignedAction, errMsg string, retry bool) error {
	shouldNotRetry := !retry
	return s.send(action, contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED, errMsg, &shouldNotRetry)
}

func (s *eventSender) cancelled(action *contracts.AssignedAction) error {
	return s.send(action, contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED, "cancelled", nil)
}

// report sends the terminal event for an outcome.
func (s *eventSender) report(action *contracts.AssignedAction, out outcome) error {
	switch out.status {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.completed(action, out.output)
	case contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED:
		return s.cancelled(action)
	default:
		return s.failed(action, out.errMsg, out.retry)
	}
}
