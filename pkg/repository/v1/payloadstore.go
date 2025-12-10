package v1

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
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
	ExternalId pgtype.UUID
	InsertedAt pgtype.Timestamptz
	Payload    []byte
}

type OffloadToExternalStoreOpts struct {
	TenantId   TenantID
	ExternalID PayloadExternalId
	InsertedAt pgtype.Timestamptz
	Payload    []byte
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
type PayloadExternalId string

type ExternalStore interface {
	Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[PayloadExternalId]ExternalPayloadLocationKey, error)
	Retrieve(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	RetrieveFromExternal(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error)
	OverwriteExternalStore(store ExternalStore)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	OLAPDualWritesEnabled() bool
	ExternalCutoverProcessInterval() time.Duration
	InlineStoreTTL() *time.Duration
	ExternalCutoverBatchSize() int32
	ExternalStoreEnabled() bool
	ExternalStore() ExternalStore
	ProcessPayloadCutovers(ctx context.Context) error
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
	enableImmediateOffloads          bool
}

type PayloadStoreRepositoryOpts struct {
	EnablePayloadDualWrites          bool
	EnableTaskEventPayloadDualWrites bool
	EnableDagDataPayloadDualWrites   bool
	EnableOLAPPayloadDualWrites      bool
	ExternalCutoverProcessInterval   time.Duration
	ExternalCutoverBatchSize         int32
	InlineStoreTTL                   *time.Duration
	EnableImmediateOffloads          bool
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
		inlineStoreTTL:                   opts.InlineStoreTTL,
		externalStore:                    &NoOpExternalStore{},
		enablePayloadDualWrites:          opts.EnablePayloadDualWrites,
		enableTaskEventPayloadDualWrites: opts.EnableTaskEventPayloadDualWrites,
		enableDagDataPayloadDualWrites:   opts.EnableDagDataPayloadDualWrites,
		enableOLAPPayloadDualWrites:      opts.EnableOLAPPayloadDualWrites,
		externalCutoverProcessInterval:   opts.ExternalCutoverProcessInterval,
		externalCutoverBatchSize:         opts.ExternalCutoverBatchSize,
		enableImmediateOffloads:          opts.EnableImmediateOffloads,
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

	if p.enableImmediateOffloads && p.externalStoreEnabled {
		externalOpts := make([]OffloadToExternalStoreOpts, 0, len(payloads))
		payloadIndexMap := make(map[PayloadUniqueKey]int)

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
				TenantId:   TenantID(payload.TenantId),
				ExternalID: PayloadExternalId(payload.ExternalId.String()),
				InsertedAt: payload.InsertedAt,
				Payload:    payload.Payload,
			})
		}

		retrieveOptsToExternalKey, err := p.externalStore.Store(ctx, externalOpts...)
		if err != nil {
			return fmt.Errorf("failed to store in external store: %w", err)
		}

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

			externalKey, exists := retrieveOptsToExternalKey[PayloadExternalId(payload.ExternalId.String())]
			if !exists {
				return fmt.Errorf("external key not found for payload %d", payload.Id)
			}

			taskIds = append(taskIds, payload.Id)
			taskInsertedAts = append(taskInsertedAts, payload.InsertedAt)
			payloadTypes = append(payloadTypes, string(payload.Type))
			tenantIds = append(tenantIds, tenantId)
			locations = append(locations, string(sqlcv1.V1PayloadLocationEXTERNAL))
			inlineContents = append(inlineContents, nil)
			externalIds = append(externalIds, payload.ExternalId)
			externalLocationKeys = append(externalLocationKeys, string(externalKey))
			offloadAts = append(offloadAts, pgtype.Timestamptz{Time: time.Now(), Valid: true})
		}
	} else {
		if p.enableImmediateOffloads {
			p.l.Warn().Msg("immediate offloads enabled but external store is not enabled, skipping immediate offloads")
		}

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

			offloadAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
			if p.inlineStoreTTL != nil {
				offloadAt = pgtype.Timestamptz{Time: time.Now().Add(*p.inlineStoreTTL), Valid: true}
			}

			offloadAts = append(offloadAts, offloadAt)
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

func (p *payloadStoreRepositoryImpl) OverwriteExternalStore(store ExternalStore) {
	p.externalStoreEnabled = true
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

func (p *payloadStoreRepositoryImpl) InlineStoreTTL() *time.Duration {
	return p.inlineStoreTTL
}

func (p *payloadStoreRepositoryImpl) ExternalCutoverBatchSize() int32 {
	return p.externalCutoverBatchSize
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

type PaginationParams struct {
	LastTenantID   pgtype.UUID
	LastInsertedAt pgtype.Timestamptz
	LastID         int64
	LastType       sqlcv1.V1PayloadType
}

type CutoverBatchOutcome struct {
	ShouldContinue bool
	NextPagination PaginationParams
}

type PartitionDate pgtype.Date

func (d PartitionDate) String() string {
	return d.Time.Format("20060102")
}

const MAX_PARTITIONS_TO_OFFLOAD = 14 // two weeks

func (p *payloadStoreRepositoryImpl) ProcessPayloadCutoverBatch(ctx context.Context, processId pgtype.UUID, partitionDate PartitionDate, pagination PaginationParams) (*CutoverBatchOutcome, error) {
	ctx, span := telemetry.NewSpan(ctx, "PayloadStoreRepository.ProcessPayloadCutoverBatch")
	defer span.End()

	tableName := fmt.Sprintf("v1_payload_offload_tmp_%s", partitionDate.String())
	windowSize := p.externalCutoverBatchSize * 10 // 10 chunks per batch

	payloadRanges, err := p.queries.CreatePayloadRangeChunks(ctx, p.pool, sqlcv1.CreatePayloadRangeChunksParams{
		Chunksize:      p.externalCutoverBatchSize,
		Partitiondate:  pgtype.Date(partitionDate),
		Windowsize:     windowSize,
		Lasttenantid:   pagination.LastTenantID,
		Lastinsertedat: pagination.LastInsertedAt,
		Lastid:         pagination.LastID,
		Lasttype:       pagination.LastType,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to create payload range chunks: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &CutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: pagination,
		}, nil
	}

	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	var returnErr error

	externalIdToKey := make(map[PayloadExternalId]ExternalPayloadLocationKey)
	externalIdToPayload := make(map[PayloadExternalId]sqlcv1.ListPaginatedPayloadsForOffloadRow)
	numPayloads := 0

	for _, payloadRange := range payloadRanges {
		wg.Add(1)
		go func(pr sqlcv1.CreatePayloadRangeChunksRow) {
			defer wg.Done()

			payloads, err := p.queries.ListPaginatedPayloadsForOffload(ctx, p.pool, sqlcv1.ListPaginatedPayloadsForOffloadParams{
				Partitiondate:  pgtype.Date(partitionDate),
				Limitparam:     p.externalCutoverBatchSize,
				Lasttenantid:   payloadRange.TenantID,
				Lastinsertedat: payloadRange.InsertedAt,
				Lastid:         payloadRange.ID,
				Lasttype:       payloadRange.Type,
			})

			if err != nil {
				returnErr = multierror.Append(err, fmt.Errorf("failed to list paginated payloads for offload"))
				return
			}

			span.SetAttributes(attribute.Int("num_payloads_read", len(payloads)))

			alreadyExternalPayloads := make(map[PayloadExternalId]ExternalPayloadLocationKey)
			externalIdToPayloadInner := make(map[PayloadExternalId]sqlcv1.ListPaginatedPayloadsForOffloadRow)
			offloadOpts := make([]OffloadToExternalStoreOpts, 0, len(payloads))

			for _, payload := range payloads {
				externalId := PayloadExternalId(payload.ExternalID.String())

				if externalId == "" {
					externalId = PayloadExternalId(uuid.NewString())
				}

				externalIdToPayloadInner[externalId] = *payload
				if payload.Location != sqlcv1.V1PayloadLocationINLINE {
					alreadyExternalPayloads[externalId] = ExternalPayloadLocationKey(payload.ExternalLocationKey)
				} else {
					offloadOpts = append(offloadOpts, OffloadToExternalStoreOpts{
						TenantId:   TenantID(payload.TenantID.String()),
						ExternalID: externalId,
						InsertedAt: payload.InsertedAt,
						Payload:    payload.InlineContent,
					})
				}
			}

			externalIdToKeyInner, err := p.ExternalStore().Store(ctx, offloadOpts...)

			if err != nil {
				returnErr = multierror.Append(err, fmt.Errorf("failed to offload payloads to external store"))
				return
			}

			mu.Lock()
			for externalId, key := range externalIdToKeyInner {
				externalIdToKey[externalId] = key
			}
			for externalId, key := range alreadyExternalPayloads {
				externalIdToKey[externalId] = key
			}
			for externalId, payload := range externalIdToPayloadInner {
				externalIdToPayload[externalId] = payload
			}
			numPayloads += len(payloads)
			mu.Unlock()
		}(*payloadRange)
	}

	wg.Wait()

	if returnErr != nil {
		return nil, returnErr
	}

	payloadsToInsert := make([]sqlcv1.CutoverPayloadToInsert, 0, numPayloads)

	for externalId, key := range externalIdToKey {
		payload := externalIdToPayload[externalId]
		payloadsToInsert = append(payloadsToInsert, sqlcv1.CutoverPayloadToInsert{
			TenantID:            payload.TenantID,
			ID:                  payload.ID,
			InsertedAt:          payload.InsertedAt,
			ExternalID:          sqlchelpers.UUIDFromStr(string(externalId)),
			Type:                payload.Type,
			ExternalLocationKey: string(key),
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for copying offloaded payloads: %w", err)
	}

	defer rollback()

	inserted, err := sqlcv1.InsertCutOverPayloadsIntoTempTable(ctx, tx, tableName, payloadsToInsert)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to copy offloaded payloads into temp table: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &CutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: pagination,
		}, nil
	}

	extendedLease, err := p.acquireOrExtendJobLease(ctx, tx, processId, partitionDate, PaginationParams{
		LastTenantID:   inserted.TenantId,
		LastInsertedAt: inserted.InsertedAt,
		LastID:         inserted.ID,
		LastType:       inserted.Type,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to extend cutover job lease: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	if numPayloads < int(windowSize) {
		return &CutoverBatchOutcome{
			ShouldContinue: false,
			NextPagination: extendedLease.Pagination,
		}, nil
	}

	return &CutoverBatchOutcome{
		ShouldContinue: true,
		NextPagination: extendedLease.Pagination,
	}, nil
}

type CutoverJobRunMetadata struct {
	ShouldRun      bool
	Pagination     PaginationParams
	PartitionDate  PartitionDate
	LeaseProcessId pgtype.UUID
}

func (p *payloadStoreRepositoryImpl) acquireOrExtendJobLease(ctx context.Context, tx pgx.Tx, processId pgtype.UUID, partitionDate PartitionDate, pagination PaginationParams) (*CutoverJobRunMetadata, error) {
	leaseInterval := 2 * time.Minute
	leaseExpiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(leaseInterval))

	lease, err := p.queries.AcquireOrExtendCutoverJobLease(ctx, tx, sqlcv1.AcquireOrExtendCutoverJobLeaseParams{
		Key:            pgtype.Date(partitionDate),
		Leaseprocessid: processId,
		Leaseexpiresat: leaseExpiresAt,
		Lasttenantid:   pagination.LastTenantID,
		Lastinsertedat: pagination.LastInsertedAt,
		Lastid:         pagination.LastID,
		Lasttype:       pagination.LastType,
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
			ShouldRun: false,
			Pagination: PaginationParams{
				LastTenantID:   lease.LastTenantID,
				LastInsertedAt: lease.LastInsertedAt,
				LastID:         lease.LastID,
				LastType:       lease.LastType,
			},
			PartitionDate:  partitionDate,
			LeaseProcessId: lease.LeaseProcessID,
		}, nil
	}

	return &CutoverJobRunMetadata{
		ShouldRun: true,
		Pagination: PaginationParams{
			LastTenantID:   lease.LastTenantID,
			LastInsertedAt: lease.LastInsertedAt,
			LastID:         lease.LastID,
			LastType:       lease.LastType,
		},
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *payloadStoreRepositoryImpl) prepareCutoverTableJob(ctx context.Context, processId pgtype.UUID, partitionDate PartitionDate) (*CutoverJobRunMetadata, error) {
	if p.inlineStoreTTL == nil {
		return nil, fmt.Errorf("inline store TTL is not set")
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	var zeroUuid uuid.UUID

	lease, err := p.acquireOrExtendJobLease(ctx, tx, processId, partitionDate, PaginationParams{
		// placeholder initial type
		LastType:       sqlcv1.V1PayloadTypeDAGINPUT,
		LastTenantID:   sqlchelpers.UUIDFromStr(zeroUuid.String()),
		LastInsertedAt: sqlchelpers.TimestamptzFromTime(time.Unix(0, 0)),
		LastID:         0,
	})

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
		Pagination:     lease.Pagination,
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (p *payloadStoreRepositoryImpl) processSinglePartition(ctx context.Context, processId pgtype.UUID, partitionDate PartitionDate) error {
	ctx, span := telemetry.NewSpan(ctx, "payload_store_repository_impl.processSinglePartition")
	defer span.End()

	jobMeta, err := p.prepareCutoverTableJob(ctx, processId, partitionDate)

	if err != nil {
		return fmt.Errorf("failed to prepare cutover table job: %w", err)
	}

	if !jobMeta.ShouldRun {
		return nil
	}

	pagination := jobMeta.Pagination

	for {
		outcome, err := p.ProcessPayloadCutoverBatch(ctx, processId, partitionDate, pagination)

		if err != nil {
			return fmt.Errorf("failed to process payload cutover batch: %w", err)
		}

		if !outcome.ShouldContinue {
			break
		}

		pagination = outcome.NextPagination
	}

	tempPartitionName := fmt.Sprintf("v1_payload_offload_tmp_%s", partitionDate.String())
	sourcePartitionName := fmt.Sprintf("v1_payload_%s", partitionDate.String())

	countsEqual, err := sqlcv1.ComparePartitionRowCounts(ctx, p.pool, tempPartitionName, sourcePartitionName)

	if err != nil {
		return fmt.Errorf("failed to compare partition row counts: %w", err)
	}

	if !countsEqual {
		return fmt.Errorf("row counts do not match between temp and source partitions for date %s", partitionDate.String())
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 10000)

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
		Time:  time.Now().Add(-1 * *p.inlineStoreTTL),
		Valid: true,
	}

	partitions, err := p.queries.FindV1PayloadPartitionsBeforeDate(ctx, p.pool, MAX_PARTITIONS_TO_OFFLOAD, mostRecentPartitionToOffload)

	if err != nil {
		return fmt.Errorf("failed to find payload partitions before date %s: %w", mostRecentPartitionToOffload.Time.String(), err)
	}

	processId := sqlchelpers.UUIDFromStr(uuid.NewString())

	for _, partition := range partitions {
		p.l.Info().Str("partition", partition.PartitionName).Msg("processing payload cutover for partition")
		err = p.processSinglePartition(ctx, processId, PartitionDate(partition.PartitionDate))

		if err != nil {
			return fmt.Errorf("failed to process partition %s: %w", partition.PartitionName, err)
		}
	}

	return nil
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (map[PayloadExternalId]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) Retrieve(ctx context.Context, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
