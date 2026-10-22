package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/listutils"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type PayloadKind int

const (
	PayloadKindTaskInput PayloadKind = iota
	PayloadKindDAGInput
	PayloadKindUserEventInput
	PayloadKindTaskOutput
	PayloadKindTaskCompletionEvent
	PayloadKindChildWorkflowSignalCreated
	PayloadKindSignalCompletedMatchData
	PayloadKindDurableEventLogEntryData
	PayloadKindDurableEventLogEntryResultData
)

func (k PayloadKind) V1PayloadType() sqlcv1.V1PayloadType {
	switch k {
	case PayloadKindTaskInput:
		return sqlcv1.V1PayloadTypeTASKINPUT
	case PayloadKindDAGInput:
		return sqlcv1.V1PayloadTypeDAGINPUT
	case PayloadKindUserEventInput:
		return sqlcv1.V1PayloadTypeUSEREVENTINPUT
	case PayloadKindTaskOutput:
		return sqlcv1.V1PayloadTypeTASKOUTPUT
	case PayloadKindTaskCompletionEvent, PayloadKindChildWorkflowSignalCreated, PayloadKindSignalCompletedMatchData:
		return sqlcv1.V1PayloadTypeTASKEVENTDATA
	case PayloadKindDurableEventLogEntryData:
		return sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA
	case PayloadKindDurableEventLogEntryResultData:
		return sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYRESULTDATA
	}

	panic(fmt.Sprintf("unknown payload kind %d", k))
}

type PayloadShape interface {
	kind() PayloadKind
}

type TaskInputShape struct{}
type DAGInputShape struct{}
type UserEventInputShape struct{}
type TaskOutputShape struct{}
type TaskCompletionEventShape struct{}
type ChildWorkflowSignalCreatedShape struct{}
type SignalCompletedMatchDataShape struct{}
type DurableEventLogEntryDataShape struct{}
type DurableEventLogEntryResultDataShape struct{}

func (TaskInputShape) kind() PayloadKind           { return PayloadKindTaskInput }
func (DAGInputShape) kind() PayloadKind            { return PayloadKindDAGInput }
func (UserEventInputShape) kind() PayloadKind      { return PayloadKindUserEventInput }
func (TaskOutputShape) kind() PayloadKind          { return PayloadKindTaskOutput }
func (TaskCompletionEventShape) kind() PayloadKind { return PayloadKindTaskCompletionEvent }
func (ChildWorkflowSignalCreatedShape) kind() PayloadKind {
	return PayloadKindChildWorkflowSignalCreated
}
func (SignalCompletedMatchDataShape) kind() PayloadKind { return PayloadKindSignalCompletedMatchData }
func (DurableEventLogEntryDataShape) kind() PayloadKind { return PayloadKindDurableEventLogEntryData }
func (DurableEventLogEntryResultDataShape) kind() PayloadKind {
	return PayloadKindDurableEventLogEntryResultData
}

type Payload[S PayloadShape] struct {
	raw []byte
}

func NewPayload[S PayloadShape](raw []byte) Payload[S] {
	return Payload[S]{raw: raw}
}

func (p Payload[S]) Bytes() []byte {
	return p.raw
}

func (p Payload[S]) IsEmpty() bool {
	trimmed := bytes.TrimSpace(p.raw)

	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("{}"))
}

func (p Payload[S]) BytesOrEmptyJSONObject() []byte {
	if len(p.raw) == 0 {
		return []byte("{}")
	}

	return p.raw
}

type AnyPayload interface {
	Bytes() []byte
	IsEmpty() bool
}

func DecodeTaskCompletionEventPayload(p Payload[TaskCompletionEventShape]) (*TaskOutputEvent, error) {
	return newTaskEventFromBytes(p.raw)
}

func DecodeChildWorkflowSignalCreatedPayload(p Payload[ChildWorkflowSignalCreatedShape]) (*ChildWorkflowSignalCreatedData, error) {
	return newChildWorkflowSignalCreatedDataFromBytes(p.raw)
}

type PayloadRef[S PayloadShape] struct {
	TenantId           uuid.UUID
	Id                 int64
	InsertedAt         pgtype.Timestamptz
	ExternalId         uuid.UUID
	PayloadReferenceId *uuid.UUID
}

func (r PayloadRef[S]) retrieveOpts() RetrievePayloadOpts {
	var shape S

	return RetrievePayloadOpts{
		TenantId:           r.TenantId,
		Id:                 r.Id,
		InsertedAt:         r.InsertedAt,
		Type:               shape.kind().V1PayloadType(),
		ExternalId:         r.ExternalId,
		PayloadReferenceId: r.PayloadReferenceId,
	}
}

