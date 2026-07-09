// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // Register gzip compression codec
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DispatcherClient interface {
	GetActionListener(ctx context.Context, req *GetActionListenerRequest) (WorkerActionListener, *string, error)

	// GetVersion calls the GetVersion RPC. Returns the engine semantic version string.
	// Old engines that do not implement this will return codes.Unimplemented.
	GetVersion(ctx context.Context) (string, error)

	SendStepActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error)

	SendGroupKeyActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error)

	ReleaseSlot(ctx context.Context, stepRunId string) error

	RefreshTimeout(ctx context.Context, stepRunId string, incrementTimeoutBy string) error

	UpsertWorkerLabels(ctx context.Context, workerId string, labels map[string]interface{}) error

	RegisterDurableEvent(ctx context.Context, req *sharedcontracts.RegisterDurableEventRequest) (*sharedcontracts.RegisterDurableEventResponse, error)

	DurableTaskStream(ctx context.Context) (sharedcontracts.V1Dispatcher_DurableTaskClient, error)
}

const (
	DefaultActionListenerRetryInterval = 5 * time.Second
	DefaultActionListenerRetryCount    = 5
)

type GetActionListenerRequest struct {
	WorkerName string
	Services   []string
	Actions    []string
	SlotConfig map[string]int32
	Labels     map[string]interface{}
	WebhookId  *string

	// LegacySlots, when non-nil, causes the registration to use the deprecated
	// `slots` proto field instead of `slot_config`. This is for backward
	// compatibility with engines that do not support multiple slot types.
	LegacySlots *int32
}

// ActionPayload unmarshals the action payload into the target. It also validates the resulting target.
type ActionPayload func(target interface{}) error

type ActionType string

const (
	ActionTypeStartStepRun     ActionType = "START_STEP_RUN"
	ActionTypeCancelStepRun    ActionType = "CANCEL_STEP_RUN"
	ActionTypeStartGetGroupKey ActionType = "START_GET_GROUP_KEY"
	ActionTypeStartBatch       ActionType = "START_BATCH"
)

// BatchStart carries the flush signal metadata for a batch task.
type BatchStart struct {
	TriggerTime   time.Time
	TriggerReason string
	ExpectedSize  int32
}

type Action struct {
	// the worker id
	WorkerId string `json:"workerId"`

	// the tenant id
	TenantId string `json:"tenantId"`

	// the workflow run id
	WorkflowRunId string `json:"workflowRunId"`

	// the get group key run id
	GetGroupKeyRunId string `json:"getGroupKeyRunId"`

	// the job id
	JobId string `json:"jobId"`

	// the job name
	JobName string `json:"jobName"`

	// the job run id
	JobRunId string `json:"jobRunId"`

	// the step id
	StepId string `json:"stepId"`

	// the step name
	StepName string `json:"stepName"`

	// the step run id
	StepRunId string `json:"stepRunId"`

	// the action id
	ActionId string `json:"actionId"`

	// the action payload
	ActionPayload []byte `json:"actionPayload"`

	// the action type
	ActionType ActionType `json:"actionType"`

	// the count of the retry attempt
	RetryCount int32 `json:"retryCount"`

	// the additional metadata for the workflow run
	AdditionalMetadata map[string]string

	// the child index for the workflow run
	ChildIndex *int32

	// the child key for the workflow run
	ChildKey *string

	// the parent workflow run id
	ParentWorkflowRunId *string

	Priority int32 `json:"priority,omitempty"`

	WorkflowId *string `json:"workflowId,omitempty"`

	WorkflowVersionId *string `json:"workflowVersionId,omitempty"`

	TriggeringEventExternalId *string `json:"triggeringEventExternalId,omitempty"`

	TriggeringEventKey *string `json:"triggeringEventKey,omitempty"`

	DurableTaskInvocationCount *int32 `json:"durableTaskInvocationCount,omitempty"`

	// Batch fields — set when this action is part of a batch task.
	BatchId    *string     `json:"batchId,omitempty"`
	BatchIndex *int32      `json:"batchIndex,omitempty"`
	BatchKey   *string     `json:"batchKey,omitempty"`
	BatchStart *BatchStart `json:"batchStart,omitempty"`
}

type WorkerActionListener interface {
	Actions(ctx context.Context) (<-chan *Action, <-chan error, error)

	Unregister() error
}

type ActionEventType string

const (
	ActionEventTypeUnknown   ActionEventType = "STEP_EVENT_TYPE_UNKNOWN"
	ActionEventTypeStarted   ActionEventType = "STEP_EVENT_TYPE_STARTED"
	ActionEventTypeCompleted ActionEventType = "STEP_EVENT_TYPE_COMPLETED"
	ActionEventTypeFailed    ActionEventType = "STEP_EVENT_TYPE_FAILED"
)

