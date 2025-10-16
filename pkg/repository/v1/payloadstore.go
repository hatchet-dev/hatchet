package v1

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
)

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	Payload    []byte
	TenantId   string
}

type OffloadToExternalStoreOpts struct {
	*StorePayloadOpts
	OffloadAt time.Time
}

type RetrievePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	TenantId   pgtype.UUID
}

type PayloadLocation string
type ExternalPayloadLocationKey string

type BulkRetrievePayloadOpts struct {
	Keys     []ExternalPayloadLocationKey
	TenantId string
}

type ExternalStore interface {
	Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error)
	BulkRetrieve(ctx context.Context, opts ...BulkRetrievePayloadOpts) (map[ExternalPayloadLocationKey][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	BulkRetrieve(ctx context.Context, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	ProcessPayloadWAL(ctx context.Context, partitionNumber int64) (bool, error)
	ProcessPayloadExternalCutovers(ctx context.Context, partitionNumber int64) (bool, error)
	OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	WALPollLimit() int
	WALProcessInterval() time.Duration
	ExternalCutoverProcessInterval() time.Duration
}

type payloadStoreRepositoryImpl struct {
	pool                             *pgxpool.Pool
	l                                *zerolog.Logger
	queries                          *sqlcv1.Queries
	externalStoreEnabled             bool
	inlineStoreTTL                   *time.Duration
	externalStore                    ExternalStore
	enablePayloadDualWrites          bool
	enableTaskEventPayloadDualWrites bool
	enableDagDataPayloadDualWrites   bool
	walPollLimit                     int
	walProcessInterval               time.Duration
	externalCutoverProcessInterval   time.Duration
}

type PayloadStoreRepositoryOpts struct {
	EnablePayloadDualWrites          bool
	EnableTaskEventPayloadDualWrites bool
	EnableDagDataPayloadDualWrites   bool
	WALPollLimit                     int
	WALProcessInterval               time.Duration
	ExternalCutoverProcessInterval   time.Duration
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

		externalStoreEnabled:             false,
		inlineStoreTTL:                   nil,
		externalStore:                    &NoOpExternalStore{},
		enablePayloadDualWrites:          opts.EnablePayloadDualWrites,
		enableTaskEventPayloadDualWrites: opts.EnableTaskEventPayloadDualWrites,
		enableDagDataPayloadDualWrites:   opts.EnableDagDataPayloadDualWrites,
		walPollLimit:                     opts.WALPollLimit,
		walProcessInterval:               opts.WALProcessInterval,
		externalCutoverProcessInterval:   opts.ExternalCutoverProcessInterval,
	}
}

type PayloadUniqueKey struct {
	ID         int64
	InsertedAt pgtype.Timestamptz
	TenantId   pgtype.UUID
	Type       sqlcv1.V1PayloadType
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	taskIds := make([]int64, 0, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	payloadTypes := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))
	offloadAts := make([]pgtype.Timestamptz, 0, len(payloads))
	tenantIds := make([]pgtype.UUID, 0, len(payloads))
	locations := make([]string, 0, len(payloads))

	seenPayloadUniqueKeys := make(map[PayloadUniqueKey]struct{})

	sort.Slice(payloads, func(i, j int) bool {
		// sort payloads descending by inserted at to deduplicate operations
		return payloads[i].InsertedAt.Time.After(payloads[j].InsertedAt.Time)
	})

	for _, payload := range payloads {
		tenantId := sqlchelpers.UUIDFromStr(payload.TenantId)
		uniqueKey := PayloadUniqueKey{
			ID:         payload.Id,
			InsertedAt: payload.InsertedAt,
			TenantId:   tenantId,
			Type:       payload.Type,
		}

		if _, exists := seenPayloadUniqueKeys[uniqueKey]; exists {
			continue
		}

		seenPayloadUniqueKeys[uniqueKey] = struct{}{}

		taskIds = append(taskIds, payload.Id)
		taskInsertedAts = append(taskInsertedAts, payload.InsertedAt)
		payloadTypes = append(payloadTypes, string(payload.Type))
		tenantIds = append(tenantIds, tenantId)
		locations = append(locations, string(sqlcv1.V1PayloadLocationINLINE))
		inlineContents = append(inlineContents, payload.Payload)

		if p.externalStoreEnabled {
			offloadAts = append(offloadAts, pgtype.Timestamptz{Time: payload.InsertedAt.Time.Add(*p.inlineStoreTTL), Valid: true})
		} else {
			offloadAts = append(offloadAts, pgtype.Timestamptz{Time: time.Now(), Valid: true})
		}
	}

	err := p.queries.WritePayloads(ctx, p.pool, sqlcv1.WritePayloadsParams{
		Ids:            taskIds,
		Insertedats:    taskInsertedAts,
		Types:          payloadTypes,
		Locations:      locations,
		Tenantids:      tenantIds,
		Inlinecontents: inlineContents,
	})

	if err != nil {
		return fmt.Errorf("failed to write payloads: %w", err)
	}

	// only need to write to the WAL if we have an external store configured
	if p.externalStoreEnabled {
		err = p.queries.WritePayloadWAL(ctx, tx, sqlcv1.WritePayloadWALParams{
			Tenantids:          tenantIds,
			Payloadids:         taskIds,
			Payloadinsertedats: taskInsertedAts,
			Payloadtypes:       payloadTypes,
			Offloadats:         offloadAts,
		})

		if err != nil {
			return fmt.Errorf("failed to write payload WAL: %w", err)
		}
	}

	return err
}

