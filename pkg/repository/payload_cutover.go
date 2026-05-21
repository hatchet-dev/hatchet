package repository

import (
	"context"
	"errors"
	"fmt"
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

type cutoverLeaseMetadata struct {
	ShouldRun      bool
	LastExternalId uuid.UUID
	PartitionDate  PartitionDate
	LeaseProcessId uuid.UUID
}

type cutoverBatchOutcome struct {
	ShouldContinue bool
	NextExternalId uuid.UUID
}

type cutoverPayloadRange struct {
	LowerExternalID uuid.UUID
	UpperExternalID uuid.UUID
}

type offloadablePayload struct {
	ExternalID uuid.UUID
	TenantID   uuid.UUID
	InsertedAt pgtype.Timestamptz
	Content    []byte
}

type cutoverPartition struct {
	PartitionName string
	PartitionDate PartitionDate
}

type cutoverDriver interface {
	acquireOrExtendLease(ctx context.Context, tx pgx.Tx, processId uuid.UUID, partitionDate PartitionDate, lastExternalId uuid.UUID) (*cutoverLeaseMetadata, error)
	optimizeWindowSize(ctx context.Context, tx sqlcv1.DBTX, partitionDate PartitionDate, candidateBatch int32, lastExternalId uuid.UUID) (*int32, error)
	createRangeChunks(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate, chunkSize, windowSize int32, lastExternalId uuid.UUID) ([]cutoverPayloadRange, error)
	// listPayloadsForOffload returns only inline payloads ready for offloading, plus the total row
	// count across all locations (used to decide whether the batch window is exhausted).
	listPayloadsForOffload(ctx context.Context, partitionDate PartitionDate, lower, upper uuid.UUID, batchSize int32) (offloadable []offloadablePayload, totalCount int, err error)
	createTempTable(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error
	swapPartition(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error
	markJobCompleted(ctx context.Context, tx pgx.Tx, partitionDate PartitionDate) error
	createIndexBlock(ctx context.Context, opts CreateIndexBlockOpts) error
	externalStore() ExternalStore
	batchSize() int32
	numConcurrentOffloads() int32
	inlineStoreTTL() *time.Duration
	findPartitions(ctx context.Context, cutoffDate pgtype.Date) ([]cutoverPartition, error)
}

type cutoverJob struct {
	pool       *pgxpool.Pool
	l          *zerolog.Logger
	driver     cutoverDriver
	spanPrefix string
}

func (j *cutoverJob) run(ctx context.Context) error {
	ttl := j.driver.inlineStoreTTL()
	if ttl == nil {
		return fmt.Errorf("inline store TTL is not set")
	}

	cutoffDate := pgtype.Date{
		Time:  time.Now().UTC().Add(-1 * *ttl),
		Valid: true,
	}

	partitions, err := j.driver.findPartitions(ctx, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to find payload partitions before date %s: %w", cutoffDate.Time.String(), err)
	}

	processId := uuid.New()

	for _, partition := range partitions {
		j.l.Info().Ctx(ctx).Str("partition", partition.PartitionName).Msg("processing payload cutover for partition")
		if err := j.processSinglePartition(ctx, processId, partition.PartitionDate); err != nil {
			return fmt.Errorf("failed to process partition %s: %w", partition.PartitionName, err)
		}
	}

	return nil
}

func (j *cutoverJob) processSinglePartition(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate) error {
	ctx, span := telemetry.NewSpan(ctx, j.spanPrefix+".processSinglePartition")
	defer span.End()

	jobMeta, err := j.prepare(ctx, processId, partitionDate)
	if err != nil {
		return fmt.Errorf("failed to prepare cutover table job: %w", err)
	}

	if !jobMeta.ShouldRun {
		return nil
	}

	lastExternalId := jobMeta.LastExternalId

	for {
		outcome, err := j.processBatch(ctx, processId, partitionDate, lastExternalId)
		if err != nil {
			return fmt.Errorf("failed to process payload cutover batch: %w", err)
		}

		if !outcome.ShouldContinue {
			break
		}

		lastExternalId = outcome.NextExternalId
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l)
	if err != nil {
		return fmt.Errorf("failed to prepare transaction for swapping payload cutover temp table: %w", err)
	}

	defer rollback()

	if err := j.driver.swapPartition(ctx, tx, partitionDate); err != nil {
		return fmt.Errorf("failed to swap payload cutover temp table: %w", err)
	}

	if err := j.driver.markJobCompleted(ctx, tx, partitionDate); err != nil {
		return fmt.Errorf("failed to mark cutover job as completed: %w", err)
	}

	if err := commit(ctx); err != nil {
		return fmt.Errorf("failed to commit swap payload cutover temp table transaction: %w", err)
	}

	return nil
}

func (j *cutoverJob) prepare(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate) (*cutoverLeaseMetadata, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l)
	if err != nil {
		return nil, err
	}

	defer rollback()

	lease, err := j.driver.acquireOrExtendLease(ctx, tx, processId, partitionDate, uuid.Nil)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire or extend cutover job lease: %w", err)
	}

	if !lease.ShouldRun {
		return lease, nil
	}

	if err := j.driver.createTempTable(ctx, tx, partitionDate); err != nil {
		return nil, fmt.Errorf("failed to create payload cutover temporary table: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	return &cutoverLeaseMetadata{
		ShouldRun:      true,
		LastExternalId: lease.LastExternalId,
		PartitionDate:  partitionDate,
		LeaseProcessId: processId,
	}, nil
}

func (j *cutoverJob) processBatch(ctx context.Context, processId uuid.UUID, partitionDate PartitionDate, lastExternalId uuid.UUID) (*cutoverBatchOutcome, error) {
	ctx, span := telemetry.NewSpan(ctx, j.spanPrefix+".processBatch")
	defer span.End()

	batchSize := j.driver.batchSize()
	concurrency := j.driver.numConcurrentOffloads()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for copying offloaded payloads: %w", err)
	}

	defer rollback()

	windowSizePtr, err := j.driver.optimizeWindowSize(ctx, tx, partitionDate, batchSize*concurrency, lastExternalId)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize payload window size: %w", err)
	}

	windowSize := *windowSizePtr

	payloadRanges, err := j.driver.createRangeChunks(ctx, tx, partitionDate, batchSize, windowSize, lastExternalId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to create payload range chunks: %w", err)
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &cutoverBatchOutcome{ShouldContinue: false, NextExternalId: lastExternalId}, nil
	}

	if err = commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit payload range chunks transaction: %w", err)
	}

	mu := sync.Mutex{}
	eg := errgroup.Group{}

	offloadOpts := make([]OffloadToExternalStoreOpts, 0)
	maxExternalId := lastExternalId
	numPayloads := 0

	for _, r := range payloadRanges {
		pr := r
		eg.Go(func() error {
			payloads, totalCount, err := j.driver.listPayloadsForOffload(ctx, partitionDate, pr.LowerExternalID, pr.UpperExternalID, batchSize)
			if err != nil {
				return err
			}

			inner := make([]OffloadToExternalStoreOpts, 0, len(payloads))
			for _, p := range payloads {
				inner = append(inner, OffloadToExternalStoreOpts{
					TenantId:   p.TenantID,
					ExternalID: p.ExternalID,
					InsertedAt: p.InsertedAt,
					Payload:    p.Content,
				})
			}

			mu.Lock()
			offloadOpts = append(offloadOpts, inner...)
			numPayloads += totalCount
			if pr.UpperExternalID.String() > maxExternalId.String() {
				maxExternalId = pr.UpperExternalID
			}
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	blockIndexKey, err := j.driver.externalStore().Store(ctx, offloadOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to offload payloads to external store: %w", err)
	}

	if blockIndexKey != nil {
		if err := j.driver.createIndexBlock(ctx, CreateIndexBlockOpts{
			PartitionDate:             partitionDate,
			BlockLowerExternalIdBound: lastExternalId,
			BlockUpperExternalIdBound: maxExternalId,
			IndexFileKey:              string(*blockIndexKey),
		}); err != nil {
			return nil, fmt.Errorf("failed to create index block: %w", err)
		}
	}

	span.SetAttributes(attribute.Int("num_payloads_read", numPayloads))

	leaseTx, leaseCommit, leaseRollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare transaction for extending cutover job lease: %w", err)
	}

	defer leaseRollback()

	extendedLease, err := j.driver.acquireOrExtendLease(ctx, leaseTx, processId, partitionDate, maxExternalId)
	if err != nil {
		return nil, fmt.Errorf("failed to extend cutover job lease: %w", err)
	}

	if !extendedLease.ShouldRun {
		return nil, fmt.Errorf("lease for partition %s was taken by another process during batch processing", partitionDate.String())
	}

	if err := leaseCommit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit copy offloaded payloads transaction: %w", err)
	}

	if numPayloads < int(windowSize) {
		return &cutoverBatchOutcome{ShouldContinue: false, NextExternalId: extendedLease.LastExternalId}, nil
	}

	return &cutoverBatchOutcome{ShouldContinue: true, NextExternalId: extendedLease.LastExternalId}, nil
}
