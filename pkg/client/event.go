package client

import (
	"context"
	"encoding/json"

	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventClient interface {
	Push(ctx context.Context, eventKey string, payload interface{}) error
}

type eventClientImpl struct {
	client eventcontracts.EventsServiceClient

	tenantId string

	l *zerolog.Logger

	v validator.Validator
}

func newEvent(conn *grpc.ClientConn, opts *sharedClientOpts) EventClient {
	return &eventClientImpl{
		client:   eventcontracts.NewEventsServiceClient(conn),
		tenantId: opts.tenantId,
		l:        opts.l,
		v:        opts.v,
	}
}

func (a *eventClientImpl) Push(ctx context.Context, eventKey string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	_, err = a.client.Push(ctx, &eventcontracts.PushEventRequest{
		TenantId:       a.tenantId,
		Key:            eventKey,
		Payload:        string(payloadBytes),
		EventTimestamp: timestamppb.Now(),
	})

	if err != nil {
		return err
	}

	return nil
}
