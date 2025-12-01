package v1

import (
	"context"
	"fmt"
	"sort"
	"time"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
)

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	ExternalId pgtype.UUID
	Type       sqlcv1.V1PayloadType
	Payload    []byte
	TenantId   string
}

type StoreOLAPPayloadOpts struct {
	Id         int64
	ExternalId pgtype.UUID
	InsertedAt pgtype.Timestamptz
	Payload    []byte
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
type TenantID string

type ExternalStore interface {
	Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error)
	Retrieve(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	RetrieveFromExternal(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error)
	ProcessPayloadWAL(ctx context.Context, partitionNumber int64, pubBuffer *msgqueue.MQPubBuffer) (bool, error)
	ProcessPayloadExternalCutovers(ctx context.Context, partitionNumber int64) (bool, error)
	OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	OLAPDualWritesEnabled() bool
	WALPollLimit() int
	WALProcessInterval() time.Duration
	WALEnabled() bool
	ExternalCutoverProcessInterval() time.Duration
	ExternalStoreEnabled() bool
	ExternalStore() ExternalStore
	CopyOffloadedPayloadsIntoTempTable(ctx context.Context) (bool, error)
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
	enableOLAPPayloadDualWrites      bool
	walPollLimit                     int
	walProcessInterval               time.Duration
	externalCutoverProcessInterval   time.Duration
	// walEnabled controls the payload storage strategy:
	// - true: Use WAL system (inline + async external offload)
	// - false: Direct external storage (skip inline, save DB space)
	walEnabled bool
}

type PayloadStoreRepositoryOpts struct {
	EnablePayloadDualWrites          bool
	EnableTaskEventPayloadDualWrites bool
	EnableDagDataPayloadDualWrites   bool
	EnableOLAPPayloadDualWrites      bool
	WALPollLimit                     int
	WALProcessInterval               time.Duration
	ExternalCutoverProcessInterval   time.Duration
	// WALEnabled controls the payload storage strategy:
	// - true: Use WAL (Write-Ahead Log) system - store payloads inline in PostgreSQL first,
	//   then asynchronously offload to external storage via WAL processing
	// - false: Skip WAL system - store payloads directly to external storage immediately,
	//   saving database space by not storing inline content in PostgreSQL
	WALEnabled bool
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
		enableOLAPPayloadDualWrites:      opts.EnableOLAPPayloadDualWrites,
		walPollLimit:                     opts.WALPollLimit,
		walProcessInterval:               opts.WALProcessInterval,
		externalCutoverProcessInterval:   opts.ExternalCutoverProcessInterval,
		walEnabled:                       opts.WALEnabled,
	}
}

