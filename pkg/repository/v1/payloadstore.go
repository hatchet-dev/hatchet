package v1

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
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
	OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	OLAPDualWritesEnabled() bool
	ExternalCutoverProcessInterval() time.Duration
	ExternalStoreEnabled() bool
	ExternalStore() ExternalStore
	CopyOffloadedPayloadsIntoTempTable(ctx context.Context) error
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
	externalCutoverProcessInterval   time.Duration
	externalCutoverBatchSize         int32
}

type PayloadStoreRepositoryOpts struct {
	EnablePayloadDualWrites          bool
	EnableTaskEventPayloadDualWrites bool
	EnableDagDataPayloadDualWrites   bool
	EnableOLAPPayloadDualWrites      bool
	ExternalCutoverProcessInterval   time.Duration
	ExternalCutoverBatchSize         int32
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
		externalCutoverProcessInterval:   opts.ExternalCutoverProcessInterval,
		externalCutoverBatchSize:         opts.ExternalCutoverBatchSize,
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
	externalIds := make([]pgtype.UUID, 0, len(payloads))
	externalLocationKeys := make([]string, 0, len(payloads))

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
		externalIds = append(externalIds, payload.ExternalId)
		externalLocationKeys = append(externalLocationKeys, "")
		offloadAts = append(offloadAts, pgtype.Timestamptz{Time: time.Now(), Valid: true})
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

func (p *payloadStoreRepositoryImpl) ExternalCutoverProcessInterval() time.Duration {
	return p.externalCutoverProcessInterval
}

func (p *payloadStoreRepositoryImpl) ExternalStoreEnabled() bool {
	return p.externalStoreEnabled
}

func (p *payloadStoreRepositoryImpl) ExternalStore() ExternalStore {
	return p.externalStore
}

type BulkCutOverPayload struct {
	TenantID            pgtype.UUID
	Id                  int64
	InsertedAt          pgtype.Timestamptz
	ExternalId          pgtype.UUID
	Type                sqlcv1.V1PayloadType
	ExternalLocationKey ExternalPayloadLocationKey
}

func (p *payloadStoreRepositoryImpl) CopyOffloadedPayloadsIntoTempTable(ctx context.Context) error {
	if !p.externalStoreEnabled {
		return nil
	}

	// todo: run this for a configurable date interval (e.g. 2 days ago or something)
	partitionDate := time.Now()
	partitionDateStr := partitionDate.Format("20060102")

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

	if err != nil {
		return err
	}

	hashKey := fmt.Sprintf("payload-cutover-temp-table-lease-%s", partitionDateStr)

	lockAcquired, err := p.queries.TryAdvisoryLock(ctx, tx, hash(hashKey))

	if err != nil {
		rollback()
		return fmt.Errorf("failed to acquire advisory lock for payload cutover temp table: %w", err)
	}

	if !lockAcquired {
		rollback()
		return nil
	}

	jobStatus, err := p.queries.FindLastOffsetForCutoverJob(ctx, p.pool, partitionDateStr)

	var offset int64
	var isCompleted bool

	if err != nil {
		if err == pgx.ErrNoRows {
			offset = 0
			isCompleted = false
		} else {
			rollback()
			return fmt.Errorf("failed to find last offset for cutover job: %w", err)
		}
	} else {
		offset = jobStatus.LastOffset
		isCompleted = jobStatus.IsCompleted
	}

	if isCompleted {
		rollback()
		return nil
	}

	err = p.queries.CreateV1PayloadCutoverTemporaryTable(ctx, tx, pgtype.Date{
		Time:  partitionDate,
		Valid: true,
	})

	if err != nil {
		rollback()
		return fmt.Errorf("failed to create payload cutover temporary table: %w", err)
	}

	if err := commit(ctx); err != nil {
		rollback()
		return fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	for true {
		tx, commit, rollback, err = sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

		if err != nil {
			return fmt.Errorf("failed to prepare transaction for copying offloaded payloads: %w", err)
		}

		defer rollback()

		tableName := fmt.Sprintf("v1_payload_offload_tmp_%s", partitionDateStr)
		payloads, err := p.queries.ListPaginatedPayloadsForOffload(ctx, tx, sqlcv1.ListPaginatedPayloadsForOffloadParams{
			Partitiondate: pgtype.Date{
				Time:  partitionDate,
				Valid: true,
			},
			Offsetparam: offset,
			Limitparam:  p.externalCutoverBatchSize,
		})

		if err != nil {
			return fmt.Errorf("failed to list payloads for offload: %w", err)
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

				alreadyExternalPayloads[retrieveOpt] = ExternalPayloadLocationKey(payload.ExternalLocationKey)
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

		tenantIds := make([]pgtype.UUID, 0, len(payloads))
		ids := make([]int64, 0, len(payloads))
		insertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
		externalIds := make([]pgtype.UUID, 0, len(payloads))
		types := make([]string, 0, len(payloads))
		locations := make([]string, 0, len(payloads))
		externalLocationKeys := make([]string, 0, len(payloads))

		for r, k := range retrieveOptsToKey {
			// qq: do we need conflict resolution here? I think the `COPY` is probably fine
			externalId := retrieveOptsToExternalId[r]

			tenantIds = append(tenantIds, r.TenantId)
			ids = append(ids, r.Id)
			insertedAts = append(insertedAts, r.InsertedAt)
			types = append(types, string(r.Type))
			locations = append(locations, string(sqlcv1.V1PayloadLocationEXTERNAL))
			externalLocationKeys = append(externalLocationKeys, string(k))
			externalIds = append(externalIds, sqlchelpers.UUIDFromStr(externalId))
		}

		row := tx.QueryRow(
			ctx,
			fmt.Sprintf(
				// we unfortunately need to use `INSERT INTO` instead of `COPY` here
				// because we can't have conflict resolution with `COPY`.
				`
				WITH inputs AS (
					SELECT
						UNNEST($1::UUID[]) AS tenant_id,
						UNNEST($2::BIGINT[]) AS id,
						UNNEST($3::TIMESTAMPTZ[]) AS inserted_at,
						UNNEST($4::UUID[]) AS external_id,
						UNNEST($5::TEXT[]) AS type,
						UNNEST($6::TEXT[]) AS location,
						UNNEST($7::TEXT[]) AS external_location_key
				), inserts AS (
					INSERT INTO %s (tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content, updated_at)
					SELECT
						tenant_id,
						id,
						inserted_at,
						external_id,
						type::v1_payload_type,
						location::v1_payload_location,
						external_location_key,
						NULL,
						NOW()
					FROM inputs
					ORDER BY tenant_id, inserted_at, id, type
					ON CONFLICT(tenant_id, id, inserted_at, type) DO NOTHING
					RETURNING *
				)

				SELECT COUNT(*)
				FROM inserts
				`,
				tableName,
			),
			tenantIds,
			ids,
			insertedAts,
			externalIds,
			types,
			locations,
			externalLocationKeys,
		)

		var copyCount int64
		err = row.Scan(&copyCount)

		if err != nil {
			return fmt.Errorf("failed to copy offloaded payloads into temp table: %w", err)
		}

		offset += int64(len(payloads))

		err = p.queries.UpsertLastOffsetForCutoverJob(ctx, tx, sqlcv1.UpsertLastOffsetForCutoverJobParams{
			Key:        partitionDateStr,
			Lastoffset: offset,
		})

		if err != nil {
			return fmt.Errorf("failed to upsert last offset for cutover job: %w", err)
		}

		if err := commit(ctx); err != nil {
			return fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
		}

		if len(payloads) < int(p.externalCutoverBatchSize) {
			break
		}
	}

	tx, commit, rollback, err = sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

	if err != nil {
		return fmt.Errorf("failed to prepare transaction for swapping payload cutover temp table: %w", err)
	}

	defer rollback()

	err = p.queries.SwapV1PayloadPartitionWithTemp(ctx, tx, pgtype.Date{
		Time:  partitionDate,
		Valid: true,
	})

	if err != nil {
		return fmt.Errorf("failed to swap payload cutover temp table: %w", err)
	}

	err = p.queries.MarkCutoverJobAsCompleted(ctx, tx, partitionDateStr)

	if err != nil {
		return fmt.Errorf("failed to mark cutover job as completed: %w", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("failed to commit swap payload cutover temp table transaction: %w", err)
	}

	return nil

}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) Retrieve(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
