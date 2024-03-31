package client

import (
	"context"
	"errors"
	"io"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type RunEvent *dispatchercontracts.WorkflowEvent

type RunHandler func(event RunEvent) error

type RunClient interface {
	On(ctx context.Context, workflowRunId string, handler RunHandler) error
}

type ClientEventListener interface {
	OnRunEvent(ctx context.Context, event *RunEvent) error
}

type runClientImpl struct {
	client dispatchercontracts.DispatcherClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newRun(conn *grpc.ClientConn, opts *sharedClientOpts) RunClient {
	return &runClientImpl{
		client: dispatchercontracts.NewDispatcherClient(conn),
		l:      opts.l,
		v:      opts.v,
		ctx:    opts.ctxLoader,
	}
}

func (r *runClientImpl) On(ctx context.Context, workflowRunId string, handler RunHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		WorkflowRunId: workflowRunId,
	})

	if err != nil {
		return err
	}

	for {
		event, err := stream.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if err := handler(event); err != nil {
			return err
		}
	}
}
