// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// ListenerStrategy selects the worker action listen RPC variant.
type ListenerStrategy string

const (
	// ListenerStrategyV1 uses the legacy Listen RPC.
	ListenerStrategyV1 ListenerStrategy = "v1"
	// ListenerStrategyV2 uses the ListenV2 RPC.
	ListenerStrategyV2 ListenerStrategy = "v2"
)

type actionListenerImpl struct {
	client       dispatchercontracts.DispatcherClient
	listenClient dispatchercontracts.Dispatcher_ListenClient
	v            validator.Validator
	actionStream *reconnectingStream[dispatchercontracts.Dispatcher_ListenClient]
	l            *zerolog.Logger
	ctx          *contextLoader
	tenantId     string
	workerId     string

	// listenerStrategy is read and written only from the single action-loop
	// goroutine — classifyActionError and connectOnce both execute there.
	listenerStrategy ListenerStrategy

	actionStreamOnce sync.Once
}

func (a *actionListenerImpl) Actions(ctx context.Context) (<-chan *Action, <-chan error, error) {
	ch := make(chan *Action)
	errCh := make(chan error, 1)

	a.l.Debug().Ctx(ctx).Msg("starting action listener")

	go a.heartbeatLoop(ctx)
	go a.actionLoop(ctx, ch, errCh)

	return ch, errCh, nil
}

func (a *actionListenerImpl) heartbeatLoop(ctx context.Context) {
	heartbeatInterval := 4 * time.Second
	timer := time.NewTicker(100 * time.Millisecond)
	defer timer.Stop()

	lastHeartbeat := time.Now().Add(-5 * time.Second)
	firstHeartbeat := true

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			now := time.Now().UTC()
			if lastHeartbeat.Add(heartbeatInterval).After(now) {
				continue
			}

			a.l.Debug().Ctx(ctx).Str("worker_id", a.workerId).Msg("updating worker heartbeat")

			_, err := a.client.Heartbeat(a.ctx.newContext(ctx), &dispatchercontracts.HeartbeatRequest{
				WorkerId:    a.workerId,
				HeartbeatAt: timestamppb.New(now),
			})

			if err != nil {
				a.l.Error().Ctx(ctx).Err(err).Str("worker_id", a.workerId).Msg("could not update worker heartbeat")

				if status.Code(err) == codes.Unimplemented {
					return
				}
			}

			if !firstHeartbeat {
				actualInterval := now.Sub(lastHeartbeat)
				if actualInterval > heartbeatInterval+1*time.Second {
					a.l.Warn().Ctx(ctx).
						Str("worker_id", a.workerId).
						Dur("actual_interval", actualInterval.Round(time.Millisecond)).
						Dur("expected_interval", heartbeatInterval+1*time.Second).
						Msg("worker heartbeat interval delay, possible CPU resource contention")
				}
			}

			firstHeartbeat = false
			lastHeartbeat = now
		}
	}
}

func (a *actionListenerImpl) actionLoop(ctx context.Context, ch chan<- *Action, errCh chan<- error) {
	defer close(ch)
	defer close(errCh)
	defer func() {
		if err := a.actionStreamCore().Close(); err != nil {
			a.l.Error().Ctx(ctx).Err(err).Msg("failed to close action listener stream")
		}
	}()

	classify := a.classifyActionError(newStreamClassifier(func(ctx context.Context) bool {
		return ctx.Err() == nil
	}))

	err := listenStream(ctx, a.actionStreamCore(),
		func(c dispatchercontracts.Dispatcher_ListenClient) (*dispatchercontracts.AssignedAction, error) {
			return c.Recv()
		},
		func(assigned *dispatchercontracts.AssignedAction) error {
			action, ok := a.actionFromAssigned(ctx, assigned)
			if !ok {
				return nil
			}
			select {
			case ch <- action:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		classify,
	)
	if err != nil && ctx.Err() == nil {
		sendListenerError(ctx, errCh, err)
	}
}

func (a *actionListenerImpl) classifyActionError(base streamClassifier) streamClassifier {
	return func(ctx context.Context, err error) streamVerdict {
		if a.listenerStrategy == ListenerStrategyV2 && status.Code(err) == codes.Unimplemented {
			a.l.Debug().Ctx(ctx).Msg("falling back to v1 listener strategy")
			a.listenerStrategy = ListenerStrategyV1
			return verdictNoProgress
		}
		return base(ctx, err)
	}
}

func (a *actionListenerImpl) actionStreamCore() *reconnectingStream[dispatchercontracts.Dispatcher_ListenClient] {
	a.actionStreamOnce.Do(func() {
		wl := a.l.With().Str("worker_id", a.workerId).Logger()
		a.actionStream = newReconnectingStream(
			&wl,
			"action listener",
			a.subscribeActionStream,
			func(client dispatchercontracts.Dispatcher_ListenClient) error {
				return client.CloseSend()
			},
			nil,
		)

		if a.listenClient != nil {
			a.actionStream.setInitialClient(a.listenClient)
		}
	})

	return a.actionStream
}

func (a *actionListenerImpl) subscribeActionStream(ctx context.Context) (dispatchercontracts.Dispatcher_ListenClient, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	var listenClient dispatchercontracts.Dispatcher_ListenClient
	var err error

	switch a.listenerStrategy {
	case ListenerStrategyV1:
		listenClient, err = a.client.Listen(a.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
			WorkerId: a.workerId,
		}, grpc_retry.Disable())
	case ListenerStrategyV2:
		listenClient, err = a.client.ListenV2(a.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
			WorkerId: a.workerId,
		}, grpc_retry.Disable())
	default:
		return nil, fmt.Errorf("unknown listener strategy %s", a.listenerStrategy)
	}

	if err != nil {
		return nil, err
	}

	return listenClient, nil
}

