package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// ErrPayloadNotFound indicates that a payload is permanently missing from the external store
// (e.g. it was deleted or never uploaded), as opposed to a transient retrieval failure.
var ErrPayloadNotFound = errors.New("payload not found in external store")

// MissingPayloadsError converts a non-empty list of missing payloads into an error wrapping
// ErrPayloadNotFound. Callers which cannot proceed with a partial result can use this to fail.
func MissingPayloadsError[T any](missing []T) error {
	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("%d payload(s) missing: %w", len(missing), ErrPayloadNotFound)
}

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

func (m RetrieveFromExternalMethod) String() string {
	switch m {
	case RetrieveFromExternalByKey:
		return "key"
	case RetrieveFromExternalByIndexFile:
		return "index_file"
	default:
		return "unknown"
	}
}

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
	// Retrieve returns the payloads which were found, alongside the opts for any payloads which
	// are permanently missing from the store, so callers can proceed with the rest of the batch.
	Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (found map[RetrieveFromExternalOpts][]byte, missing []RetrieveFromExternalOpts, err error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, payloads ...StorePayloadOpts) error
	// Retrieve returns the payloads which were found, alongside the opts for any payloads which
	// are permanently missing from the external store, so callers can proceed with the rest of
	// the batch.
	Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (found map[RetrievePayloadOpts][]byte, missing []RetrievePayloadOpts, err error)
	RetrieveSingle(ctx context.Context, tx sqlcv1.DBTX, opt RetrievePayloadOpts) ([]byte, error)
	RetrieveFromExternal(ctx context.Context, opts ...RetrieveFromExternalOpts) (found map[RetrieveFromExternalOpts][]byte, missing []RetrieveFromExternalOpts, err error)
	OverwriteExternalStore(store ExternalStore)
	DualWritesEnabled() bool
	TaskEventDualWritesEnabled() bool
	DagDataDualWritesEnabled() bool
	OLAPDualWritesEnabled() bool
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
	enablePayloadDualWrites              bool
	enableTaskEventPayloadDualWrites     bool
	enableDagDataPayloadDualWrites       bool
	enableOLAPPayloadDualWrites          bool
	externalCutoverProcessInterval       time.Duration
	externalCutoverBatchSize             int32
	externalCutoverNumConcurrentOffloads int32
	enableWindowSizeOptimization         bool
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
	EnableWindowSizeOptimization         bool
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
		enableWindowSizeOptimization:         opts.EnableWindowSizeOptimization,
	}
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
	taskIds := make([]int64, 0, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	payloadTypes := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))
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
			ID:              payload.Id,
			InsertedAtMicro: payload.InsertedAt.Time.UnixMicro(),
			TenantId:        tenantId,
			Type:            payload.Type,
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

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, []RetrievePayloadOpts, error) {
	if tx == nil {
		tx = p.pool
	}

	return p.retrieve(ctx, tx, opts...)
}

func (p *payloadStoreRepositoryImpl) RetrieveSingle(ctx context.Context, tx sqlcv1.DBTX, opt RetrievePayloadOpts) ([]byte, error) {
	if tx == nil {
		tx = p.pool
	}

	optsToPayload, missing, err := p.retrieve(ctx, tx, opt)

	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	if missingErr := MissingPayloadsError(missing); missingErr != nil {
		return nil, missingErr
	}

	if len(optsToPayload) == 0 || err == pgx.ErrNoRows {
		return nil, nil
	}

	return optsToPayload[opt], nil
}

func (p *payloadStoreRepositoryImpl) RetrieveFromExternal(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, []RetrieveFromExternalOpts, error) {
	if !p.externalStoreEnabled {
		return nil, nil, fmt.Errorf("external store not enabled")
	}

	return p.externalStore.Retrieve(ctx, opts...)
}