func (r PayloadRef[S]) StoreOpts(payload Payload[S]) StorePayloadOpts {
	return StorePayloadOpts{key: r.retrieveOpts(), payload: payload}
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
	opts := listutils.Map(refs, func(ref PayloadRef[S]) RetrievePayloadOpts {
		return ref.retrieveOpts()
	})

	optsToBytes, err := s.store.Retrieve(ctx, tx, opts...)

	if err != nil {
		return nil, err
	}

	return typedPayloadsFromBytes[S](refs, optsToBytes), nil
}

// Resolve turns v1_payload rows the caller already read for refs into payloads, fetching external and
// offloaded content the way Retrieve does but without re-reading the rows.
func (s TypedPayloadStore[S]) Resolve(ctx context.Context, tx sqlcv1.DBTX, rows []*sqlcv1.V1Payload, refs ...PayloadRef[S]) (map[PayloadRef[S]]Payload[S], error) {
	opts := listutils.Map(refs, func(ref PayloadRef[S]) RetrievePayloadOpts {
		return ref.retrieveOpts()
	})

	optsToBytes, err := s.store.Resolve(ctx, tx, rows, opts...)

	if err != nil {
		return nil, err
	}

	return typedPayloadsFromBytes[S](refs, optsToBytes), nil
}

func typedPayloadsFromBytes[S PayloadShape](refs []PayloadRef[S], optsToBytes map[RetrievePayloadOpts][]byte) map[PayloadRef[S]]Payload[S] {
	result := make(map[PayloadRef[S]]Payload[S], len(optsToBytes))

	for _, ref := range refs {
		if raw, ok := optsToBytes[ref.retrieveOpts()]; ok {
			result[ref] = NewPayload[S](raw)
		}
	}

	return result
}

func RetrievePayloadOrFallback[S PayloadShape](payloads map[PayloadRef[S]]Payload[S], ref PayloadRef[S], fallback []byte) Payload[S] {
	if payload, ok := payloads[ref]; ok {
		return payload
	}

	return NewPayload[S](fallback)
}

type StorePayloadOpts struct {
	key     RetrievePayloadOpts
	payload AnyPayload
}

type StoreOLAPPayloadOpts struct {
	ExternalId uuid.UUID
	InsertedAt pgtype.Timestamptz
	Payload    AnyPayload
}

type OffloadToExternalStoreOpts struct {
	TenantId   uuid.UUID
	ExternalID uuid.UUID
	InsertedAt pgtype.Timestamptz
	Payload    []byte
}

type RetrievePayloadOpts struct {
	TenantId           uuid.UUID
	Id                 int64
	InsertedAt         pgtype.Timestamptz
	Type               sqlcv1.V1PayloadType
	ExternalId         uuid.UUID
	PayloadReferenceId *uuid.UUID
}

func (o RetrievePayloadOpts) effectiveExternalId() uuid.UUID {
	if o.PayloadReferenceId != nil {
		return *o.PayloadReferenceId
	}

	return o.ExternalId
}

// payloadReferenceIdOrZero is the reference id as every payload query takes it: the zero uuid means no
// reference. The query derives the inserted_at the row is stored under from this id and InsertedAt
// through v1_payload_inserted_at, so Go always passes the referrer's own InsertedAt.
func (o RetrievePayloadOpts) payloadReferenceIdOrZero() uuid.UUID {
	if o.PayloadReferenceId == nil {
		return uuid.Nil
	}

	return *o.PayloadReferenceId
}

type RetrieveFromExternalOpts struct {
	IndexFileKey ExternalIndexFileLocationKey
	ExternalId   uuid.UUID
}

type PayloadLocation string
type ExternalIndexFileLocationKey string

type CreateIndexBlockOpts struct {
	PartitionDate             PartitionDate
	BlockLowerExternalIdBound uuid.UUID
	BlockUpperExternalIdBound uuid.UUID
	IndexFileKey              string
}