func (a *actionListenerImpl) actionFromAssigned(ctx context.Context, assignedAction *dispatchercontracts.AssignedAction) (*Action, bool) {
	var actionType ActionType

	switch assignedAction.ActionType {
	case dispatchercontracts.ActionType_START_STEP_RUN:
		actionType = ActionTypeStartStepRun
	case dispatchercontracts.ActionType_CANCEL_STEP_RUN:
		actionType = ActionTypeCancelStepRun
	case dispatchercontracts.ActionType_START_GET_GROUP_KEY:
		actionType = ActionTypeStartGetGroupKey
	case dispatchercontracts.ActionType_START_BATCH:
		actionType = ActionTypeStartBatch
	default:
		a.l.Error().Ctx(ctx).Str("action_type", string(assignedAction.ActionType)).Msg("unknown action type")
		return nil, false
	}

	a.l.Debug().Ctx(ctx).Str("action_type", string(actionType)).Str("action_id", assignedAction.ActionId).Msg("received action")

	additionalMetadata, ok := a.parseAdditionalMetadata(ctx, assignedAction)
	if !ok {
		return nil, false
	}
	act := &Action{
		TenantId:                   assignedAction.TenantId,
		WorkflowRunId:              assignedAction.WorkflowRunId,
		GetGroupKeyRunId:           assignedAction.GetGroupKeyRunId,
		WorkerId:                   a.workerId,
		JobId:                      assignedAction.JobId,
		JobName:                    assignedAction.JobName,
		JobRunId:                   assignedAction.JobRunId,
		StepId:                     assignedAction.TaskId,
		StepName:                   assignedAction.TaskName,
		StepRunId:                  assignedAction.TaskRunExternalId,
		ActionId:                   assignedAction.ActionId,
		ActionType:                 actionType,
		ActionPayload:              []byte(assignedAction.ActionPayload),
		RetryCount:                 assignedAction.RetryCount,
		AdditionalMetadata:         additionalMetadata,
		ChildIndex:                 assignedAction.ChildWorkflowIndex,
		ChildKey:                   assignedAction.ChildWorkflowKey,
		ParentWorkflowRunId:        assignedAction.ParentWorkflowRunId,
		Priority:                   assignedAction.Priority,
		WorkflowId:                 assignedAction.WorkflowId,
		WorkflowVersionId:          assignedAction.WorkflowVersionId,
		TriggeringEventExternalId:  assignedAction.TriggeringEventExternalId,
		TriggeringEventKey:         assignedAction.TriggeringEventKey,
		DurableTaskInvocationCount: assignedAction.DurableTaskInvocationCount,
		BatchId:                    assignedAction.BatchId,
		BatchIndex:                 assignedAction.BatchIndex,
		BatchKey:                   assignedAction.BatchKey,
	}
	if assignedAction.BatchStartPayload != nil {
		bs := assignedAction.BatchStartPayload
		batchStart := &BatchStart{
			ExpectedSize: bs.GetExpectedSize(),
		}
		if bs.TriggerReason != "" {
			batchStart.TriggerReason = bs.TriggerReason
		}
		if bs.TriggerTime != nil {
			batchStart.TriggerTime = bs.TriggerTime.AsTime()
		}
		act.BatchStart = batchStart
	}
	return act, true
}

func (a *actionListenerImpl) parseAdditionalMetadata(ctx context.Context, assignedAction *dispatchercontracts.AssignedAction) (map[string]string, bool) {
	if assignedAction.AdditionalMetadata == nil {
		return nil, true
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(*assignedAction.AdditionalMetadata), &rawMap); err != nil {
		a.l.Error().Ctx(ctx).Err(err).Msg("could not unmarshal additional metadata")
		return nil, false
	}

	additionalMetadata := make(map[string]string)
	for k, v := range rawMap {
		if strVal, ok := v.(string); ok {
			additionalMetadata[k] = strVal
		}
	}

	return additionalMetadata, true
}

func streamErrorCode(err error) string {
	if err == nil {
		return ""
	}

	if st, ok := status.FromError(err); ok {
		return st.Code().String()
	}

	return "unknown"
}

func (a *actionListenerImpl) Unregister() error {
	_, err := a.client.Unsubscribe(
		a.ctx.newContext(context.Background()),
		&dispatchercontracts.WorkerUnsubscribeRequest{
			WorkerId: a.workerId,
		},
	)

	if err != nil {
		return err
	}

	return nil
}
