package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type PayloadShape interface {
	payloadType() sqlcv1.V1PayloadType
}

type TaskInputShape struct{}
type DAGInputShape struct{}
type TaskOutputShape struct{}
type TaskEventDataShape struct{}
type UserEventInputShape struct{}
type DurableEventLogEntryDataShape struct{}
type DurableEventLogEntryResultDataShape struct{}

func (TaskInputShape) payloadType() sqlcv1.V1PayloadType  { return sqlcv1.V1PayloadTypeTASKINPUT }
func (DAGInputShape) payloadType() sqlcv1.V1PayloadType   { return sqlcv1.V1PayloadTypeDAGINPUT }
func (TaskOutputShape) payloadType() sqlcv1.V1PayloadType { return sqlcv1.V1PayloadTypeTASKOUTPUT }
func (TaskEventDataShape) payloadType() sqlcv1.V1PayloadType {
	return sqlcv1.V1PayloadTypeTASKEVENTDATA
}
func (UserEventInputShape) payloadType() sqlcv1.V1PayloadType {
	return sqlcv1.V1PayloadTypeUSEREVENTINPUT
}
func (DurableEventLogEntryDataShape) payloadType() sqlcv1.V1PayloadType {
	return sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA
}
func (DurableEventLogEntryResultDataShape) payloadType() sqlcv1.V1PayloadType {
	return sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA
}

type Payload[S PayloadShape] struct {
	raw []byte
}

func NewPayload[S PayloadShape](raw []byte) Payload[S] {
	return Payload[S]{raw: raw}
}

func (p Payload[S]) PayloadType() sqlcv1.V1PayloadType {
	var shape S
	return shape.payloadType()
}

func (p Payload[S]) Bytes() []byte {
	return p.raw
}

type AnyPayload interface {
	PayloadType() sqlcv1.V1PayloadType
	Bytes() []byte
}

func NewTaskInputPayload(data *V1StepRunData) Payload[TaskInputShape] {
	return NewPayload[TaskInputShape](data.Bytes())
}

func DecodeTaskInputPayload(p Payload[TaskInputShape]) (*V1StepRunData, error) {
	if len(p.raw) == 0 {
		return nil, nil
	}

	data := &V1StepRunData{}

	if err := json.Unmarshal(p.raw, data); err != nil {
		return nil, fmt.Errorf("failed to decode task input payload: %w", err)
	}

	return data, nil
}

func DecodeTaskOutputEventPayload(p Payload[TaskEventDataShape]) (*TaskOutputEvent, error) {
	return newTaskEventFromBytes(p.raw)
}

func DecodeChildWorkflowSignalCreatedPayload(p Payload[TaskEventDataShape]) (*ChildWorkflowSignalCreatedData, error) {
	return newChildWorkflowSignalCreatedDataFromBytes(p.raw)
}

type PayloadRef[S PayloadShape] struct {
	TenantId   uuid.UUID
	Id         int64
	InsertedAt pgtype.Timestamptz
	ExternalId uuid.UUID
}

func (r PayloadRef[S]) retrieveOpts() RetrievePayloadOpts {
	return RetrievePayloadOpts{
		TenantId:   r.TenantId,
		Id:         r.Id,
		InsertedAt: r.InsertedAt,
		Type:       Payload[S]{}.PayloadType(),
		ExternalId: r.ExternalId,
	}
}

type TypedPayloadStore[S PayloadShape] struct {
	store PayloadStoreRepository
}

func (s TypedPayloadStore[S]) RetrieveOne(ctx context.Context, tx sqlcv1.DBTX, ref PayloadRef[S]) (Payload[S], error) {
	payloads, err := s.Retrieve(ctx, tx, ref)

	if err != nil {
		return Payload[S]{}, err
	}

	return payloads[ref], nil
}

func (s TypedPayloadStore[S]) Retrieve(ctx context.Context, tx sqlcv1.DBTX, refs ...PayloadRef[S]) (map[PayloadRef[S]]Payload[S], error) {
	opts := make([]RetrievePayloadOpts, len(refs))

	for i, ref := range refs {
		opts[i] = ref.retrieveOpts()
	}

	optsToBytes, err := s.store.Retrieve(ctx, tx, opts...)

	if err != nil {
		return nil, err
	}

	result := make(map[PayloadRef[S]]Payload[S], len(optsToBytes))

	for _, ref := range refs {
		if raw, ok := optsToBytes[ref.retrieveOpts()]; ok {
			result[ref] = NewPayload[S](raw)
		}
	}

	return result, nil
}

func RetrievePayloadOrFallback[S PayloadShape](payloads map[PayloadRef[S]]Payload[S], ref PayloadRef[S], fallback []byte) Payload[S] {
	if payload, ok := payloads[ref]; ok {
		return payload
	}

	return NewPayload[S](fallback)
}
