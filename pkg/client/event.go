// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type pushOpt struct {
	additionalMetadata map[string]string
	priority           *int32
	scope              *string
}

type PushOpFunc func(*pushOpt) error

type BulkPushOpFunc func(*eventcontracts.BulkPushEventRequest) error

type streamEventOpts struct {
	index *int64
}

type StreamEventOption func(*streamEventOpts)

func WithStreamEventIndex(index int64) StreamEventOption {
	return func(opts *streamEventOpts) {
		opts.index = &index
	}
}

type EventClient interface {
	Push(ctx context.Context, eventKey string, payload interface{}, options ...PushOpFunc) error

	BulkPush(ctx context.Context, payloads []EventWithAdditionalMetadata, options ...BulkPushOpFunc) error

	PutLog(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32) error

	PutLogWithTimestamp(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32, createdAt *timestamppb.Timestamp) error

	PutStreamEvent(ctx context.Context, stepRunId string, message []byte, options ...StreamEventOption) error
}

type EventWithAdditionalMetadata struct {
	Event              interface{}       `json:"event"`
	AdditionalMetadata map[string]string `json:"metadata"`
	Key                string            `json:"key"`
	Priority           *int32            `json:"priority"`
	Scope              *string           `json:"scope"`
}

type eventClientImpl struct {
	client eventcontracts.EventsServiceClient

	tenantId string

	namespace string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	sharedMeta map[string]string
}

func newEvent(conn *grpc.ClientConn, opts *sharedClientOpts) EventClient {
	return &eventClientImpl{
		client:     eventcontracts.NewEventsServiceClient(conn),
		tenantId:   opts.tenantId,
		namespace:  opts.namespace,
		l:          opts.l,
		v:          opts.v,
		ctx:        opts.ctxLoader,
		sharedMeta: opts.sharedMeta,
	}
}

func WithEventMetadata(metadata map[string]string) PushOpFunc {
	return func(r *pushOpt) error {
		r.additionalMetadata = metadata

		return nil
	}
}

func WithEventPriority(priority *int32) PushOpFunc {
	return func(r *pushOpt) error {
		r.priority = priority
		return nil
	}
}

func WithFilterScope(scope *string) PushOpFunc {
	return func(r *pushOpt) error {
		r.scope = scope
		return nil
	}
}

func (a *eventClientImpl) Push(ctx context.Context, eventKey string, payload interface{}, options ...PushOpFunc) error {
	key := client.ApplyNamespace(eventKey, &a.namespace)

	request := eventcontracts.PushEventRequest{
		Key:            key,
		EventTimestamp: timestamppb.Now(),
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	request.Payload = string(payloadBytes)

	opts := &pushOpt{}

	for _, optionFunc := range options {
		err = optionFunc(opts)
		if err != nil {
			return err
		}
	}

	additionalMetaBytes, err := a.getAdditionalMetaBytes(&opts.additionalMetadata)

	if err != nil {
		return err
	}

	additionalMetaString := string(additionalMetaBytes)

	request.AdditionalMetadata = &additionalMetaString
	request.Priority = opts.priority
	request.Scope = opts.scope

	_, err = a.client.Push(a.ctx.newContext(ctx), &request)

	if err != nil {
		return err
	}

	return nil
}

func (a *eventClientImpl) BulkPush(ctx context.Context, payload []EventWithAdditionalMetadata, options ...BulkPushOpFunc) error {

	request := eventcontracts.BulkPushEventRequest{}

	var events []*eventcontracts.PushEventRequest

	for _, p := range payload {

		ePayload, err := json.Marshal(p.Event)
		if err != nil {
			return err
		}
		md := p.AdditionalMetadata
		eMetadata, err := a.getAdditionalMetaBytes(&md)
		if err != nil {
			return err
		}
		eMetadataString := string(eMetadata)

		events = append(events, &eventcontracts.PushEventRequest{
			Key:                a.namespace + p.Key,
			EventTimestamp:     timestamppb.Now(),
			Payload:            string(ePayload),
			AdditionalMetadata: &eMetadataString,
			Priority:           p.Priority,
			Scope:              p.Scope,
		})
	}

	request.Events = events

	_, err := a.client.BulkPush(a.ctx.newContext(ctx), &request)

	if err != nil {
		return err
	}

	return nil
}

func (a *eventClientImpl) PutLog(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32) error {
	return a.PutLogWithTimestamp(ctx, taskRunId, msg, level, taskRetryCount, timestamppb.Now())
}

func (a *eventClientImpl) PutLogWithTimestamp(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32, createdAt *timestamppb.Timestamp) error {
	_, err := a.client.PutLog(a.ctx.newContext(ctx), &eventcontracts.PutLogRequest{
		CreatedAt:         createdAt,
		TaskRunExternalId: taskRunId,
		Message:           msg,
		Level:             level,
		TaskRetryCount:    taskRetryCount,
	})

	return err
}

func (a *eventClientImpl) PutStreamEvent(ctx context.Context, taskRunId string, message []byte, options ...StreamEventOption) error {
	opts := &streamEventOpts{}

	for _, optionFunc := range options {
		optionFunc(opts)
	}

	request := &eventcontracts.PutStreamEventRequest{
		CreatedAt:         timestamppb.Now(),
		TaskRunExternalId: taskRunId,
		Message:           message,
	}

	if opts.index != nil {
		request.EventIndex = opts.index
	}

	_, err := a.client.PutStreamEvent(a.ctx.newContext(ctx), request)

	return err
}

func (e *eventClientImpl) getAdditionalMetaBytes(opt *map[string]string) ([]byte, error) {
	additionalMeta := make(map[string]string)

	for key, value := range e.sharedMeta {
		additionalMeta[key] = value
	}

	if opt != nil {
		for key, value := range *opt {
			additionalMeta[key] = value
		}
	}

	metadataBytes, err := json.Marshal(additionalMeta)

	if err != nil {
		return nil, fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	return metadataBytes, nil
}