func (p *payloadStoreRepositoryImpl) BulkRetrieve(ctx context.Context, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	return p.bulkRetrieve(ctx, p.pool, opts...)
}

func (p *payloadStoreRepositoryImpl) bulkRetrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if len(opts) == 0 {
		return make(map[RetrievePayloadOpts][]byte), nil
	}

	taskIds := make([]int64, len(opts))
	taskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	payloadTypes := make([]string, len(opts))
	tenantIds := make([]pgtype.UUID, len(opts))

	for i, opt := range opts {
		taskIds[i] = opt.Id
		taskInsertedAts[i] = opt.InsertedAt
		payloadTypes[i] = string(opt.Type)
		tenantIds[i] = opt.TenantId
	}

	payloads, err := p.queries.ReadPayloads(ctx, tx, sqlcv1.ReadPayloadsParams{
		Tenantids:   tenantIds,
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)

	externalKeysToOpts := make(map[ExternalPayloadLocationKey]RetrievePayloadOpts)
	retrievePayloadOpts := make([]BulkRetrievePayloadOpts, 0)

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		opts := RetrievePayloadOpts{
			Id:         payload.ID,
			InsertedAt: payload.InsertedAt,
			Type:       payload.Type,
			TenantId:   payload.TenantID,
		}

		if payload.Location == sqlcv1.V1PayloadLocationEXTERNAL {
			key := ExternalPayloadLocationKey(payload.ExternalLocationKey.String)
			externalKeysToOpts[key] = opts
			retrievePayloadOpts = append(retrievePayloadOpts, BulkRetrievePayloadOpts{
				Keys:     []ExternalPayloadLocationKey{key},
				TenantId: opts.TenantId.String(),
			})
		} else {
			optsToPayload[opts] = payload.InlineContent
		}
	}

	if len(retrievePayloadOpts) > 0 {
		externalData, err := p.externalStore.BulkRetrieve(ctx, retrievePayloadOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve external payloads: %w", err)
		}

		for externalKey, data := range externalData {
			if opt, exists := externalKeysToOpts[externalKey]; exists {
				optsToPayload[opt] = data
			}
		}
	}

	return optsToPayload, nil
}

