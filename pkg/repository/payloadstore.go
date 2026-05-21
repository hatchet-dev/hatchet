package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	ExternalId uuid.UUID
	Type       sqlcv1.V1PayloadType
	Payload    []byte
	TenantId   uuid.UUID
}

type StoreOLAPPayloadOpts struct {
	ExternalId uuid.UUID
	InsertedAt pgtype.Timestamptz
	Payload    []byte
}

type OffloadToExternalStoreOpts struct {
	TenantId   uuid.UUID
	ExternalID uuid.UUID
	InsertedAt pgtype.Timestamptz
	Payload    []byte
}

type RetrievePayloadOpts struct {
	TenantId   uuid.UUID
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	ExternalId uuid.UUID
}

type RetrieveFromExternalByKeyOpt struct {
	Key ExternalPayloadLocationKey
}

type RetrieveFromExternalByIndexFileOpt struct {
	IndexFileKey ExternalIndexFileLocationKey
	ExternalId   uuid.UUID
}

type RetrieveFromExternalMethod int

const (
	RetrieveFromExternalByKey RetrieveFromExternalMethod = iota
	RetrieveFromExternalByIndexFile
)

type RetrieveFromExternalOpts struct {
	Method      RetrieveFromExternalMethod
	ByKey       *RetrieveFromExternalByKeyOpt
	ByIndexFile *RetrieveFromExternalByIndexFileOpt
}

type PayloadLocation string
type ExternalPayloadLocationKey string
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
	Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	RetrieveSingle(ctx context.Context, tx sqlcv1.DBTX, opt RetrievePayloadOpts) ([]byte, error)
	RetrieveFromExternal(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error)
	OverwriteExternalStore(store ExternalStore)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	OLAPDualWritesEnabled() bool
	ExternalCutoverProcessInterval() time.Duration
	InlineStoreTTL() *time.Duration
	ExternalCutoverBatchSize() int32
	ExternalCutoverNumConcurrentOffloads() int32
	ExternalStoreEnabled() bool
	ExternalStore() ExternalStore
	ProcessPayloadCutovers(ctx context.Context) error
	CreateIndexBlock(ctx context.Context, opts CreateIndexBlockOpts) error
}

type payloadStoreRepositoryImpl struct {
	pool                                 *pgxpool.Pool
	l                                    *zerolog.Logger
	queries                              *sqlcv1.Queries
	externalStoreEnabled                 bool
	inlineStoreTTL                       *time.Duration
	externalStore                        ExternalStore
	enablePayloadDualWrites              bool
	enableTaskEventPayloadDualWrites     bool
	enableDagDataPayloadDualWrites       bool
	enableOLAPPayloadDualWrites          bool
	externalCutoverProcessInterval       time.Duration
	externalCutoverBatchSize             int32
	externalCutoverNumConcurrentOffloads int32
}

type PayloadStoreRepositoryOpts struct {
	EnablePayloadDualWrites              bool
	EnableTaskEventPayloadDualWrites     bool
	EnableDagDataPayloadDualWrites       bool
	EnableOLAPPayloadDualWrites          bool
	ExternalCutoverProcessInterval       time.Duration
	ExternalCutoverBatchSize             int32
	ExternalCutoverNumConcurrentOffloads int32
	InlineStoreTTL                       *time.Duration
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
		enablePayloadDualWrites:              opts.EnablePayloadDualWrites,
		enableTaskEventPayloadDualWrites:     opts.EnableTaskEventPayloadDualWrites,
		enableDagDataPayloadDualWrites:       opts.EnableDagDataPayloadDualWrites,
		enableOLAPPayloadDualWrites:          opts.EnableOLAPPayloadDualWrites,
		externalCutoverProcessInterval:       opts.ExternalCutoverProcessInterval,
		externalCutoverBatchSize:             opts.ExternalCutoverBatchSize,
		externalCutoverNumConcurrentOffloads: opts.ExternalCutoverNumConcurrentOffloads,
	}
}