type ExternalStore interface {
	Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (*ExternalIndexFileLocationKey, error)
	Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	OverwriteExisting(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	// Resolve is the second half of Retrieve for callers that already hold the v1_payload rows for opts
	// (read by a statement of their own): inline content is returned as is, external and offloaded
	// payloads are fetched exactly as Retrieve fetches them.
	Resolve(ctx context.Context, tx sqlcv1.DBTX, rows []*sqlcv1.V1Payload, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	TaskInputs() TypedPayloadStore[TaskInputShape]
	DAGInputs() TypedPayloadStore[DAGInputShape]
	UserEventInputs() TypedPayloadStore[UserEventInputShape]
	TaskOutputs() TypedPayloadStore[TaskOutputShape]
	TaskCompletionEvents() TypedPayloadStore[TaskCompletionEventShape]
	ChildWorkflowSignalCreatedEvents() TypedPayloadStore[ChildWorkflowSignalCreatedShape]
	SignalCompletedMatchData() TypedPayloadStore[SignalCompletedMatchDataShape]
	DurableEventLogEntryData() TypedPayloadStore[DurableEventLogEntryDataShape]
	DurableEventLogEntryResultData() TypedPayloadStore[DurableEventLogEntryResultDataShape]
	PayloadConsolidationEnabled() bool
	RetrieveFromExternal(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error)
	OverwriteExternalStore(store ExternalStore)
	ExternalCutoverProcessInterval() time.Duration
	InlineStoreTTL() *time.Duration
	ExternalCutoverBatchSize() int32
	ExternalCutoverNumConcurrentOffloads() int32
	EnableWindowSizeOptimization() bool
	ExternalStoreEnabled() bool
	ExternalStore() ExternalStore
	ProcessPayloadCutovers(ctx context.Context) error
	CreateIndexBlock(ctx context.Context, tx pgx.Tx, opts CreateIndexBlockOpts) error
}

type payloadStoreRepositoryImpl struct {
	pool                                 *pgxpool.Pool
	l                                    *zerolog.Logger
	queries                              *sqlcv1.Queries
	externalStoreEnabled                 bool
	inlineStoreTTL                       *time.Duration
	externalStore                        ExternalStore
	externalCutoverProcessInterval       time.Duration
	externalCutoverBatchSize             int32
	externalCutoverNumConcurrentOffloads int32
	enableWindowSizeOptimization         bool
	enablePayloadConsolidation           bool
}

type PayloadStoreRepositoryOpts struct {
	ExternalCutoverProcessInterval       time.Duration
	ExternalCutoverBatchSize             int32
	ExternalCutoverNumConcurrentOffloads int32
	InlineStoreTTL                       *time.Duration
	EnableWindowSizeOptimization         bool
	WrapTaskInputInStepRunData           bool
	EnablePayloadConsolidation           bool
}

func NewPayloadStoreRepository(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	queries *sqlcv1.Queries,
	opts PayloadStoreRepositoryOpts,
) PayloadStoreRepository {
	return &payloadStoreRepositoryImpl{
		pool:    pool,
		l:       l,
		queries: queries,

		externalStoreEnabled:                 false,
		inlineStoreTTL:                       opts.InlineStoreTTL,
		externalStore:                        &NoOpExternalStore{},
		externalCutoverProcessInterval:       opts.ExternalCutoverProcessInterval,
		externalCutoverBatchSize:             opts.ExternalCutoverBatchSize,
		externalCutoverNumConcurrentOffloads: opts.ExternalCutoverNumConcurrentOffloads,
		enableWindowSizeOptimization:         opts.EnableWindowSizeOptimization,
		enablePayloadConsolidation:           opts.EnablePayloadConsolidation,
	}
}

func (p *payloadStoreRepositoryImpl) PayloadConsolidationEnabled() bool {
	return p.enablePayloadConsolidation
}

type PayloadUniqueKey struct {
	ID              int64
	InsertedAtMicro int64 // Unix microseconds — avoids time.Time map-key pitfalls (mono clock, location, sub-μs precision)
	TenantId        uuid.UUID
	Type            sqlcv1.V1PayloadType
}

func payloadUniqueKeyFromRetrieveOpt(opt RetrievePayloadOpts) PayloadUniqueKey {
	return PayloadUniqueKey{
		ID:              opt.Id,
		InsertedAtMicro: opt.InsertedAt.Time.UnixMicro(),
		TenantId:        opt.TenantId,
		Type:            opt.Type,
	}
}

func payloadUniqueKeyFromRow(payload *sqlcv1.V1Payload) PayloadUniqueKey {
	return PayloadUniqueKey{
		ID:              payload.ID,
		InsertedAtMicro: payload.InsertedAt.Time.UnixMicro(),
		TenantId:        payload.TenantID,
		Type:            payload.Type,
	}
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	if len(payloads) == 0 {
		return nil
	}

	taskIds := make([]int64, 0, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	payloadReferenceIds := make([]uuid.UUID, 0, len(payloads))
	payloadTypes := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))
	tenantIds := make([]uuid.UUID, 0, len(payloads))
	externalIds := make([]uuid.UUID, 0, len(payloads))

	seenPayloadUniqueKeys := make(map[PayloadUniqueKey]struct{})

	sort.Slice(payloads, func(i, j int) bool {
		// sort payloads descending by inserted at to deduplicate operations
		return payloads[i].key.InsertedAt.Time.After(payloads[j].key.InsertedAt.Time)
	})

	for _, payload := range payloads {
		if payload.payload.IsEmpty() {
			continue
		}

		uniqueKey := payloadUniqueKeyFromRetrieveOpt(payload.key)

		if _, exists := seenPayloadUniqueKeys[uniqueKey]; exists {
			continue
		}

		seenPayloadUniqueKeys[uniqueKey] = struct{}{}

		taskIds = append(taskIds, payload.key.Id)
		taskInsertedAts = append(taskInsertedAts, payload.key.InsertedAt)
		payloadReferenceIds = append(payloadReferenceIds, payload.key.payloadReferenceIdOrZero())
		payloadTypes = append(payloadTypes, string(payload.key.Type))
		tenantIds = append(tenantIds, payload.key.TenantId)
		inlineContents = append(inlineContents, payload.payload.Bytes())
		externalIds = append(externalIds, payload.key.effectiveExternalId())
	}

	err := p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Ids:                 taskIds,
		Insertedats:         taskInsertedAts,
		Payloadreferenceids: payloadReferenceIds,
		Types:               payloadTypes,
		Tenantids:           tenantIds,
		Inlinecontents:      inlineContents,
		Externalids:         externalIds,
	})

	if err != nil {
		return fmt.Errorf("failed to write payloads: %w", err)
	}

	return err
}