func (p *payloadStoreRepositoryImpl) offloadToExternal(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	ctx, span := telemetry.NewSpan(ctx, "payloadstore.offload_to_external_store")
	defer span.End()

	span.SetAttributes(attribute.Int("payloadstore.offload_to_external_store.num_payloads_to_offload", len(payloads)))

	// this is only intended to be called from ProcessPayloadWAL, which short-circuits if external store is not enabled
	if !p.externalStoreEnabled {
		return nil, fmt.Errorf("external store not enabled")
	}

	return p.externalStore.Store(ctx, payloads...)
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadWAL(ctx context.Context, partitionNumber int64) (bool, error) {
	// no need to process the WAL if external store is not enabled
	if !p.externalStoreEnabled {
		return false, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "payloadstore.process_payload_wal")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	if err != nil {
		return false, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	defer rollback()

	advisoryLockAcquired, err := p.queries.TryAdvisoryLock(ctx, tx, hash(fmt.Sprintf("process-payload-wal-lease-%d", partitionNumber)))

	if err != nil {
		return false, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	if !advisoryLockAcquired {
		return false, nil
	}

	walRecords, err := p.queries.PollPayloadWALForRecordsToReplicate(ctx, tx, sqlcv1.PollPayloadWALForRecordsToReplicateParams{
		Polllimit:       int32(p.walPollLimit),
		Partitionnumber: int32(partitionNumber),
	})

	hasMoreWALRecords := len(walRecords) == p.walPollLimit

	if len(walRecords) == 0 {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	retrieveOpts := make([]RetrievePayloadOpts, len(walRecords))
	retrieveOptsToOffloadAt := make(map[RetrievePayloadOpts]pgtype.Timestamptz)

	for i, record := range walRecords {
		opts := RetrievePayloadOpts{
			Id:         record.PayloadID,
			InsertedAt: record.PayloadInsertedAt,
			Type:       record.PayloadType,
			TenantId:   record.TenantID,
		}

		retrieveOpts[i] = opts
		retrieveOptsToOffloadAt[opts] = record.OffloadAt
	}

	payloads, err := p.bulkRetrieve(ctx, tx, retrieveOpts...)

	if err != nil {
		return false, err
	}

	externalStoreOpts := make([]OffloadToExternalStoreOpts, 0)
	minOffloadAt := time.Now().Add(100 * time.Hour)
	offloadLag := time.Since(minOffloadAt).Seconds()

	attrs := []attribute.KeyValue{
		{
			Key:   "payloadstore.process_payload_wal.payload_wal_offload_partition_number",
			Value: attribute.Int64Value(partitionNumber),
		},
		{
			Key:   "payloadstore.process_payload_wal.payload_wal_offload_count",
			Value: attribute.IntValue(len(retrieveOpts)),
		},
		{
			Key:   "payloadstore.process_payload_wal.payload_wal_offload_lag_seconds",
			Value: attribute.Float64Value(offloadLag),
		},
	}

	span.SetAttributes(attrs...)

	for _, opts := range retrieveOpts {
		offloadAt, ok := retrieveOptsToOffloadAt[opts]

		if !ok {
			return false, fmt.Errorf("offload at not found for opts: %+v", opts)
		}

		externalStoreOpts = append(externalStoreOpts, OffloadToExternalStoreOpts{
			StorePayloadOpts: &StorePayloadOpts{
				Id:         opts.Id,
				InsertedAt: opts.InsertedAt,
				Type:       opts.Type,
				Payload:    payloads[opts],
				TenantId:   opts.TenantId.String(),
			},
			OffloadAt: offloadAt.Time,
		})

		if offloadAt.Time.Before(minOffloadAt) {
			minOffloadAt = offloadAt.Time
		}
	}

	if err := commit(ctx); err != nil {
		return false, err
	}

	retrieveOptsToStoredKey, err := p.offloadToExternal(ctx, externalStoreOpts...)

	if err != nil {
		return false, err
	}

	offloadAts := make([]pgtype.Timestamptz, 0, len(retrieveOptsToStoredKey))
	ids := make([]int64, 0, len(retrieveOptsToStoredKey))
	insertedAts := make([]pgtype.Timestamptz, 0, len(retrieveOptsToStoredKey))
	types := make([]string, 0, len(retrieveOptsToStoredKey))
	tenantIds := make([]pgtype.UUID, 0, len(retrieveOptsToStoredKey))
	externalLocationKeys := make([]string, 0, len(retrieveOptsToStoredKey))

	for _, opt := range retrieveOpts {
		offloadAt, exists := retrieveOptsToOffloadAt[opt]

		if !exists {
			return false, fmt.Errorf("offload at not found for opts: %+v", opt)
		}

		offloadAts = append(offloadAts, offloadAt)
		ids = append(ids, opt.Id)
		insertedAts = append(insertedAts, opt.InsertedAt)
		types = append(types, string(opt.Type))

		key, ok := retrieveOptsToStoredKey[opt]
		if !ok {
			// important: if there's no key here, it's likely because the payloads table did not contain the payload
			// this is okay - it can happen if e.g. a payload partition is dropped before the WAL is processed (not a great situation, but not catastrophic)
			// if this happens, we log an error and set the key to `""` which will allow it to be evicted from the WAL. it'll never cause
			// an update in the payloads table because there won't be a matching row
			p.l.Error().Int64("id", opt.Id).Time("insertedAt", opt.InsertedAt.Time).Msg("external location key not found for opts")
			key = ""
		}

		externalLocationKeys = append(externalLocationKeys, string(key))
		tenantIds = append(tenantIds, opt.TenantId)
	}

	// Second transaction, persist the offload to the db once we've successfully offloaded to the external store
	tx, commit, rollback, err = sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)
	defer rollback()

	if err != nil {
		return false, fmt.Errorf("failed to prepare transaction for offloading: %w", err)
	}

	err = p.queries.SetPayloadExternalKeys(ctx, tx, sqlcv1.SetPayloadExternalKeysParams{
		Ids:                  ids,
		Insertedats:          insertedAts,
		Payloadtypes:         types,
		Offloadats:           offloadAts,
		Tenantids:            tenantIds,
		Externallocationkeys: externalLocationKeys,
	})

	if err != nil {
		return false, err
	}

	if err := commit(ctx); err != nil {
		return false, err
	}

	return hasMoreWALRecords, nil
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadExternalCutovers(ctx context.Context, partitionNumber int64) (bool, error) {
	// no need to cut over if external store is not enabled
	if !p.externalStoreEnabled {
		return false, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "payloadstore.process_payload_external_cutovers")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	if err != nil {
		return false, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	defer rollback()

	advisoryLockAcquired, err := p.queries.TryAdvisoryLock(ctx, tx, hash(fmt.Sprintf("process-payload-cut-overs-lease-%d", partitionNumber)))

	if err != nil {
		return false, fmt.Errorf("failed to acquire advisory lock: %w", err)
	}

	if !advisoryLockAcquired {
		return false, nil
	}

	queueItemsCutOver, err := p.queries.CutOverPayloadsToExternal(ctx, tx, sqlcv1.CutOverPayloadsToExternalParams{
		Polllimit:       int32(p.walPollLimit),  // nolint: gosec
		Partitionnumber: int32(partitionNumber), // nolint: gosec
	})

	if err != nil {
		return false, err
	}

	if queueItemsCutOver == 0 {
		return false, nil
	}

	hasMoreQueueItems := int(queueItemsCutOver) == p.walPollLimit

	if err := commit(ctx); err != nil {
		return false, err
	}

	return hasMoreQueueItems, nil
}

func (p *payloadStoreRepositoryImpl) OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration) {
	p.externalStoreEnabled = true
	p.inlineStoreTTL = &inlineStoreTTL
	p.externalStore = store
}

func (p *payloadStoreRepositoryImpl) DualWritesEnabled() bool {
	return p.enablePayloadDualWrites
}

func (p *payloadStoreRepositoryImpl) TaskEventDualWritesEnabled() bool {
	return p.enableTaskEventPayloadDualWrites
}

func (p *payloadStoreRepositoryImpl) DagDataDualWritesEnabled() bool {
	return p.enableDagDataPayloadDualWrites
}

func (p *payloadStoreRepositoryImpl) WALPollLimit() int {
	return p.walPollLimit
}

func (p *payloadStoreRepositoryImpl) WALProcessInterval() time.Duration {
	return p.walProcessInterval
}

func (p *payloadStoreRepositoryImpl) ExternalCutoverProcessInterval() time.Duration {
	return p.externalCutoverProcessInterval
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) BulkRetrieve(ctx context.Context, opts ...BulkRetrievePayloadOpts) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
