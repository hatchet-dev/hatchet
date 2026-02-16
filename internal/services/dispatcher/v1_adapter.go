package dispatcher

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type V1DispatcherAdapter struct {
	v1contracts.UnimplementedDispatcherServer
	dispatcher *DispatcherImpl
}

func NewV1DispatcherAdapter(dispatcher *DispatcherImpl) *V1DispatcherAdapter {
	if dispatcher != nil {
		dispatcher.SetDurableCallbackHandler(dispatcher.DeliverCallbackCompletion)
	}
	return &V1DispatcherAdapter{dispatcher: dispatcher}
}

func (a *V1DispatcherAdapter) Register(ctx context.Context, req *v1contracts.WorkerRegisterRequest) (*v1contracts.WorkerRegisterResponse, error) {
	legacyReq := &dispatchercontracts.WorkerRegisterRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.Register(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.WorkerRegisterResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) Listen(req *v1contracts.WorkerListenRequest, server v1contracts.Dispatcher_ListenServer) error {
	legacyReq := &dispatchercontracts.WorkerListenRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return err
	}

	return a.dispatcher.Listen(legacyReq, &v1ListenAdapter{Dispatcher_ListenServer: server})
}

func (a *V1DispatcherAdapter) ListenV2(req *v1contracts.WorkerListenRequest, server v1contracts.Dispatcher_ListenV2Server) error {
	legacyReq := &dispatchercontracts.WorkerListenRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return err
	}

	return a.dispatcher.ListenV2(legacyReq, &v1ListenV2Adapter{Dispatcher_ListenV2Server: server})
}

func (a *V1DispatcherAdapter) Heartbeat(ctx context.Context, req *v1contracts.HeartbeatRequest) (*v1contracts.HeartbeatResponse, error) {
	legacyReq := &dispatchercontracts.HeartbeatRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.Heartbeat(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.HeartbeatResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) SubscribeToWorkflowEvents(req *v1contracts.SubscribeToWorkflowEventsRequest, server v1contracts.Dispatcher_SubscribeToWorkflowEventsServer) error {
	legacyReq := &dispatchercontracts.SubscribeToWorkflowEventsRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return err
	}

	return a.dispatcher.SubscribeToWorkflowEvents(legacyReq, &v1SubscribeToWorkflowEventsAdapter{Dispatcher_SubscribeToWorkflowEventsServer: server})
}

func (a *V1DispatcherAdapter) SubscribeToWorkflowRuns(server v1contracts.Dispatcher_SubscribeToWorkflowRunsServer) error {
	return a.dispatcher.SubscribeToWorkflowRuns(&v1SubscribeToWorkflowRunsAdapter{Dispatcher_SubscribeToWorkflowRunsServer: server})
}