type PayloadUniqueKey struct {
	ID         int64
	InsertedAt pgtype.Timestamptz
	TenantId   uuid.UUID
	Type       sqlcv1.V1PayloadType
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error {
	taskIds := make([]int64, 0, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	payloadTypes := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))
	offloadAts := make([]pgtype.Timestamptz, 0, len(payloads))
	tenantIds := make([]uuid.UUID, 0, len(payloads))
	locations := make([]string, 0, len(payloads))
	externalIds := make([]uuid.UUID, 0, len(payloads))
	externalLocationKeys := make([]string, 0, len(payloads))

	seenPayloadUniqueKeys := make(map[PayloadUniqueKey]struct{})

	sort.Slice(payloads, func(i, j int) bool {
		// sort payloads descending by inserted at to deduplicate operations
		return payloads[i].InsertedAt.Time.After(payloads[j].InsertedAt.Time)
	})

	for _, payload := range payloads {
		tenantId := payload.TenantId
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

func (p *payloadStoreRepositoryImpl) RetrieveSingle(ctx context.Context, tx sqlcv1.DBTX, opt RetrievePayloadOpts) ([]byte, error) {
	if tx == nil {
		tx = p.pool
	}

	optsToPayload, err := p.retrieve(ctx, tx, opt)

	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	if len(optsToPayload) == 0 || err == pgx.ErrNoRows {
		return nil, nil
	}

	return optsToPayload[opt], nil
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

	externalIds := make([]uuid.UUID, len(opts))
	ids := make([]int64, len(opts))
	insertedAts := make([]pgtype.Timestamptz, len(opts))
	types := make([]string, len(opts))
	tenantIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		externalIds[i] = opt.ExternalId
		types[i] = string(opt.Type)
		ids[i] = opt.Id
		insertedAts[i] = opt.InsertedAt
		tenantIds[i] = opt.TenantId
	}

	payloads, err := p.queries.ReadPayloads(ctx, tx, sqlcv1.ReadPayloadsParams{
		Ids:         ids,
		Insertedats: insertedAts,
		Tenantids:   tenantIds,
		Types:       types,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)

	retrieveFromExternalOptsToOpts := make(map[RetrieveFromExternalOpts]RetrievePayloadOpts)
	retrieveFromExternalOpts := make([]RetrieveFromExternalOpts, 0)

	foundKeys := make(map[PayloadUniqueKey]struct{})

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		foundKeys[PayloadUniqueKey{
			ID:         payload.ID,
			InsertedAt: payload.InsertedAt,
			TenantId:   payload.TenantID,
			Type:       payload.Type,
		}] = struct{}{}

		opt := RetrievePayloadOpts{
			Id:         payload.ID,
			InsertedAt: payload.InsertedAt,
			Type:       payload.Type,
			TenantId:   payload.TenantID,
			ExternalId: payload.ExternalID,
		}

		if payload.Location == sqlcv1.V1PayloadLocationEXTERNAL {
			key := ExternalPayloadLocationKey(payload.ExternalLocationKey.String)
			var retrieveFromExternalOpt RetrieveFromExternalOpts

			if strings.HasSuffix(string(key), ".index") {
				retrieveFromExternalOpt = RetrieveFromExternalOpts{
					Method: RetrieveFromExternalByIndexFile,
					ByIndexFile: &RetrieveFromExternalByIndexFileOpt{
						IndexFileKey: ExternalIndexFileLocationKey(key),
						ExternalId:   payload.ExternalID,
					},
				}
			} else {
				retrieveFromExternalOpt = RetrieveFromExternalOpts{
					Method: RetrieveFromExternalByKey,
					ByKey:  &RetrieveFromExternalByKeyOpt{Key: key},
				}
			}

			retrieveFromExternalOptsToOpts[retrieveFromExternalOpt] = opt
			retrieveFromExternalOpts = append(retrieveFromExternalOpts, retrieveFromExternalOpt)
		} else {
			optsToPayload[opt] = payload.InlineContent
		}
	}

	if p.externalStoreEnabled {
		for _, opt := range opts {
			if opt.ExternalId == uuid.Nil {
				continue
			}

			key := PayloadUniqueKey{
				ID:         opt.Id,
				InsertedAt: opt.InsertedAt,
				TenantId:   opt.TenantId,
				Type:       opt.Type,
			}

			if _, found := foundKeys[key]; found {
				continue
			}

			indexFileKey, err := p.queries.GetOffloadedPayloadIndexBlock(ctx, p.pool, sqlcv1.GetOffloadedPayloadIndexBlockParams{
				Insertedatdate: pgtype.Date{Time: opt.InsertedAt.Time.UTC(), Valid: true},
				Externalid:     opt.ExternalId,
			})

			if err != nil {
				continue
			}

			retrieveFromExternalOpt := RetrieveFromExternalOpts{
				Method: RetrieveFromExternalByIndexFile,
				ByIndexFile: &RetrieveFromExternalByIndexFileOpt{
					IndexFileKey: ExternalIndexFileLocationKey(indexFileKey),
					ExternalId:   opt.ExternalId,
				},
			}

			retrieveFromExternalOptsToOpts[retrieveFromExternalOpt] = opt
			retrieveFromExternalOpts = append(retrieveFromExternalOpts, retrieveFromExternalOpt)
		}
	}

	if len(retrieveFromExternalOpts) > 0 {
		externalData, err := p.RetrieveFromExternal(ctx, retrieveFromExternalOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve external payloads: %w", err)
		}

		for retrieveFromExternalOpt, data := range externalData {
			if opt, exists := retrieveFromExternalOptsToOpts[retrieveFromExternalOpt]; exists {
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

func (p *payloadStoreRepositoryImpl) ExternalCutoverNumConcurrentOffloads() int32 {
	return p.externalCutoverNumConcurrentOffloads
}

func (p *payloadStoreRepositoryImpl) ExternalStoreEnabled() bool {
	return p.externalStoreEnabled
}

func (p *payloadStoreRepositoryImpl) ExternalStore() ExternalStore {
	return p.externalStore
}

type BulkCutOverPayload struct {
	TenantID            uuid.UUID
	Id                  int64
	InsertedAt          pgtype.Timestamptz
	ExternalId          uuid.UUID
	Type                sqlcv1.V1PayloadType
	ExternalLocationKey ExternalPayloadLocationKey
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

type payloadStoreCutoverDriver struct {
	impl *payloadStoreRepositoryImpl
}

func (d *payloadStoreCutoverDriver) acquireOrExtendLease(ctx context.Context, tx pgx.Tx, processId uuid.UUID, partitionDate PartitionDate, lastExternalId uuid.UUID) (*cutoverLeaseMetadata, error) {
	leaseExpiresAt := sqlchelpers.TimestamptzFromTime(time.Now().Add(2 * time.Minute))

	lease, err := d.impl.queries.AcquireOrExtendCutoverJobLease(ctx, tx, sqlcv1.AcquireOrExtendCutoverJobLeaseParams{
		Key:            pgtype.Date(partitionDate),
		Leaseprocessid: processId,
		Leaseexpiresat: leaseExpiresAt,
		Lastexternalid: lastExternalId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &cutoverLeaseMetadata{ShouldRun: false, PartitionDate: partitionDate, LeaseProcessId: processId}, nil
		}
		return nil, fmt.Errorf("failed to create initial cutover job lease: %w", err)
	}

	if lease.LeaseProcessID != processId || lease.IsCompleted {
		return &cutoverLeaseMetadata{ShouldRun: false, LastExternalId: lease.LastExternalID, PartitionDate: partitionDate, LeaseProcessId: lease.LeaseProcessID}, nil
	}

	return &cutoverLeaseMetadata{ShouldRun: true, LastExternalId: lease.LastExternalID, PartitionDate: partitionDate, LeaseProcessId: processId}, nil
}

func (d *payloadStoreCutoverDriver) optimizeWindowSize(ctx context.Context, tx sqlcv1.DBTX, partitionDate PartitionDate, candidateBatch int32, lastExternalId uuid.UUID) (*int32, error) {
	return d.impl.OptimizePayloadWindowSize(ctx, tx, partitionDate, candidateBatch, lastExternalId)
}

func (d *payloadStoreCutoverDriver) createRangeChunks(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate, chunkSize, windowSize int32, lastExternalId uuid.UUID) ([]cutoverPayloadRange, error) {
	rows, err := d.impl.queries.CreatePayloadRangeChunks(ctx, tx, sqlcv1.CreatePayloadRangeChunksParams{
		Chunksize:      chunkSize,
		Partitiondate:  pgtype.Date(partitionDate),
		Windowsize:     windowSize,
		Lastexternalid: lastExternalId,
	})
	if err != nil {
		return nil, err
	}
	ranges := make([]cutoverPayloadRange, len(rows))
	for i, r := range rows {
		ranges[i] = cutoverPayloadRange{LowerExternalID: r.LowerExternalID, UpperExternalID: r.UpperExternalID}
	}
	return ranges, nil
}

func (d *payloadStoreCutoverDriver) listPayloadsForOffload(ctx context.Context, partitionDate PartitionDate, lower, upper uuid.UUID, batchSize int32) ([]offloadablePayload, int, error) {
	payloads, err := d.impl.queries.ListPaginatedPayloadsForOffload(ctx, d.impl.pool, sqlcv1.ListPaginatedPayloadsForOffloadParams{
		Partitiondate:  pgtype.Date(partitionDate),
		Lastexternalid: lower,
		Nextexternalid: upper,
		Batchsize:      batchSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list paginated payloads for offload: %w", err)
	}

	result := make([]offloadablePayload, 0, len(payloads))
	for _, p := range payloads {
		if p.Location != sqlcv1.V1PayloadLocationINLINE {
			continue
		}
		externalId := p.ExternalID
		if externalId == uuid.Nil {
			externalId = uuid.New()
		}
		result = append(result, offloadablePayload{
			ExternalID: externalId,
			TenantID:   p.TenantID,
			InsertedAt: p.InsertedAt,
			Content:    p.InlineContent,
		})
	}

	return result, len(payloads), nil
}

func (d *payloadStoreCutoverDriver) createTempTable(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error {
	return d.impl.queries.CreateV1PayloadCutoverTemporaryTable(ctx, tx, pgtype.Date(partitionDate))
}

func (d *payloadStoreCutoverDriver) swapPartition(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error {
	return d.impl.queries.SwapV1PayloadPartitionWithTemp(ctx, tx, pgtype.Date(partitionDate))
}

func (d *payloadStoreCutoverDriver) markJobCompleted(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error {
	return d.impl.queries.MarkCutoverJobAsCompleted(ctx, tx, pgtype.Date(partitionDate))
}

func (d *payloadStoreCutoverDriver) createIndexBlock(ctx context.Context, opts CreateIndexBlockOpts) error {
	return d.impl.CreateIndexBlock(ctx, opts)
}

func (d *payloadStoreCutoverDriver) externalStore() ExternalStore {
	return d.impl.ExternalStore()
}

func (d *payloadStoreCutoverDriver) batchSize() int32 {
	return d.impl.externalCutoverBatchSize
}

func (d *payloadStoreCutoverDriver) numConcurrentOffloads() int32 {
	return d.impl.externalCutoverNumConcurrentOffloads
}

func (d *payloadStoreCutoverDriver) inlineStoreTTL() *time.Duration {
	return d.impl.inlineStoreTTL
}

func (d *payloadStoreCutoverDriver) findPartitions(ctx context.Context, cutoffDate pgtype.Date) ([]cutoverPartition, error) {
	rows, err := d.impl.queries.FindV1PayloadPartitionsBeforeDate(ctx, d.impl.pool, MAX_PARTITIONS_TO_OFFLOAD, cutoffDate)
	if err != nil {
		return nil, err
	}
	partitions := make([]cutoverPartition, len(rows))
	for i, r := range rows {
		partitions[i] = cutoverPartition{PartitionName: r.PartitionName, PartitionDate: PartitionDate(r.PartitionDate)}
	}
	return partitions, nil
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadCutovers(ctx context.Context) error {
	if !p.externalStoreEnabled {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "payload_store_repository_impl.ProcessPayloadCutovers")
	defer span.End()

	job := &cutoverJob{
		pool:       p.pool,
		l:          p.l,
		driver:     &payloadStoreCutoverDriver{impl: p},
		spanPrefix: "payload_store",
	}

	return job.run(ctx)
}

func (p *payloadStoreRepositoryImpl) CreateIndexBlock(ctx context.Context, opts CreateIndexBlockOpts) error {
	_, err := p.queries.CreateOffloadedPayloadIndexBlock(ctx, p.pool, sqlcv1.CreateOffloadedPayloadIndexBlockParams{
		Payloadinsertedatdate:     pgtype.Date(opts.PartitionDate),
		Blocklowerexternalidbound: opts.BlockLowerExternalIdBound,
		Blockupperexternalidbound: opts.BlockUpperExternalIdBound,
		Indexfilekey:              opts.IndexFileKey,
	})

	return err
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, payloads ...OffloadToExternalStoreOpts) (*ExternalIndexFileLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
