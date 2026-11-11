package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/parquet-go/parquet-go"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

const auditLogArchiveRowsPerRowGroup = 50_000

type AuditLogArchiveStore interface {
	PutObject(ctx context.Context, key string, body []byte) error
}

type ArchiveAuditLogPartitionsOpts struct {
	PartitionsEndingBefore time.Time
	LeaseDuration          time.Duration
}

type AuditLogParquetRow struct {
	ID            uuid.UUID `parquet:"id"`
	InsertedAt    time.Time `parquet:"inserted_at,timestamp(microsecond)"`
	TenantID      uuid.UUID `parquet:"tenant_id"`
	ActorType     string    `parquet:"actor_type"`
	ActorID       uuid.UUID `parquet:"actor_id"`
	Action        string    `parquet:"action"`
	CorrelationID *string   `parquet:"correlation_id"`
	ResourceID    string    `parquet:"resource_id"`
	ResourceType  string    `parquet:"resource_type"`
	IpAddress     *string   `parquet:"ip_address"`
	UserAgent     *string   `parquet:"user_agent"`
}

func (r *OLAPRepositoryImpl) SetAuditLogArchiveStore(store AuditLogArchiveStore, rowsPerFile int32) {
	r.auditLogArchiveStore = store
	r.auditLogArchiveRowsPerFile = rowsPerFile
}

func (r *OLAPRepositoryImpl) ArchiveAuditLogPartitions(ctx context.Context, opts ArchiveAuditLogPartitionsOpts) error {
	if r.auditLogArchiveStore == nil {
		return nil
	}

	if r.auditLogArchiveRowsPerFile <= 0 {
		return fmt.Errorf("audit log archive rows per file must be positive, got %d", r.auditLogArchiveRowsPerFile)
	}

	ctx, span := telemetry.NewSpan(ctx, "OLAPRepository.ArchiveAuditLogPartitions")
	defer span.End()

	partitions, err := r.queries.ListAuditLogPartitionsBeforeDate(ctx, r.ddlPool, pgtype.Date{Time: opts.PartitionsEndingBefore, Valid: true})

	if err != nil {
		return fmt.Errorf("failed to list audit log partitions: %w", err)
	}

	processId := uuid.New()

	for _, partition := range partitions {
		cursor, err := r.acquireOrExtendAuditLogArchiveLease(ctx, processId, opts.LeaseDuration, initialAuditLogArchiveCursor(partition))

		if err != nil {
			return fmt.Errorf("failed to acquire archive lease for audit log partition %s: %w", partition.PartitionName, err)
		}

		if cursor == nil {
			r.l.Debug().Ctx(ctx).Msg("audit log archive lease is held by another process, skipping")
			return nil
		}

		r.l.Info().Ctx(ctx).Str("partition", partition.PartitionName).Msg("archiving audit log partition")

		if err := r.uploadAuditLogPartition(ctx, processId, partition, cursor, opts); err != nil {
			return fmt.Errorf("failed to upload audit log partition %s: %w", partition.PartitionName, err)
		}

		if err := r.detachAndDropAuditLogPartition(ctx, partition.PartitionName); err != nil {
			return fmt.Errorf("failed to drop audit log partition %s: %w", partition.PartitionName, err)
		}

		if err := r.queries.MarkAuditLogArchiveAsCompleted(ctx, r.pool.ForShared(), partition.PartitionDate); err != nil {
			return fmt.Errorf("failed to mark audit log partition %s as archived: %w", partition.PartitionName, err)
		}
	}

	return nil
}