func (p *payloadStoreRepositoryImpl) OverwriteExisting(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	if len(payloads) == 0 {
		return nil
	}

	uniquePayloads := listutils.UniqBy(payloads, func(payload StorePayloadOpts) PayloadUniqueKey {
		return payloadUniqueKeyFromRetrieveOpt(payload.key)
	})

	params := sqlcv1.OverwritePayloadsParams{}

	for _, payload := range uniquePayloads {
		if payload.payload.IsEmpty() {
			continue
		}

		params.Ids = append(params.Ids, payload.key.Id)
		params.Insertedats = append(params.Insertedats, payload.key.InsertedAt)
		params.Payloadreferenceids = append(params.Payloadreferenceids, payload.key.payloadReferenceIdOrZero())
		params.Externalids = append(params.Externalids, payload.key.effectiveExternalId())
		params.Types = append(params.Types, string(payload.key.Type))
		params.Inlinecontents = append(params.Inlinecontents, payload.payload.Bytes())
		params.Tenantids = append(params.Tenantids, payload.key.TenantId)
	}

	if len(params.Ids) == 0 {
		return nil
	}

	err := p.queries.OverwritePayloads(ctx, tx, params)

	if err != nil {
		return fmt.Errorf("failed to overwrite payloads: %w", err)
	}

	return nil
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if tx == nil {
		tx = p.pool
	}

	return p.retrieve(ctx, tx, opts...)
}

