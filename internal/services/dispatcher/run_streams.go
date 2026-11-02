package dispatcher

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// RegisterRunStream serves one of the dispatcher's run observation streams
// (subscribeToWorkflowRunsV1, listenForDurableEvent, subscribeToWorkflowEventsV1) over
// channels to a caller in the same process; see rpcstream.ChannelStream for the contract.
// ctx must carry the tenant, as the auth middleware puts it. The durable streams kind is not
// the dispatcher's; the streams service serves it.
func (d *DispatcherImpl) RegisterRunStream(ctx context.Context, kind operator.RunStreamKind, first proto.Message) (*rpcstream.ChannelStream, error) {
	if _, ok := ctx.Value("tenant").(*sqlcv1.Tenant); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tenant not found on context"))
	}

	switch kind {
	case operator.RunStreamWorkflowRuns:
		return rpcstream.ServeBidi(ctx, first, func(ctx context.Context, receive func() (*contracts.SubscribeToWorkflowRunsRequest, error), sender *rpcstream.Sender[contracts.WorkflowRunEvent]) error {
			return d.subscribeToWorkflowRunsV1(ctx, receive, sender)
		})
	case operator.RunStreamDurableEvents:
		return rpcstream.ServeBidi(ctx, first, func(ctx context.Context, receive func() (*v1.ListenForDurableEventRequest, error), sender *rpcstream.Sender[v1.DurableEvent]) error {
			return d.serviceV1.listenForDurableEvent(ctx, receive, sender)
		})
	case operator.RunStreamWorkflowEvents:
		req, ok := first.(*contracts.SubscribeToWorkflowEventsRequest)

		if !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s takes a SubscribeToWorkflowEventsRequest, not a %T", kind, first))
		}

		return rpcstream.ServeServer(ctx, func(ctx context.Context, sender *rpcstream.Sender[contracts.WorkflowEvent]) error {
			return d.subscribeToWorkflowEventsV1(ctx, req, sender)
		}), nil
	case operator.RunStreamDurableStreams:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("%s is served by the streams service, not the dispatcher", kind))
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("run stream kind %d is not served by the dispatcher", int(kind)))
	}
}