func (a *V1DispatcherAdapter) SendStepActionEvent(ctx context.Context, req *v1contracts.StepActionEvent) (*v1contracts.ActionEventResponse, error) {
	legacyReq := &dispatchercontracts.StepActionEvent{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.SendStepActionEvent(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.ActionEventResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) SendGroupKeyActionEvent(ctx context.Context, req *v1contracts.GroupKeyActionEvent) (*v1contracts.ActionEventResponse, error) {
	legacyReq := &dispatchercontracts.GroupKeyActionEvent{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.SendGroupKeyActionEvent(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.ActionEventResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) PutOverridesData(ctx context.Context, req *v1contracts.OverridesData) (*v1contracts.OverridesDataResponse, error) {
	legacyReq := &dispatchercontracts.OverridesData{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.PutOverridesData(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.OverridesDataResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) Unsubscribe(ctx context.Context, req *v1contracts.WorkerUnsubscribeRequest) (*v1contracts.WorkerUnsubscribeResponse, error) {
	legacyReq := &dispatchercontracts.WorkerUnsubscribeRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.Unsubscribe(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.WorkerUnsubscribeResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) RefreshTimeout(ctx context.Context, req *v1contracts.RefreshTimeoutRequest) (*v1contracts.RefreshTimeoutResponse, error) {
	legacyReq := &dispatchercontracts.RefreshTimeoutRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.RefreshTimeout(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.RefreshTimeoutResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) ReleaseSlot(ctx context.Context, req *v1contracts.ReleaseSlotRequest) (*v1contracts.ReleaseSlotResponse, error) {
	legacyReq := &dispatchercontracts.ReleaseSlotRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.ReleaseSlot(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.ReleaseSlotResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) UpsertWorkerLabels(ctx context.Context, req *v1contracts.UpsertWorkerLabelsRequest) (*v1contracts.UpsertWorkerLabelsResponse, error) {
	legacyReq := &dispatchercontracts.UpsertWorkerLabelsRequest{}
	if err := convertProto(req, legacyReq); err != nil {
		return nil, err
	}

	legacyResp, err := a.dispatcher.UpsertWorkerLabels(ctx, legacyReq)
	if err != nil {
		return nil, err
	}

	resp := &v1contracts.UpsertWorkerLabelsResponse{}
	if err := convertProto(legacyResp, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *V1DispatcherAdapter) DurableTask(server v1contracts.Dispatcher_DurableTaskServer) error {
	return a.dispatcher.DurableTask(server)
}

func (a *V1DispatcherAdapter) RegisterDurableEvent(ctx context.Context, req *v1contracts.RegisterDurableEventRequest) (*v1contracts.RegisterDurableEventResponse, error) {
	return a.dispatcher.RegisterDurableEvent(ctx, req)
}

func (a *V1DispatcherAdapter) ListenForDurableEvent(server v1contracts.Dispatcher_ListenForDurableEventServer) error {
	return a.dispatcher.ListenForDurableEvent(server)
}

type v1ListenAdapter struct {
	v1contracts.Dispatcher_ListenServer
}

func (a *v1ListenAdapter) Send(msg *dispatchercontracts.AssignedAction) error {
	converted := &v1contracts.AssignedAction{}
	if err := convertProto(msg, converted); err != nil {
		return err
	}
	return a.Dispatcher_ListenServer.Send(converted)
}

type v1ListenV2Adapter struct {
	v1contracts.Dispatcher_ListenV2Server
}

func (a *v1ListenV2Adapter) Send(msg *dispatchercontracts.AssignedAction) error {
	converted := &v1contracts.AssignedAction{}
	if err := convertProto(msg, converted); err != nil {
		return err
	}
	return a.Dispatcher_ListenV2Server.Send(converted)
}

type v1SubscribeToWorkflowEventsAdapter struct {
	v1contracts.Dispatcher_SubscribeToWorkflowEventsServer
}

func (a *v1SubscribeToWorkflowEventsAdapter) Send(msg *dispatchercontracts.WorkflowEvent) error {
	converted := &v1contracts.WorkflowEvent{}
	if err := convertProto(msg, converted); err != nil {
		return err
	}
	return a.Dispatcher_SubscribeToWorkflowEventsServer.Send(converted)
}

type v1SubscribeToWorkflowRunsAdapter struct {
	v1contracts.Dispatcher_SubscribeToWorkflowRunsServer
}

func (a *v1SubscribeToWorkflowRunsAdapter) Send(msg *dispatchercontracts.WorkflowRunEvent) error {
	converted := &v1contracts.WorkflowRunEvent{}
	if err := convertProto(msg, converted); err != nil {
		return err
	}
	return a.Dispatcher_SubscribeToWorkflowRunsServer.Send(converted)
}

func (a *v1SubscribeToWorkflowRunsAdapter) Recv() (*dispatchercontracts.SubscribeToWorkflowRunsRequest, error) {
	msg, err := a.Dispatcher_SubscribeToWorkflowRunsServer.Recv()
	if err != nil {
		return nil, err
	}

	converted := &dispatchercontracts.SubscribeToWorkflowRunsRequest{}
	if err := convertProto(msg, converted); err != nil {
		return nil, err
	}

	return converted, nil
}

func convertProto(in proto.Message, out proto.Message) error {
	if in == nil {
		return errors.New("source proto is nil")
	}

	if out == nil {
		return errors.New("destination proto is nil")
	}

	data, err := proto.Marshal(in)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, out)
}