type ActionEvent struct {
	*Action

	// the event timestamp
	EventTimestamp *time.Time

	// the step event type
	EventType ActionEventType

	// The event payload. This must be JSON-compatible as it gets marshalled to a JSON string.
	EventPayload interface{}

	// If this is an error, whether to retry on failure
	ShouldNotRetry *bool
}

type ActionEventResponse struct {
	// the tenant id
	TenantId string

	// the id of the worker
	WorkerId string
}

type dispatcherClientImpl struct {
	client   dispatchercontracts.DispatcherClient
	clientv1 sharedcontracts.V1DispatcherClient

	tenantId string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	presetWorkerLabels map[string]string
}

func newDispatcher(conn *grpc.ClientConn, opts *sharedClientOpts, presetWorkerLabels map[string]string) DispatcherClient {
	return &dispatcherClientImpl{
		client:             dispatchercontracts.NewDispatcherClient(conn),
		clientv1:           sharedcontracts.NewV1DispatcherClient(conn),
		tenantId:           opts.tenantId,
		l:                  opts.l,
		v:                  opts.v,
		ctx:                opts.ctxLoader,
		presetWorkerLabels: presetWorkerLabels,
	}
}

func (d *dispatcherClientImpl) newActionListener(ctx context.Context, req *GetActionListenerRequest) (*actionListenerImpl, *string, error) {
	// validate the request
	if err := d.v.Validate(req); err != nil {
		return nil, nil, err
	}

	// Get OS information
	var goVersion string
	var hatchetVersion string

	// Get Go version
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		goVersion = buildInfo.GoVersion

		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/hatchet-dev/hatchet" {
				hatchetVersion = dep.Version
				break
			}
		}
	}

	os := runtime.GOOS

	registerReq := &dispatchercontracts.WorkerRegisterRequest{
		WorkerName: req.WorkerName,
		Actions:    req.Actions,
		Services:   req.Services,
		WebhookId:  req.WebhookId,
		Labels:     map[string]*dispatchercontracts.WorkerLabels{},
		RuntimeInfo: &dispatchercontracts.RuntimeInfo{
			Language:        dispatchercontracts.SDKS_GO.Enum(),
			LanguageVersion: &goVersion,
			Os:              &os,
			SdkVersion:      &hatchetVersion,
		},
	}

	registerReq.Labels = map[string]*dispatchercontracts.WorkerLabels{}

	if req.Labels != nil {
		registerReq.Labels = mapLabels(req.Labels)
	}

	if d.presetWorkerLabels != nil {
		for k, v := range d.presetWorkerLabels {
			label := dispatchercontracts.WorkerLabels{
				StrValue: &v,
			}

			registerReq.Labels[k] = &label
		}
	}

	if req.LegacySlots != nil {
		registerReq.Slots = req.LegacySlots
	} else if len(req.SlotConfig) > 0 {
		registerReq.SlotConfig = req.SlotConfig
	} else {
		return nil, nil, fmt.Errorf("slot config is required for worker registration")
	}

	// register the worker
	resp, err := d.client.Register(d.ctx.newContext(ctx), registerReq)

	if err != nil {
		return nil, nil, fmt.Errorf("could not register the worker: %w", err)
	}

	d.l.Debug().Ctx(ctx).Msgf("Registered worker with id: %s", resp.WorkerId)

	// subscribe to the worker
	listener, err := d.client.ListenV2(d.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
		WorkerId: resp.WorkerId,
	}, grpc_retry.Disable())

	if err != nil {
		return nil, nil, fmt.Errorf("could not subscribe to the worker: %w", err)
	}

	return &actionListenerImpl{
		client:           d.client,
		listenClient:     listener,
		workerId:         resp.WorkerId,
		l:                d.l,
		v:                d.v,
		tenantId:         d.tenantId,
		ctx:              d.ctx,
		listenerStrategy: ListenerStrategyV2,
	}, &resp.WorkerId, nil
}

func (d *dispatcherClientImpl) GetVersion(ctx context.Context) (string, error) {
	resp, err := d.client.GetVersion(d.ctx.newContext(ctx), &dispatchercontracts.GetVersionRequest{})
	if err != nil {
		return "", err
	}
	return resp.Version, nil
}

func (d *dispatcherClientImpl) GetActionListener(ctx context.Context, req *GetActionListenerRequest) (WorkerActionListener, *string, error) {
	return d.newActionListener(ctx, req)
}