func (p *payloadStoreRepositoryImpl) TaskInputs() TypedPayloadStore[TaskInputShape] {
	return TypedPayloadStore[TaskInputShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) DAGInputs() TypedPayloadStore[DAGInputShape] {
	return TypedPayloadStore[DAGInputShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) UserEventInputs() TypedPayloadStore[UserEventInputShape] {
	return TypedPayloadStore[UserEventInputShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) TaskOutputs() TypedPayloadStore[TaskOutputShape] {
	return TypedPayloadStore[TaskOutputShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) TaskCompletionEvents() TypedPayloadStore[TaskCompletionEventShape] {
	return TypedPayloadStore[TaskCompletionEventShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) ChildWorkflowSignalCreatedEvents() TypedPayloadStore[ChildWorkflowSignalCreatedShape] {
	return TypedPayloadStore[ChildWorkflowSignalCreatedShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) SignalCompletedMatchData() TypedPayloadStore[SignalCompletedMatchDataShape] {
	return TypedPayloadStore[SignalCompletedMatchDataShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) DurableEventLogEntryData() TypedPayloadStore[DurableEventLogEntryDataShape] {
	return TypedPayloadStore[DurableEventLogEntryDataShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) DurableEventLogEntryResultData() TypedPayloadStore[DurableEventLogEntryResultDataShape] {
	return TypedPayloadStore[DurableEventLogEntryResultDataShape]{store: p}
}

func (p *payloadStoreRepositoryImpl) RetrieveFromExternal(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error) {
	if !p.externalStoreEnabled {
		return nil, fmt.Errorf("external store not enabled")
	}

	return p.externalStore.Retrieve(ctx, opts...)
}

func (p *payloadStoreRepositoryImpl) retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if len(opts) == 0 {
		return make(map[RetrievePayloadOpts][]byte), nil
	}

	ids := make([]int64, len(opts))
	insertedAts := make([]pgtype.Timestamptz, len(opts))
	types := make([]string, len(opts))
	tenantIds := make([]uuid.UUID, len(opts))
	payloadReferenceIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		ids[i] = opt.Id
		insertedAts[i] = opt.InsertedAt
		types[i] = string(opt.Type)
		tenantIds[i] = opt.TenantId
		payloadReferenceIds[i] = opt.payloadReferenceIdOrZero()
	}

	payloads, err := p.queries.ReadPayloads(ctx, tx, sqlcv1.ReadPayloadsParams{
		Ids:                 ids,
		Insertedats:         insertedAts,
		Tenantids:           tenantIds,
		Types:               types,
		Payloadreferenceids: payloadReferenceIds,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	return p.resolve(ctx, tx, payloads, opts...)
}

func (p *payloadStoreRepositoryImpl) Resolve(ctx context.Context, tx sqlcv1.DBTX, rows []*sqlcv1.V1Payload, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if tx == nil {
		tx = p.pool
	}

	return p.resolve(ctx, tx, rows, opts...)
}

// resolve maps the v1_payload rows read for opts back onto opts and fetches whatever is not inline:
// opts without a row are looked up in the external store's index blocks when the external store is
// enabled.
func (p *payloadStoreRepositoryImpl) resolve(ctx context.Context, tx sqlcv1.DBTX, payloads []*sqlcv1.V1Payload, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if len(opts) == 0 {
		return make(map[RetrievePayloadOpts][]byte), nil
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)
	originalOptsByKey := make(map[PayloadUniqueKey]RetrievePayloadOpts, len(opts))
	// several rows can share one referenced payload, so each reference id maps to every opt that asked for it
	originalOptsByReferenceId := make(map[uuid.UUID][]RetrievePayloadOpts)

	for _, opt := range opts {
		if opt.PayloadReferenceId != nil {
			originalOptsByReferenceId[*opt.PayloadReferenceId] = append(originalOptsByReferenceId[*opt.PayloadReferenceId], opt)
			continue
		}

		originalOptsByKey[payloadUniqueKeyFromRetrieveOpt(opt)] = opt
	}

	foundKeys := make(map[PayloadUniqueKey]struct{})

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		if referrers, ok := originalOptsByReferenceId[payload.ExternalID]; ok {
			for _, opt := range referrers {
				foundKeys[payloadUniqueKeyFromRetrieveOpt(opt)] = struct{}{}
				optsToPayload[opt] = payload.InlineContent
			}

			continue
		}

		opt, ok := originalOptsByKey[payloadUniqueKeyFromRow(payload)]

		if !ok {
			opt = RetrievePayloadOpts{
				Id:         payload.ID,
				InsertedAt: payload.InsertedAt,
				Type:       payload.Type,
				TenantId:   payload.TenantID,
				ExternalId: payload.ExternalID,
			}
		}

		foundKeys[payloadUniqueKeyFromRetrieveOpt(opt)] = struct{}{}
		optsToPayload[opt] = payload.InlineContent
	}

	if !p.externalStoreEnabled {
		return optsToPayload, nil
	}

	notFoundInline := make([]RetrievePayloadOpts, 0)

	for _, opt := range opts {
		if _, found := foundKeys[payloadUniqueKeyFromRetrieveOpt(opt)]; !found {
			notFoundInline = append(notFoundInline, opt)
		}
	}

	retrieveFromExternalOptsToOpts, err := p.offloadedRetrieveOpts(ctx, tx, notFoundInline)

	if err != nil {
		return nil, err
	}

	if len(retrieveFromExternalOptsToOpts) == 0 {
		return optsToPayload, nil
	}

	externalData, err := p.RetrieveFromExternal(ctx, slices.Collect(maps.Keys(retrieveFromExternalOptsToOpts))...)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve external payloads: %w", err)
	}

	for retrieveFromExternalOpt, data := range externalData {
		for _, opt := range retrieveFromExternalOptsToOpts[retrieveFromExternalOpt] {
			optsToPayload[opt] = data
		}
	}

	return optsToPayload, nil
}

func NewPayloadReferenceId() uuid.UUID {
	return uuid.Must(uuid.NewV7())
}

// offloadedRetrieveOpts resolves the index file holding each payload that is no longer inline. The index is
// keyed by the date of the inserted_at the row was stored under, which the query derives from the reference
// id and the referrer's InsertedAt the same way the write did.
func (p *payloadStoreRepositoryImpl) offloadedRetrieveOpts(ctx context.Context, tx sqlcv1.DBTX, opts []RetrievePayloadOpts) (map[RetrieveFromExternalOpts][]RetrievePayloadOpts, error) {
	retrieveFromExternalOptsToOpts := make(map[RetrieveFromExternalOpts][]RetrievePayloadOpts)
	externalIdToOpts := make(map[uuid.UUID][]RetrievePayloadOpts, len(opts))
	indexBlockExternalIds := make([]uuid.UUID, 0, len(opts))
	indexBlockReferenceIds := make([]uuid.UUID, 0, len(opts))
	indexBlockInsertedAts := make([]pgtype.Timestamptz, 0, len(opts))

	for _, opt := range opts {
		externalId := opt.effectiveExternalId()

		if externalId == uuid.Nil {
			continue
		}

		if _, seen := externalIdToOpts[externalId]; !seen {
			indexBlockExternalIds = append(indexBlockExternalIds, externalId)
			indexBlockReferenceIds = append(indexBlockReferenceIds, opt.payloadReferenceIdOrZero())
			indexBlockInsertedAts = append(indexBlockInsertedAts, opt.InsertedAt)
		}

		externalIdToOpts[externalId] = append(externalIdToOpts[externalId], opt)
	}

	if len(externalIdToOpts) == 0 {
		return retrieveFromExternalOptsToOpts, nil
	}

	indexBlocks, err := p.queries.GetOffloadedPayloadIndexBlocks(ctx, tx, sqlcv1.GetOffloadedPayloadIndexBlocksParams{
		Payloadreferenceids: indexBlockReferenceIds,
		Insertedats:         indexBlockInsertedAts,
		Externalids:         indexBlockExternalIds,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to get offloaded payload index block: %w", err)
	}

	for _, block := range indexBlocks {
		optsForExternalId, requested := externalIdToOpts[block.ExternalID]

		if !requested {
			p.l.Error().Msg("got index file key for external id that was not requested")
			continue
		}

		retrieveFromExternalOptsToOpts[RetrieveFromExternalOpts{
			IndexFileKey: ExternalIndexFileLocationKey(block.IndexFileKey),
			ExternalId:   block.ExternalID,
		}] = optsForExternalId
	}

	return retrieveFromExternalOptsToOpts, nil
}

func (p *payloadStoreRepositoryImpl) OverwriteExternalStore(store ExternalStore) {
	p.externalStoreEnabled = true
	p.externalStore = store
}

func (p *payloadStoreRepositoryImpl) ExternalCutoverProcessInterval() time.Duration {
	return p.externalCutoverProcessInterval
}

func (p *payloadStoreRepositoryImpl) InlineStoreTTL() *time.Duration {
	return p.inlineStoreTTL
}

func (p *payloadStoreRepositoryImpl) ExternalCutoverBatchSize() int32 {
	return p.externalCutoverBatchSize
}

func (p *payloadStoreRepositoryImpl) ExternalCutoverNumConcurrentOffloads() int32 {
	return p.externalCutoverNumConcurrentOffloads
}

func (p *payloadStoreRepositoryImpl) EnableWindowSizeOptimization() bool {
	return p.enableWindowSizeOptimization
}

func (p *payloadStoreRepositoryImpl) ExternalStoreEnabled() bool {
	return p.externalStoreEnabled
}

func (p *payloadStoreRepositoryImpl) ExternalStore() ExternalStore {
	return p.externalStore
}

type CutoverBatchOutcome struct {
	ShouldContinue bool
	NextExternalId uuid.UUID
}

type PartitionDate pgtype.Date

type PayloadMetadata struct {
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	ID         int64
	TenantID   uuid.UUID
}

func (d PartitionDate) String() string {
	return d.Time.Format("20060102")
}

const MAX_PARTITIONS_TO_OFFLOAD = 14                  // two weeks
const MAX_BATCH_SIZE_BYTES = 1.5 * 1024 * 1024 * 1024 // 1.5 GB

func (p *payloadStoreRepositoryImpl) OptimizePayloadWindowSize(ctx context.Context, tx sqlcv1.DBTX, partitionDate PartitionDate, candidateBatchNumRows int32, lastExternalId uuid.UUID) (*int32, error) {
	if candidateBatchNumRows <= 0 {
		// trivial case that we'll never hit, but to prevent infinite recursion
		zero := int32(0)
		return &zero, nil
	}

	proposedBatchSizeBytes, err := p.queries.ComputePayloadBatchSize(ctx, tx, sqlcv1.ComputePayloadBatchSizeParams{
		Partitiondate:  pgtype.Date(partitionDate),
		Lastexternalid: lastExternalId,
		Batchsize:      candidateBatchNumRows,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to compute payload batch size: %w", err)
	}

	if proposedBatchSizeBytes < MAX_BATCH_SIZE_BYTES {
		return &candidateBatchNumRows, nil
	}

	// if the proposed batch size is too large, then
	// cut it in half and try again
	return p.OptimizePayloadWindowSize(
		ctx,
		tx,
		partitionDate,
		candidateBatchNumRows/2,
		lastExternalId,
	)
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadCutoverBatch(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate, lastExternalId uuid.UUID) (*CutoverBatchOutcome, error) {
	ctx, span := telemetry.NewSpan(ctx, "PayloadStoreRepository.ProcessPayloadCutoverBatch")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for copying offloaded payloads: %w", err)
	}

	defer rollback()

	windowSize := p.externalCutoverBatchSize * p.externalCutoverNumConcurrentOffloads

	if p.enableWindowSizeOptimization {
		windowSizePtr, err := p.OptimizePayloadWindowSize(
			ctx,
			tx,
			partitionDate,
			windowSize,
			lastExternalId,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to optimize payload window size: %w", err)
		}

		windowSize = *windowSizePtr
	}

	payloadRanges, err := p.queries.CreatePayloadRangeChunks(ctx, tx, sqlcv1.CreatePayloadRangeChunksParams{
		Chunksize:      p.externalCutoverBatchSize,
		Partitiondate:  pgtype.Date(partitionDate),
		Windowsize:     windowSize,
		Lastexternalid: lastExternalId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to create payload range chunks: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) || len(payloadRanges) == 0 {
		return &CutoverBatchOutcome{
			ShouldContinue: false,
			NextExternalId: lastExternalId,
		}, nil
	}

	if err = commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit payload range chunks transaction: %w", err)
	}

	mu := sync.Mutex{}
	eg := errgroup.Group{}

	offloadToExternalStoreOpts := make([]OffloadToExternalStoreOpts, 0)
	maxExternalId := payloadRanges[len(payloadRanges)-1].UpperExternalID

	numPayloads := 0

	for _, payloadRange := range payloadRanges {
		pr := payloadRange
		eg.Go(func() error {
			payloads, err := p.queries.ListPaginatedPayloadsForOffload(ctx, p.pool, sqlcv1.ListPaginatedPayloadsForOffloadParams{
				Partitiondate:  pgtype.Date(partitionDate),
				Lastexternalid: pr.LowerExternalID,
				Nextexternalid: pr.UpperExternalID,
				Batchsize:      p.externalCutoverBatchSize,
			})

			if err != nil {
				return fmt.Errorf("failed to list paginated payloads for offload: %w", err)
			}

			offloadToExternalStoreOptsInner := make([]OffloadToExternalStoreOpts, 0)

			for _, payload := range payloads {
				externalId := payload.ExternalID

				if externalId == uuid.Nil {
					externalId = uuid.New()
				}

				if payload.Location == sqlcv1.V1PayloadLocationINLINE {
					offloadToExternalStoreOptsInner = append(offloadToExternalStoreOptsInner, OffloadToExternalStoreOpts{
						TenantId:   payload.TenantID,
						ExternalID: externalId,
						InsertedAt: payload.InsertedAt,
						Payload:    payload.InlineContent,
					})
				}
			}

			mu.Lock()
			offloadToExternalStoreOpts = append(offloadToExternalStoreOpts, offloadToExternalStoreOptsInner...)
			numPayloads += len(payloads)
			mu.Unlock()

			return nil
		})
	}

	err = eg.Wait()

	if err != nil {
		return nil, err
	}

	blockIndexKey, err := p.ExternalStore().Store(ctx, offloadToExternalStoreOpts...)

	if err != nil {
		return nil, fmt.Errorf("failed to offload payloads to external store: %w", err)
	}

	span.SetAttributes(attribute.Int("num_payloads_read", numPayloads))

	leaseTx, leaseCommit, leaseRollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for extending cutover job lease: %w", err)
	}

	defer leaseRollback()

	extendedLease, err := p.acquireOrExtendJobLease(ctx, leaseTx, processId, partitionDate, maxExternalId)

	if err != nil {
		return nil, fmt.Errorf("failed to extend cutover job lease: %w", err)
	}

	if !extendedLease.ShouldRun {
		return nil, fmt.Errorf("lease for partition %s was taken by another process during batch processing", partitionDate.String())
	}

	if blockIndexKey != nil {
		if err := p.CreateIndexBlock(ctx, leaseTx, CreateIndexBlockOpts{
			PartitionDate:             partitionDate,
			BlockLowerExternalIdBound: lastExternalId,
			BlockUpperExternalIdBound: maxExternalId,
			IndexFileKey:              string(*blockIndexKey),
		}); err != nil {
			return nil, fmt.Errorf("failed to create index block: %w", err)
		}
	}

	if err := leaseCommit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	if numPayloads < int(windowSize) {
		return &CutoverBatchOutcome{
			ShouldContinue: false,
			NextExternalId: extendedLease.LastExternalId,
		}, nil
	}

	return &CutoverBatchOutcome{
		ShouldContinue: true,
		NextExternalId: extendedLease.LastExternalId,
	}, nil
}

type CutoverJobRunMetadata struct {
	ShouldRun      bool
	LastExternalId uuid.UUID
	PartitionDate  PartitionDate
	LeaseProcessId uuid.UUID
}

func (p *payloadStoreRepositoryImpl) acquireOrExtendJobLease(ctx context.Context, tx pgx.Tx, processId uuid.UUID, partitionDate PartitionDate, lastExternalId uuid.UUID) (*CutoverJobRunMetadata, error) {
	leaseInterval := 2 * time.Minute
	leaseExpiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(leaseInterval))

	lease, err := p.queries.AcquireOrExtendCutoverJobLease(ctx, tx, sqlcv1.AcquireOrExtendCutoverJobLeaseParams{
		Key:            pgtype.Date(partitionDate),
		Leaseprocessid: processId,
		Leaseexpiresat: leaseExpiresAt,
		Lastexternalid: lastExternalId,
	})

	if err != nil {
		// ErrNoRows here means that something else is holding the lease
		// since we did not insert a new record, and the `UPDATE` returned an empty set
		if errors.Is(err, pgx.ErrNoRows) {
			return &CutoverJobRunMetadata{
				ShouldRun:      false,
				PartitionDate:  partitionDate,
				LeaseProcessId: processId,
			}, nil
		}
		return nil, fmt.Errorf("failed to create initial cutover job lease: %w", err)
	}

	if lease.LeaseProcessID != processId || lease.IsCompleted {
		return &CutoverJobRunMetadata{
			ShouldRun:      false,
			LastExternalId: lease.LastExternalID,
			PartitionDate:  partitionDate,
			LeaseProcessId: lease.LeaseProcessID,
		}, nil
	}

	return &CutoverJobRunMetadata{
		ShouldRun:      true,
		LastExternalId: lease.LastExternalID,
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *payloadStoreRepositoryImpl) prepareCutoverTableJob(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate) (*CutoverJobRunMetadata, error) {
	if p.inlineStoreTTL == nil {
		return nil, fmt.Errorf("inline store TTL is not set")
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	var zeroUuid uuid.UUID

	lease, err := p.acquireOrExtendJobLease(ctx, tx, processId, partitionDate, zeroUuid)

	if err != nil {
		return nil, fmt.Errorf("failed to acquire or extend cutover job lease: %w", err)
	}

	if !lease.ShouldRun {
		return lease, nil
	}

	err = p.queries.CreateV1PayloadCutoverTemporaryTable(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return nil, fmt.Errorf("failed to create payload cutover temporary table: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	return &CutoverJobRunMetadata{
		ShouldRun:      true,
		LastExternalId: lease.LastExternalId,
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *payloadStoreRepositoryImpl) processSinglePartition(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate) error {
	ctx, span := telemetry.NewSpan(ctx, "payload_store_repository_impl.processSinglePartition")
	defer span.End()

	jobMeta, err := p.prepareCutoverTableJob(ctx, processId, partitionDate)

	if err != nil {
		return fmt.Errorf("failed to prepare cutover table job: %w", err)
	}

	if !jobMeta.ShouldRun {
		return nil
	}

	lastExternalId := jobMeta.LastExternalId

	for {
		outcome, err := p.ProcessPayloadCutoverBatch(ctx, processId, partitionDate, lastExternalId)

		if err != nil {
			return fmt.Errorf("failed to process payload cutover batch: %w", err)
		}

		if !outcome.ShouldContinue {
			break
		}

		lastExternalId = outcome.NextExternalId
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

	if err != nil {
		return fmt.Errorf("failed to prepare transaction for swapping payload cutover temp table: %w", err)
	}

	defer rollback()

	err = p.queries.SwapV1PayloadPartitionWithTemp(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return fmt.Errorf("failed to swap payload cutover temp table: %w", err)
	}

	err = p.queries.MarkCutoverJobAsCompleted(ctx, tx, pgtype.Date(partitionDate))

	if err != nil {
		return fmt.Errorf("failed to mark cutover job as completed: %w", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("failed to commit swap payload cutover temp table transaction: %w", err)
	}

	return nil
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadCutovers(ctx context.Context) error {
	if !p.externalStoreEnabled {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "payload_store_repository_impl.ProcessPayloadCutovers")
	defer span.End()

	if p.inlineStoreTTL == nil {
		return fmt.Errorf("inline store TTL is not set")
	}

	mostRecentPartitionToOffload := pgtype.Date{
		Time:  time.Now().Add(-1 * (*p.inlineStoreTTL + 2*time.Hour)), // 2 hour offset to limit race conditions and hot-path i/o
		Valid: true,
	}

	partitions, err := p.queries.FindV1PayloadPartitionsBeforeDate(ctx, p.pool, MAX_PARTITIONS_TO_OFFLOAD, mostRecentPartitionToOffload)

	if err != nil {
		return fmt.Errorf("failed to find payload partitions before date %s: %w", mostRecentPartitionToOffload.Time.String(), err)
	}

	processId := uuid.New()

	for _, partition := range partitions {
		p.l.Info().Ctx(ctx).Str("partition", partition.PartitionName).Msg("processing payload cutover for partition")
		err = p.processSinglePartition(ctx, processId, PartitionDate(partition.PartitionDate))

		if err != nil {
			return fmt.Errorf("failed to process partition %s: %w", partition.PartitionName, err)
		}
	}

	return nil
}

func (p *payloadStoreRepositoryImpl) CreateIndexBlock(ctx context.Context, tx pgx.Tx, opts CreateIndexBlockOpts) error {
	return p.queries.CreateOffloadedPayloadIndexBlock(ctx, tx, sqlcv1.CreateOffloadedPayloadIndexBlockParams{
		Payloadinsertedatdate:     pgtype.Date(opts.PartitionDate),
		Blocklowerexternalidbound: opts.BlockLowerExternalIdBound,
		Blockupperexternalidbound: opts.BlockUpperExternalIdBound,
		Indexfilekey:              opts.IndexFileKey,
	})
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (*ExternalIndexFileLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
