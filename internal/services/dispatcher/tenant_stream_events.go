package dispatcher

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
)

// eventConverter adapts a per-payload field mapping into a converter over raw
// tenant-stream payloads: each payload is decoded as T and mapped to a
// contracts.WorkflowEvent.
func eventConverter[T any](toEvent func(payload *T) *contracts.WorkflowEvent) func(payloads [][]byte) []*contracts.WorkflowEvent {
	return func(payloads [][]byte) []*contracts.WorkflowEvent {
		converted := msgqueue.JSONConvert[T](payloads)
		workflowEvents := []*contracts.WorkflowEvent{}

		for _, payload := range converted {
			workflowEvents = append(workflowEvents, toEvent(payload))
		}

		return workflowEvents
	}
}

// workflowEventConverters maps each message ID the dispatcher's workflow event
// streams consume off the tenant stream to its contracts.WorkflowEvent
// converter. Together with workflowRunMatchers, its keys are the source of
// truth for the message IDs the streams consume: msgqueue's tenant-stream
// publish allowlist is asserted against them in TestTenantStreamMsgIDsInSync.
var workflowEventConverters = map[string]func(payloads [][]byte) []*contracts.WorkflowEvent{
	msgqueue.MsgIDCreatedTask: eventConverter(func(payload *tasktypes.CreatedTaskPayload) *contracts.WorkflowEvent {
		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.WorkflowRunID.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
			ResourceId:     payload.ExternalID.String(),
			EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED,
			EventTimestamp: timestamppb.New(payload.InsertedAt.Time),
			RetryCount:     &payload.RetryCount,
		}
	}),
	msgqueue.MsgIDTaskCompleted: eventConverter(func(payload *tasktypes.CompletedTaskPayload) *contracts.WorkflowEvent {
		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.WorkflowRunId.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
			ResourceId:     payload.ExternalId.String(),
			EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
			EventTimestamp: timestamppb.New(time.Now()),
			RetryCount:     &payload.RetryCount,
			EventPayload:   string(payload.Output),
		}
	}),
	msgqueue.MsgIDTaskFailed: eventConverter(func(payload *tasktypes.FailedTaskPayload) *contracts.WorkflowEvent {
		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.WorkflowRunId.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
			ResourceId:     payload.ExternalId.String(),
			EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED,
			EventTimestamp: timestamppb.New(time.Now()),
			RetryCount:     &payload.RetryCount,
			EventPayload:   payload.ErrorMsg,
		}
	}),
	msgqueue.MsgIDTaskCancelled: eventConverter(func(payload *tasktypes.CancelledTaskPayload) *contracts.WorkflowEvent {
		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.WorkflowRunId.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
			ResourceId:     payload.ExternalId.String(),
			EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED,
			EventTimestamp: timestamppb.New(time.Now()),
			RetryCount:     &payload.RetryCount,
		}
	}),
	msgqueue.MsgIDTaskStreamEvent: eventConverter(func(payload *tasktypes.StreamEventPayload) *contracts.WorkflowEvent {
		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.WorkflowRunId.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
			ResourceId:     payload.TaskRunId.String(),
			EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
			EventTimestamp: timestamppb.New(payload.CreatedAt),
			EventPayload:   string(payload.Payload),
			EventIndex:     payload.EventIndex,
		}
	}),
	msgqueue.MsgIDWorkflowRunFinished: eventConverter(func(payload *tasktypes.NotifyFinalizedPayload) *contracts.WorkflowEvent {
		eventType := contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED

		switch payload.Status {
		case sqlcv1.V1ReadableStatusOlapCANCELLED:
			eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED
		case sqlcv1.V1ReadableStatusOlapFAILED:
			eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED
		case sqlcv1.V1ReadableStatusOlapCOMPLETED:
			eventType = contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED
		}

		return &contracts.WorkflowEvent{
			WorkflowRunId:  payload.ExternalId.String(),
			ResourceType:   contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN,
			ResourceId:     payload.ExternalId.String(),
			EventType:      eventType,
			EventTimestamp: timestamppb.New(time.Now()),
		}
	}),
}

func msgsToWorkflowEvent(msgId string, payloads [][]byte, filter func(tasks []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error), hangupFunc func(tasks []*contracts.WorkflowEvent) ([]*contracts.WorkflowEvent, error)) ([]*contracts.WorkflowEvent, error) {
	workflowEvents := []*contracts.WorkflowEvent{}

	if convert, ok := workflowEventConverters[msgId]; ok {
		workflowEvents = convert(payloads)
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

// runMatcher adapts a per-payload run ID extractor into a matcher over raw
// tenant-stream payloads: each payload is decoded as T, and the run IDs the
// given acks are subscribed to are collected.
func runMatcher[T any](runID func(payload *T) uuid.UUID) func(payloads [][]byte, acks *workflowRunAcks) []uuid.UUID {
	return func(payloads [][]byte, acks *workflowRunAcks) []uuid.UUID {
		converted := msgqueue.JSONConvert[T](payloads)
		res := make([]uuid.UUID, 0)

		for _, payload := range converted {
			if id := runID(payload); acks.hasWorkflowRun(id) {
				res = append(res, id)
			}
		}

		return res
	}
}

// workflowRunMatchers maps each message ID the dispatcher's workflow run
// subscriptions consume off the tenant stream to a matcher returning the
// subscribed workflow run IDs the message finalizes (or nominates as
// candidates). See the comment on workflowEventConverters: the keys of both
// maps together are asserted against msgqueue's tenant-stream publish
// allowlist in TestTenantStreamMsgIDsInSync.
var workflowRunMatchers = map[string]func(payloads [][]byte, acks *workflowRunAcks) []uuid.UUID{
	msgqueue.MsgIDWorkflowRunFinished: runMatcher(func(payload *tasktypes.NotifyFinalizedPayload) uuid.UUID {
		return payload.ExternalId
	}),
	msgqueue.MsgIDWorkflowRunFinishedCandidate: runMatcher(func(payload *tasktypes.CandidateFinalizedPayload) uuid.UUID {
		return payload.WorkflowRunId
	}),
}

func isMatchingWorkflowRunV1(msg *msgqueue.Message, acks *workflowRunAcks) ([]uuid.UUID, bool) {
	match, ok := workflowRunMatchers[msg.ID]

	if !ok {
		return nil, false
	}

	res := match(msg.Payloads, acks)

	if len(res) == 0 {
		return nil, false
	}

	return res, true
}