// Each file holds one tenant's audit logs. The cursor is saved after each uploaded file, so a restart
// resumes at the next file. If the process dies between an upload and saving the cursor, the retry
// overwrites that file with the same rows, plus possibly more of that tenant's rows that follow them.
func (r *OLAPRepositoryImpl) uploadAuditLogPartition(ctx context.Context, processId uuid.UUID, partition *sqlcv1.ListAuditLogPartitionsBeforeDateRow, cursor *sqlcv1.V1AuditLogArchiveCursor, opts ArchiveAuditLogPartitionsOpts) error {
	for {
		auditLogs, err := r.queries.ListAuditLogsInRangeAfterCursor(ctx, r.readPool.ForShared(), sqlcv1.ListAuditLogsInRangeAfterCursorParams{
			Lowerbound:       partition.LowerBound,
			Upperbound:       partition.UpperBound,
			Cursortenantid:   cursor.LastTenantID,
			Cursorinsertedat: cursor.LastInsertedAt,
			Cursorid:         cursor.LastID,
			Batchsize:        r.auditLogArchiveRowsPerFile,
		})

		if err != nil {
			return fmt.Errorf("failed to list audit logs: %w", err)
		}

		if len(auditLogs) == 0 {
			return nil
		}

		auditLogsByTenant := enumerables.ChunkBy(auditLogs, func(auditLog *sqlcv1.V1AuditLog) uuid.UUID {
			return auditLog.TenantID
		})

		for _, tenantAuditLogs := range auditLogsByTenant {
			cursor, err = r.uploadTenantAuditLogFile(ctx, processId, partition, cursor, tenantAuditLogs, opts.LeaseDuration)

			if err != nil {
				return err
			}
		}

		if len(auditLogs) < int(r.auditLogArchiveRowsPerFile) {
			return nil
		}
	}
}