type PayloadUniqueKey struct {
	ID         int64
	InsertedAt pgtype.Timestamptz
	TenantId   pgtype.UUID
	Type       sqlcv1.V1PayloadType
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	// We store payloads as array in order to use PostgreSQL's UNNEST to batch insert them
	taskIds := make([]int64, 0, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	payloadTypes := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))
	offloadAts := make([]pgtype.Timestamptz, 0, len(payloads))
	tenantIds := make([]pgtype.UUID, 0, len(payloads))
	locations := make([]string, 0, len(payloads))
	externalIds := make([]pgtype.UUID, 0, len(payloads))
	// If the WAL is disabled, we'll store the external location key in the payloads table right away
	externalLocationKeys := make([]string, 0, len(payloads))

	seenPayloadUniqueKeys := make(map[PayloadUniqueKey]struct{})

	sort.Slice(payloads, func(i, j int) bool {
		// sort payloads descending by inserted at to deduplicate operations
		return payloads[i].InsertedAt.Time.After(payloads[j].InsertedAt.Time)
	})

	// Handle WAL_ENABLED logic
	if p.walEnabled {
		// WAL_ENABLED = true: insert into payloads table and WAL table
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
			externalIds = append(externalIds, payload.ExternalId)
			externalLocationKeys = append(externalLocationKeys, "")

			if p.externalStoreEnabled {
				offloadAts = append(offloadAts, pgtype.Timestamptz{Time: payload.InsertedAt.Time.Add(*p.inlineStoreTTL), Valid: true})
			} else {
				offloadAts = append(offloadAts, pgtype.Timestamptz{Time: time.Now(), Valid: true})
			}
		}
	} else {
		// WAL_ENABLED = false: Skip inline, go directly to external store, we will:
		// 1. Prepare external store options for immediate offload
		// 2. Call external store first to get storage keys (S3/GCS URLs)
		// 3. Process results and prepare for PostgreSQL with external keys only
		if !p.externalStoreEnabled {
			return fmt.Errorf("external store must be enabled when WAL is disabled")
		}

		// 1. Prepare external store options for immediate offload
		externalOpts := make([]OffloadToExternalStoreOpts, 0, len(payloads))
		payloadIndexMap := make(map[PayloadUniqueKey]int) // Map to track original payload indices

		for i, payload := range payloads {
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
			payloadIndexMap[uniqueKey] = i

			externalOpts = append(externalOpts, OffloadToExternalStoreOpts{
				StorePayloadOpts: &payload,
				OffloadAt:        time.Now(), // Immediate offload
			})
		}

		// 2. Call external store first to get storage keys (S3/GCS URLs)
		externalKeys, err := p.externalStore.Store(ctx, externalOpts...)
		if err != nil {
			return fmt.Errorf("failed to store in external store: %w", err)
		}

		// 3. Process results and prepare for PostgreSQL with external keys only
		for _, payload := range payloads {
			tenantId := sqlchelpers.UUIDFromStr(payload.TenantId)
			uniqueKey := PayloadUniqueKey{
				ID:         payload.Id,
				InsertedAt: payload.InsertedAt,
				TenantId:   tenantId,
				Type:       payload.Type,
			}

			if _, exists := seenPayloadUniqueKeys[uniqueKey]; !exists {
				continue // Skip if already processed
			}

			// Find the external key for this payload
			retrieveOpts := RetrievePayloadOpts{
				Id:         payload.Id,
				InsertedAt: payload.InsertedAt,
				Type:       payload.Type,
				TenantId:   tenantId,
			}

			externalKey, exists := externalKeys[retrieveOpts]
			if !exists {
				return fmt.Errorf("external key not found for payload %d", payload.Id)
			}

			taskIds = append(taskIds, payload.Id)
			taskInsertedAts = append(taskInsertedAts, payload.InsertedAt)
			payloadTypes = append(payloadTypes, string(payload.Type))
			tenantIds = append(tenantIds, tenantId)
			locations = append(locations, string(sqlcv1.V1PayloadLocationEXTERNAL))
			inlineContents = append(inlineContents, nil) // No inline content
			externalIds = append(externalIds, payload.ExternalId)
			externalLocationKeys = append(externalLocationKeys, string(externalKey))
			offloadAts = append(offloadAts, pgtype.Timestamptz{Time: time.Now(), Valid: true})
		}
	}

	err := p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Ids:                  taskIds,
		Insertedats:          taskInsertedAts,
		Types:                payloadTypes,
		Locations:            locations,
		Tenantids:            tenantIds,
		Inlinecontents:       inlineContents,
		Externalids:          externalIds,
		Externallocationkeys: externalLocationKeys,
	})

	if err != nil {
		return fmt.Errorf("failed to write payloads: %w", err)
	}

	// Only write to WAL if external store is configured AND WAL is enabled
	// When WAL is disabled, payloads are already in external storage, so no WAL needed
	if p.externalStoreEnabled && p.walEnabled {
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

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if tx == nil {
		tx = p.pool
	}

	return p.retrieve(ctx, tx, opts...)
}

func (p *payloadStoreRepositoryImpl) RetrieveFromExternal(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error) {
	if !p.externalStoreEnabled {
		return nil, fmt.Errorf("external store not enabled")
	}

	return p.externalStore.Retrieve(ctx, keys...)
}