func (p *payloadStoreRepositoryImpl) retrieve(ctx context.Context, tx sqlcv1.DBTX, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, []RetrievePayloadOpts, error) {
	if len(opts) == 0 {
		return make(map[RetrievePayloadOpts][]byte), nil, nil
	}

	ids := make([]int64, len(opts))
	insertedAts := make([]pgtype.Timestamptz, len(opts))
	types := make([]string, len(opts))
	tenantIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		ids[i] = opt.Id
		insertedAts[i] = opt.InsertedAt
		types[i] = string(opt.Type)
		tenantIds[i] = opt.TenantId
	}

	payloads, err := p.queries.ReadPayloads(ctx, tx, sqlcv1.ReadPayloadsParams{
		Ids:         ids,
		Insertedats: insertedAts,
		Tenantids:   tenantIds,
		Types:       types,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)
	originalOptsByKey := make(map[PayloadUniqueKey]RetrievePayloadOpts, len(opts))
	for _, opt := range opts {
		originalOptsByKey[payloadUniqueKeyFromRetrieveOpt(opt)] = opt
	}

	retrieveFromExternalOptsToOpts := make(map[RetrieveFromExternalOpts]RetrievePayloadOpts)
	retrieveFromExternalOpts := make([]RetrieveFromExternalOpts, 0)

	foundKeys := make(map[PayloadUniqueKey]struct{})

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		payloadKey := payloadUniqueKeyFromRow(payload)
		foundKeys[payloadKey] = struct{}{}

		opt, ok := originalOptsByKey[payloadKey]
		if !ok {
			opt = RetrievePayloadOpts{
				Id:         payload.ID,
				InsertedAt: payload.InsertedAt,
				Type:       payload.Type,
				TenantId:   payload.TenantID,
				ExternalId: payload.ExternalID,
			}
		}

		if payload.Location == sqlcv1.V1PayloadLocationEXTERNAL {
			key := ExternalPayloadLocationKey(payload.ExternalLocationKey.String)
			var retrieveFromExternalOpt RetrieveFromExternalOpts

			retrieveFromExternalOpt = RetrieveFromExternalOpts{
				Method: RetrieveFromExternalByKey,
				ByKey:  &RetrieveFromExternalByKeyOpt{Key: key},
			}

			retrieveFromExternalOptsToOpts[retrieveFromExternalOpt] = opt
			retrieveFromExternalOpts = append(retrieveFromExternalOpts, retrieveFromExternalOpt)
		} else {
			optsToPayload[opt] = payload.InlineContent
		}
	}

	if p.externalStoreEnabled {
		retrieveIndexBlockExternalIds := make([]uuid.UUID, 0)
		retrieveIndexBlockInsertedAtDates := make([]pgtype.Date, 0)
		externalIdToOpt := make(map[uuid.UUID]RetrievePayloadOpts)

		for _, opt := range opts {
			if opt.ExternalId == uuid.Nil {
				continue
			}

			key := payloadUniqueKeyFromRetrieveOpt(opt)

			if _, found := foundKeys[key]; found {
				continue
			}
			retrieveIndexBlockExternalIds = append(retrieveIndexBlockExternalIds, opt.ExternalId)
			retrieveIndexBlockInsertedAtDates = append(retrieveIndexBlockInsertedAtDates, pgtype.Date{Time: opt.InsertedAt.Time.UTC(), Valid: true})
			externalIdToOpt[opt.ExternalId] = opt
		}

		if len(retrieveIndexBlockExternalIds) > 0 {
			indexFileKeys, err := p.queries.GetOffloadedPayloadIndexBlocks(ctx, tx, sqlcv1.GetOffloadedPayloadIndexBlocksParams{
				Insertedats: retrieveIndexBlockInsertedAtDates,
				Externalids: retrieveIndexBlockExternalIds,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, nil, fmt.Errorf("failed to get offloaded payload index block: %w", err)
			}

			for _, k := range indexFileKeys {
				retrieveFromExternalOpt := RetrieveFromExternalOpts{
					Method: RetrieveFromExternalByIndexFile,
					ByIndexFile: &RetrieveFromExternalByIndexFileOpt{
						IndexFileKey: ExternalIndexFileLocationKey(k.IndexFileKey),
						ExternalId:   k.ExternalID,
					},
				}
				opt, ok := externalIdToOpt[k.ExternalID]

				if !ok {
					p.l.Error().Msg("got index file key for external id that was not requested")
					continue
				}

				retrieveFromExternalOptsToOpts[retrieveFromExternalOpt] = opt
				retrieveFromExternalOpts = append(retrieveFromExternalOpts, retrieveFromExternalOpt)
			}
		}
	}

	missing := make([]RetrievePayloadOpts, 0)

	if len(retrieveFromExternalOpts) > 0 {
		externalData, missingExternal, err := p.RetrieveFromExternal(ctx, retrieveFromExternalOpts...)

		if err != nil {
			return nil, nil, fmt.Errorf("failed to retrieve external payloads: %w", err)
		}

		for retrieveFromExternalOpt, data := range externalData {
			if opt, exists := retrieveFromExternalOptsToOpts[retrieveFromExternalOpt]; exists {
				optsToPayload[opt] = data
			}
		}

		for _, externalOpt := range missingExternal {
			if opt, exists := retrieveFromExternalOptsToOpts[externalOpt]; exists {
				missing = append(missing, opt)
			}
		}
	}

	return optsToPayload, missing, nil
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

func (p *payloadStoreRepositoryImpl) EnableWindowSizeOptimization() bool {
	return p.enableWindowSizeOptimization
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

type DuplicatedExternalIdRow struct {
	ExternalId uuid.UUID
	Count      int64
}

func (p *payloadStoreRepositoryImpl) ValidateNoDuplicateExternalIds(ctx context.Context, tx sqlcv1.DBTX, partitionDate PartitionDate) ([]*DuplicatedExternalIdRow, error) {
	tableName := fmt.Sprintf("v1_payload_%s", partitionDate.String())
	rows, err := tx.Query(
		ctx,
		fmt.Sprintf(
			`
			SELECT external_id, COUNT(*)
			FROM %s
			GROUP BY external_id
			HAVING COUNT(*) > 1
			LIMIT 100
			`,
			tableName,
		),
	)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*DuplicatedExternalIdRow
	for rows.Next() {
		var i DuplicatedExternalIdRow
		if err := rows.Scan(
			&i.ExternalId,
			&i.Count,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
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

	// if the job is running for the first time, check that there aren't any duplicate external ids before proceeding
	if jobMeta.LastExternalId == uuid.Nil {
		connStatementTimeout := 15 * 60 * 1000 // 15 minutes

		conn, release, err := sqlchelpers.AcquireConnectionWithStatementTimeout(ctx, p.pool, p.l, connStatementTimeout)

		if err != nil {
			return fmt.Errorf("failed to acquire connection with statement timeout: %w", err)
		}

		defer release()

		stopLeaseExtension := make(chan struct{})
		leaseExtensionDone := make(chan struct{})

		go func() {
			defer close(leaseExtensionDone)

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-stopLeaseExtension:
					return
				case <-ticker.C:
					leaseTx, leaseCommit, leaseRollback, txErr := sqlchelpers.PrepareTx(ctx, p.pool, p.l)

					if txErr != nil {
						p.l.Error().Err(txErr).Msg("failed to prepare transaction for lease extension during duplicate check")
						continue
					}

					_, txErr = p.acquireOrExtendJobLease(ctx, leaseTx, processId, partitionDate, jobMeta.LastExternalId)

					if txErr != nil {
						leaseRollback()
						p.l.Error().Err(txErr).Msg("failed to extend lease during duplicate check")
						continue
					}

					if txErr = leaseCommit(ctx); txErr != nil {
						leaseRollback()
						p.l.Error().Err(txErr).Msg("failed to commit lease extension during duplicate check")
					}
				}
			}
		}()

		duplicatedExternalIds, err := p.ValidateNoDuplicateExternalIds(ctx, conn, partitionDate)
		close(stopLeaseExtension)
		<-leaseExtensionDone

		if err != nil {
			return fmt.Errorf("failed to validate no duplicate external ids: %w", err)
		}

		if len(duplicatedExternalIds) > 0 {
			var duplicatedIds []string

			for _, row := range duplicatedExternalIds {
				duplicatedIds = append(duplicatedIds, row.ExternalId.String())
			}

			return fmt.Errorf("found duplicate external ids in partition %s. Sampled ids: %s", partitionDate.String(), strings.Join(duplicatedIds, ", "))
		}
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

func (n *NoOpExternalStore) Retrieve(ctx context.Context, opts ...RetrieveFromExternalOpts) (map[RetrieveFromExternalOpts][]byte, []RetrieveFromExternalOpts, error) {
	return nil, nil, fmt.Errorf("external store disabled")
}