func (r *OLAPRepositoryImpl) uploadTenantAuditLogFile(ctx context.Context, processId uuid.UUID, partition *sqlcv1.ListAuditLogPartitionsBeforeDateRow, cursor *sqlcv1.V1AuditLogArchiveCursor, tenantAuditLogs []*sqlcv1.V1AuditLog, leaseDuration time.Duration) (*sqlcv1.V1AuditLogArchiveCursor, error) {
	tenantId := tenantAuditLogs[0].TenantID
	fileIndex := nextFileIndexForTenant(cursor, tenantId)

	parquetFile, err := encodeAuditLogsAsParquet(tenantAuditLogs)

	if err != nil {
		return nil, fmt.Errorf("failed to encode audit logs as parquet: %w", err)
	}

	key := auditLogArchiveObjectKey(tenantId, partition.PartitionDate.Time, fileIndex)

	if err := r.auditLogArchiveStore.PutObject(ctx, key, parquetFile); err != nil {
		return nil, fmt.Errorf("failed to upload %s: %w", key, err)
	}

	lastAuditLog := tenantAuditLogs[len(tenantAuditLogs)-1]

	nextCursor, err := r.acquireOrExtendAuditLogArchiveLease(ctx, processId, leaseDuration, &sqlcv1.V1AuditLogArchiveCursor{
		PartitionDate:  partition.PartitionDate,
		NextFileIndex:  fileIndex + 1,
		LastTenantID:   tenantId,
		LastInsertedAt: lastAuditLog.InsertedAt,
		LastID:         lastAuditLog.ID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to extend audit log archive lease: %w", err)
	}

	if nextCursor == nil {
		return nil, fmt.Errorf("audit log archive lease was taken by another process")
	}

	return nextCursor, nil
}

func nextFileIndexForTenant(cursor *sqlcv1.V1AuditLogArchiveCursor, tenantId uuid.UUID) int32 {
	if cursor.LastTenantID == tenantId {
		return cursor.NextFileIndex
	}

	return 0
}

func (r *OLAPRepositoryImpl) detachAndDropAuditLogPartition(ctx context.Context, partitionName string) error {
	conn, release, err := sqlchelpers.AcquireConnectionWithStatementTimeout(ctx, r.ddlPool, r.l, 30*60*1000) // 30 minutes

	if err != nil {
		return err
	}

	defer func() {
		resetCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, resetErr := conn.Exec(resetCtx, "SET lock_timeout = 0"); resetErr != nil {
			r.l.Error().Err(resetErr).Msg("failed to reset lock_timeout on DDL connection")
		}
		release()
	}()

	if _, err := conn.Exec(ctx, "SET lock_timeout = '1min'"); err != nil {
		return fmt.Errorf("failed to set lock_timeout for detach: %w", err)
	}

	// important: DETACH PARTITION CONCURRENTLY cannot run inside a transaction
	_, err = conn.Exec(ctx, fmt.Sprintf("ALTER TABLE v1_audit_log DETACH PARTITION %s CONCURRENTLY", partitionName))

	if isPendingDetach(err) {
		_, err = conn.Exec(ctx, fmt.Sprintf("ALTER TABLE v1_audit_log DETACH PARTITION %s FINALIZE", partitionName))
	}

	if err == nil {
		_, err = conn.Exec(ctx, fmt.Sprintf("DROP TABLE %s", partitionName))
	}

	if isLockNotAvailable(err) {
		return ErrPartitionLockConflict
	}

	return err
}

func initialAuditLogArchiveCursor(partition *sqlcv1.ListAuditLogPartitionsBeforeDateRow) *sqlcv1.V1AuditLogArchiveCursor {
	return &sqlcv1.V1AuditLogArchiveCursor{
		PartitionDate:  partition.PartitionDate,
		NextFileIndex:  0,
		LastTenantID:   uuid.Nil,
		LastInsertedAt: partition.LowerBound,
		LastID:         uuid.Nil,
	}
}

func (r *OLAPRepositoryImpl) acquireOrExtendAuditLogArchiveLease(ctx context.Context, processId uuid.UUID, leaseDuration time.Duration, cursor *sqlcv1.V1AuditLogArchiveCursor) (*sqlcv1.V1AuditLogArchiveCursor, error) {
	lease, err := r.queries.AcquireOrExtendAuditLogArchiveLease(ctx, r.pool.ForShared(), sqlcv1.AcquireOrExtendAuditLogArchiveLeaseParams{
		Partitiondate:  cursor.PartitionDate,
		Leaseprocessid: processId,
		Leaseexpiresat: sqlchelpers.TimestamptzFromTime(time.Now().Add(leaseDuration)),
		Nextfileindex:  cursor.NextFileIndex,
		Lasttenantid:   cursor.LastTenantID,
		Lastinsertedat: cursor.LastInsertedAt,
		Lastid:         cursor.LastID,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return lease, nil
}

func auditLogArchiveObjectKey(tenantId uuid.UUID, partitionDate time.Time, fileIndex int32) string {
	return fmt.Sprintf("%s/%s/%05d.parquet", tenantId, partitionDate.Format("2006/01/02"), fileIndex)
}

func encodeAuditLogsAsParquet(auditLogs []*sqlcv1.V1AuditLog) ([]byte, error) {
	rows := enumerables.Map(auditLogs, func(auditLog *sqlcv1.V1AuditLog) AuditLogParquetRow {
		return AuditLogParquetRow{
			ID:            auditLog.ID,
			InsertedAt:    auditLog.InsertedAt.Time,
			TenantID:      auditLog.TenantID,
			ActorType:     string(auditLog.ActorType),
			ActorID:       auditLog.ActorID,
			Action:        auditLog.Action,
			CorrelationID: sqlchelpers.TextToPtr(auditLog.CorrelationID),
			ResourceID:    auditLog.ResourceID,
			ResourceType:  auditLog.ResourceType,
			IpAddress:     sqlchelpers.TextToPtr(auditLog.IpAddress),
			UserAgent:     sqlchelpers.TextToPtr(auditLog.UserAgent),
		}
	})

	var buffer bytes.Buffer

	writer := parquet.NewGenericWriter[AuditLogParquetRow](&buffer, parquet.Compression(&parquet.Zstd), parquet.MaxRowsPerRowGroup(auditLogArchiveRowsPerRowGroup))

	if _, err := writer.Write(rows); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