func (p *payloadStoreRepositoryImpl) retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
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
	externalKeys := make([]ExternalPayloadLocationKey, 0)

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
			externalKeys = append(externalKeys, key)
		} else {
			optsToPayload[opts] = payload.InlineContent
		}
	}

	if len(externalKeys) > 0 {
		externalData, err := p.RetrieveFromExternal(ctx, externalKeys...)
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

func (p *payloadStoreRepositoryImpl) ProcessPayloadWAL(ctx context.Context, partitionNumber int64, pubBuffer *msgqueue.MQPubBuffer) (bool, error) {
	// no need to process the WAL if external store is not enabled
	if !p.externalStoreEnabled {
		return false, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "payloadstore.process_payload_wal")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 30000)

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

	if err != nil {
		return false, err
	}

	hasMoreWALRecords := len(walRecords) == p.walPollLimit

	if len(walRecords) == 0 {
		return false, nil
	}

	retrieveOpts := make([]RetrievePayloadOpts, len(walRecords))
	retrieveOptsToOffloadAt := make(map[RetrievePayloadOpts]pgtype.Timestamptz)
	retrieveOptsToPayload := make(map[RetrievePayloadOpts][]byte)

	for i, record := range walRecords {
		opts := RetrievePayloadOpts{
			Id:         record.PayloadID,
			InsertedAt: record.PayloadInsertedAt,
			Type:       record.PayloadType,
			TenantId:   record.TenantID,
		}

		retrieveOpts[i] = opts
		retrieveOptsToOffloadAt[opts] = record.OffloadAt

		if record.Location == sqlcv1.V1PayloadLocationINLINE {
			retrieveOptsToPayload[opts] = record.InlineContent
		}
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
				Payload:    retrieveOptsToPayload[opts],
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

	updatedPayloads, err := p.queries.SetPayloadExternalKeys(ctx, tx, sqlcv1.SetPayloadExternalKeysParams{
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

	tenantIdToPayloads := make(map[string][]OLAPPayloadToOffload)

	for _, updatedPayload := range updatedPayloads {
		if updatedPayload == nil || updatedPayload.Type == sqlcv1.V1PayloadTypeTASKEVENTDATA {
			continue
		}

		tenantIdToPayloads[updatedPayload.TenantID.String()] = append(tenantIdToPayloads[updatedPayload.TenantID.String()], OLAPPayloadToOffload{
			ExternalId:          updatedPayload.ExternalID,
			ExternalLocationKey: updatedPayload.ExternalLocationKey.String,
		})
	}

	if err := commit(ctx); err != nil {
		return false, err
	}

	// todo: make this transactionally safe
	// there's no application-level risk here because the worst case if
	// we miss an event is we don't mark the payload as external and there's a bit
	// of disk bloat, but it'd be good to not need to worry about that
	for tenantId, payloads := range tenantIdToPayloads {
		msg, err := OLAPPayloadOffloadMessage(tenantId, payloads)
		if err != nil {
			return false, fmt.Errorf("failed to create OLAP payload offload message: %w", err)
		}
		pubBuffer.Pub(ctx, msgqueue.OLAP_QUEUE, msg, false)
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

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 30000)

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

func (p *payloadStoreRepositoryImpl) OLAPDualWritesEnabled() bool {
	return p.enableOLAPPayloadDualWrites
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

func (p *payloadStoreRepositoryImpl) ExternalStoreEnabled() bool {
	return p.externalStoreEnabled
}

func (p *payloadStoreRepositoryImpl) ExternalStore() ExternalStore {
	return p.externalStore
}

func (p *payloadStoreRepositoryImpl) WALEnabled() bool {
	return p.walEnabled
}

type BulkCutOverPayload struct {
	TenantID            pgtype.UUID
	Id                  int64
	InsertedAt          pgtype.Timestamptz
	ExternalId          pgtype.UUID
	Type                sqlcv1.V1PayloadType
	ExternalLocationKey ExternalPayloadLocationKey
}

func (p *payloadStoreRepositoryImpl) CopyOffloadedPayloadsIntoTempTable(ctx context.Context) (bool, error) {
	partitionDate := time.Now()
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

	fmt.Println("processing payload cutovers for partition date", partitionDate.String())

	if err != nil {
		return false, err
	}

	defer rollback()

	partitionDateStr := partitionDate.Format("20060102")
	err = p.queries.CreateV1PayloadCutoverTemporaryTable(ctx, tx, pgtype.Date{
		Time:  partitionDate,
		Valid: true,
	})

	if err != nil {
		return false, fmt.Errorf("failed to create payload cutover temporary table: %w", err)
	}

	tableName := fmt.Sprintf("v1_payload_offload_tmp_%s", partitionDateStr)
	payloads, err := p.queries.ListPaginatedPayloadsForOffload(ctx, tx, sqlcv1.ListPaginatedPayloadsForOffloadParams{
		Offsetparam: 0,
		Limitparam:  1000,
	})

	if err != nil {
		return false, fmt.Errorf("failed to list payloads for offload: %w", err)
	}

	alreadyExternalPayloads := make(map[RetrievePayloadOpts]ExternalPayloadLocationKey)
	offloadOpts := make([]OffloadToExternalStoreOpts, 0, len(payloads))
	retrieveOptsToExternalId := make(map[RetrievePayloadOpts]string)

	for _, payload := range payloads {
		if payload.Location != sqlcv1.V1PayloadLocationINLINE {
			retrieveOpt := RetrievePayloadOpts{
				Id:         payload.ID,
				InsertedAt: payload.InsertedAt,
				Type:       payload.Type,
				TenantId:   payload.TenantID,
			}

			alreadyExternalPayloads[retrieveOpt] = ExternalPayloadLocationKey(payload.ExternalLocationKey.String)
			retrieveOptsToExternalId[retrieveOpt] = payload.ExternalID.String()
		} else {
			offloadOpts = append(offloadOpts, OffloadToExternalStoreOpts{
				StorePayloadOpts: &StorePayloadOpts{
					Id:         payload.ID,
					InsertedAt: payload.InsertedAt,
					Type:       payload.Type,
					Payload:    payload.InlineContent,
					TenantId:   payload.TenantID.String(),
					ExternalId: payload.ExternalID,
				},
				OffloadAt: time.Now(),
			})
			retrieveOptsToExternalId[RetrievePayloadOpts{
				Id:         payload.ID,
				InsertedAt: payload.InsertedAt,
				Type:       payload.Type,
				TenantId:   payload.TenantID,
			}] = payload.ExternalID.String()
		}
	}

	retrieveOptsToKey, err := p.ExternalStore().Store(ctx, offloadOpts...)

	for r, k := range alreadyExternalPayloads {
		retrieveOptsToKey[r] = k
	}

	rows := make([][]any, 0, len(payloads))
	for r, k := range retrieveOptsToKey {
		// first, offload each payload
		// next, insert into temp table using new keys
		// finally, swap the tables
		// do we need conflict resolution?

		externalId := retrieveOptsToExternalId[r]

		rows = append(rows, []any{
			r.TenantId,
			r.Id,
			r.InsertedAt,
			externalId,
			string(r.Type),
			string(sqlcv1.V1PayloadLocationEXTERNAL),
			k,
			nil,
			time.Now(),
		})
	}

	copyCount, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		[]string{"tenant_id", "id", "inserted_at", "external_id", "type", "location", "external_location_key", "inline_content", "updated_at"},
		pgx.CopyFromRows(rows),
	)

	if err != nil {
		return false, fmt.Errorf("failed to copy offloaded payloads into temp table: %w", err)
	}

	if int(copyCount) != len(payloads) {
		return false, fmt.Errorf("copied payload count %d does not match expected %d", copyCount, len(payloads))
	}

	if err := commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	return copyCount > 0, nil
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) Retrieve(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