func (d *dispatcherClientImpl) SendStepActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error) {
	// validate the request
	if err := d.v.Validate(in); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(in.EventPayload)

	if err != nil {
		return nil, err
	}

	var actionEventType dispatchercontracts.StepActionEventType

	switch in.EventType {
	case ActionEventTypeStarted:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED
	case ActionEventTypeCompleted:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED
	case ActionEventTypeFailed:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_FAILED
	default:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_UNKNOWN
	}

	resp, err := d.client.SendStepActionEvent(d.ctx.newContext(ctx), &dispatchercontracts.StepActionEvent{
		WorkerId:          in.WorkerId,
		JobId:             in.JobId,
		JobRunId:          in.JobRunId,
		TaskId:            in.StepId,
		TaskRunExternalId: in.StepRunId,
		ActionId:          in.ActionId,
		EventTimestamp:    timestamppb.New(*in.EventTimestamp),
		EventType:         actionEventType,
		EventPayload:      string(payloadBytes),
		RetryCount:        &in.RetryCount,
		ShouldNotRetry:    in.ShouldNotRetry,
	})

	if err != nil {
		return nil, err
	}

	return &ActionEventResponse{
		TenantId: resp.TenantId,
		WorkerId: resp.WorkerId,
	}, nil
}

func (d *dispatcherClientImpl) SendGroupKeyActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error) {
	// validate the request
	if err := d.v.Validate(in); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(in.EventPayload)

	if err != nil {
		return nil, err
	}

	var actionEventType dispatchercontracts.GroupKeyActionEventType

	switch in.EventType {
	case ActionEventTypeStarted:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_STARTED
	case ActionEventTypeCompleted:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_COMPLETED
	case ActionEventTypeFailed:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_FAILED
	default:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_UNKNOWN
	}

	resp, err := d.client.SendGroupKeyActionEvent(d.ctx.newContext(ctx), &dispatchercontracts.GroupKeyActionEvent{
		WorkerId:         in.WorkerId,
		WorkflowRunId:    in.WorkflowRunId,
		GetGroupKeyRunId: in.GetGroupKeyRunId,
		ActionId:         in.ActionId,
		EventTimestamp:   timestamppb.New(*in.EventTimestamp),
		EventType:        actionEventType,
		EventPayload:     string(payloadBytes),
	})

	if err != nil {
		return nil, err
	}

	return &ActionEventResponse{
		TenantId: resp.TenantId,
		WorkerId: resp.WorkerId,
	}, nil
}

func (d *dispatcherClientImpl) ReleaseSlot(ctx context.Context, stepRunId string) error {
	_, err := d.client.ReleaseSlot(d.ctx.newContext(ctx), &dispatchercontracts.ReleaseSlotRequest{
		TaskRunExternalId: stepRunId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *dispatcherClientImpl) RefreshTimeout(ctx context.Context, stepRunId string, incrementTimeoutBy string) error {
	_, err := d.client.RefreshTimeout(d.ctx.newContext(ctx), &dispatchercontracts.RefreshTimeoutRequest{
		TaskRunExternalId:  stepRunId,
		IncrementTimeoutBy: incrementTimeoutBy,
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *dispatcherClientImpl) UpsertWorkerLabels(ctx context.Context, workerId string, req map[string]interface{}) error {
	labels := mapLabels(req)

	_, err := d.client.UpsertWorkerLabels(d.ctx.newContext(ctx), &dispatchercontracts.UpsertWorkerLabelsRequest{
		WorkerId: workerId,
		Labels:   labels,
	})

	if err != nil {
		return err
	}

	return nil
}

func mapLabels(req map[string]interface{}) map[string]*dispatchercontracts.WorkerLabels {
	labels := map[string]*dispatchercontracts.WorkerLabels{}

	for k, v := range req {
		label := dispatchercontracts.WorkerLabels{}

		switch value := v.(type) {
		case string:
			strValue := value
			label.StrValue = &strValue
		case int:
			intValue := int32(value) // nolint: gosec
			label.IntValue = &intValue
		case int32:
			label.IntValue = &value
		case int64:
			intValue := int32(value) // nolint: gosec
			label.IntValue = &intValue
		default:
			// For any other type, convert to string
			strValue := fmt.Sprintf("%v", value)
			label.StrValue = &strValue
		}

		labels[k] = &label
	}
	return labels
}

func (d *dispatcherClientImpl) RegisterDurableEvent(ctx context.Context, req *sharedcontracts.RegisterDurableEventRequest) (*sharedcontracts.RegisterDurableEventResponse, error) {
	return d.clientv1.RegisterDurableEvent(d.ctx.newContext(ctx), req)
}

func (d *dispatcherClientImpl) DurableTaskStream(ctx context.Context) (sharedcontracts.V1Dispatcher_DurableTaskClient, error) {
	return d.clientv1.DurableTask(d.ctx.newContext(ctx), grpc_retry.Disable())
}
