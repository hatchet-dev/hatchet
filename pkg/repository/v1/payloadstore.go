package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	Payload    []byte
	TenantId   string
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
	Store(ctx context.Context, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error)
	BulkRetrieve(ctx context.Context, opts ...BulkRetrievePayloadOpts) (map[ExternalPayloadLocationKey][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, opts RetrievePayloadOpts) ([]byte, error)
	BulkRetrieve(ctx context.Context, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	ProcessPayloadWAL(ctx context.Context, partitionNumber int32) (bool, error)
	OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration)
}

type payloadStoreRepositoryImpl struct {
	pool                 *pgxpool.Pool
	l                    *zerolog.Logger
	queries              *sqlcv1.Queries
	externalStoreEnabled bool
	inlineStoreTTL       *time.Duration
	externalStore        ExternalStore
}

func NewPayloadStoreRepository(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	queries *sqlcv1.Queries,
) PayloadStoreRepository {
	return &payloadStoreRepositoryImpl{
		pool:    pool,
		l:       l,
		queries: queries,

		externalStoreEnabled: false,
		inlineStoreTTL:       nil,
		externalStore:        &NoOpExternalStore{},
	}
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	taskIds := make([]int64, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, len(payloads))
	payloadTypes := make([]string, len(payloads))
	inlineContents := make([][]byte, len(payloads))
	offloadAts := make([]pgtype.Timestamptz, len(payloads))
	operations := make([]string, len(payloads))
	tenantIds := make([]pgtype.UUID, len(payloads))
	locations := make([]string, len(payloads))

	for i, payload := range payloads {
		taskIds[i] = payload.Id
		taskInsertedAts[i] = payload.InsertedAt
		payloadTypes[i] = string(payload.Type)
		tenantIds[i] = sqlchelpers.UUIDFromStr(payload.TenantId)
		locations[i] = string(sqlcv1.V1PayloadLocationINLINE)

		if p.externalStoreEnabled {
			offloadAts[i] = pgtype.Timestamptz{Time: payload.InsertedAt.Time.Add(*p.inlineStoreTTL), Valid: true}
		} else {
			offloadAts[i] = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}

		operations[i] = string(sqlcv1.V1PayloadWalOperationCREATE)
		inlineContents[i] = payload.Payload
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
			Operations:         operations,
		})

		if err != nil {
			return fmt.Errorf("failed to write payload WAL: %w", err)
		}
	}

	return err
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, opts RetrievePayloadOpts) ([]byte, error) {
	payloadMap, err := p.BulkRetrieve(ctx, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	payload, ok := payloadMap[opts]

	if !ok {
		return nil, nil
	}

	return payload, nil
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

func (p *payloadStoreRepositoryImpl) offloadToExternal(ctx context.Context, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	// this is only intended to be called from ProcessPayloadWAL, which short-circuits if external store is not enabled
	if !p.externalStoreEnabled {
		return nil, fmt.Errorf("external store not enabled")
	}

	fmt.Println("offloading", len(payloads), "payloads to external store...")
	return p.externalStore.Store(ctx, payloads...)
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadWAL(ctx context.Context, partitionNumber int32) (bool, error) {
	// no need to process the WAL if external store is not enabled
	if !p.externalStoreEnabled {
		return false, nil
	}

	fmt.Println("processing payload WAL...")

	ctx, span := telemetry.NewSpan(ctx, "process-payload-wal")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	if err != nil {
		return false, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	defer rollback()

	pollLimit := 1000
	leaseId := uuid.NewString()

	walRecords, err := p.queries.PollPayloadWALForRecordsToOffload(ctx, tx, sqlcv1.PollPayloadWALForRecordsToOffloadParams{
		Leaseid:         sqlchelpers.UUIDFromStr(leaseId),
		Polllimit:       int32(pollLimit),
		Partitionnumber: partitionNumber,
	})

	fmt.Println("found", len(walRecords), "WAL records to process...")

	hasMoreWALRecords := len(walRecords) == pollLimit

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

	externalStoreOpts := make([]StorePayloadOpts, 0)
	retrieveOptsToPayload := make(map[RetrievePayloadOpts][]byte)

	for opts, payload := range payloads {
		externalStoreOpts = append(externalStoreOpts, StorePayloadOpts{
			Id:         opts.Id,
			InsertedAt: opts.InsertedAt,
			Type:       opts.Type,
			Payload:    payload,
			TenantId:   opts.TenantId.String(),
		})
		retrieveOptsToPayload[opts] = payload
	}

	if err := commit(ctx); err != nil {
		return false, err
	}

	retrieveOptsToStoredKey, err := p.offloadToExternal(ctx, externalStoreOpts...)

	fmt.Println("offloaded", len(retrieveOptsToStoredKey), "payloads to external store...")

	if err != nil {
		return false, err
	}

	offloadAts := make([]pgtype.Timestamptz, 0, len(retrieveOptsToStoredKey))
	ids := make([]int64, 0, len(retrieveOptsToStoredKey))
	insertedAts := make([]pgtype.Timestamptz, 0, len(retrieveOptsToStoredKey))
	types := make([]string, 0, len(retrieveOptsToStoredKey))
	tenantIds := make([]pgtype.UUID, 0, len(retrieveOptsToStoredKey))
	externalLocationKeys := make([]string, 0, len(retrieveOptsToStoredKey))

	for opt := range retrieveOptsToStoredKey {
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
			return false, fmt.Errorf("external location key not found for opts: %+v", opt)
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

	fmt.Println("finalizing", len(ids), "offloaded payloads...")

	err = p.queries.FinalizePayloadOffloads(ctx, tx, sqlcv1.FinalizePayloadOffloadsParams{
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

func (p *payloadStoreRepositoryImpl) OverwriteExternalStore(store ExternalStore, inlineStoreTTL time.Duration) {
	p.externalStoreEnabled = true
	p.inlineStoreTTL = &inlineStoreTTL
	p.externalStore = store
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) BulkRetrieve(ctx context.Context, opts ...BulkRetrievePayloadOpts) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
