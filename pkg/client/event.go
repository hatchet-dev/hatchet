package client

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type EventClient interface {
	Push(ctx context.Context, eventKey string, payload interface{}) error

	PutLog(ctx context.Context, stepRunId, msg string) error

	PutStreamEvent(ctx context.Context, stepRunId string, message []byte) error
}

type eventClientImpl struct {
	client eventcontracts.EventsServiceClient

	tenantId string

	namespace string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newEvent(conn *grpc.ClientConn, opts *sharedClientOpts) EventClient {
	return &eventClientImpl{
		client:    eventcontracts.NewEventsServiceClient(conn),
		tenantId:  opts.tenantId,
		namespace: opts.namespace,
		l:         opts.l,
		v:         opts.v,
		ctx:       opts.ctxLoader,
	}
}

func (a *eventClientImpl) Push(ctx context.Context, eventKey string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	_, err = a.client.Push(a.ctx.newContext(ctx), &eventcontracts.PushEventRequest{
		Key:            a.namespace + eventKey,
		Payload:        string(payloadBytes),
		EventTimestamp: timestamppb.Now(),
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *eventClientImpl) PutLog(ctx context.Context, stepRunId, msg string) error {
	_, err := a.client.PutLog(a.ctx.newContext(ctx), &eventcontracts.PutLogRequest{
		CreatedAt: timestamppb.Now(),
		StepRunId: stepRunId,
		Message:   msg,
	})

	return err
}

func (a *eventClientImpl) PutStreamEvent(ctx context.Context, stepRunId string, message []byte) error {
	_, err := a.client.PutStreamEvent(a.ctx.newContext(ctx), &eventcontracts.PutStreamEventRequest{
		CreatedAt: timestamppb.Now(),
		StepRunId: stepRunId,
		Message:   message,
	})

	return err
}
