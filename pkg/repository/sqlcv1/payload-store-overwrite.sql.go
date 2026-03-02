package sqlcv1

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CutoverPayloadToInsert struct {
	TenantID            uuid.UUID
	ID                  int64
	InsertedAt          pgtype.Timestamptz
	ExternalID          uuid.UUID
	Type                V1PayloadType
	ExternalLocationKey string
	InlineContent       []byte
	Location            V1PayloadLocation
}

type InsertCutOverPayloadsIntoTempTableRow struct {
	TenantId   uuid.UUID
	ID         int64
	InsertedAt pgtype.Timestamptz
	Type       V1PayloadType
}

func InsertCutOverPayloadsIntoTempTable(ctx context.Context, tx DBTX, tableName string, payloads []CutoverPayloadToInsert) (*InsertCutOverPayloadsIntoTempTableRow, error) {
	tenantIds := make([]uuid.UUID, 0, len(payloads))
	ids := make([]int64, 0, len(payloads))
	insertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	externalIds := make([]uuid.UUID, 0, len(payloads))
	types := make([]string, 0, len(payloads))
	locations := make([]string, 0, len(payloads))
	externalLocationKeys := make([]string, 0, len(payloads))
	inlineContents := make([][]byte, 0, len(payloads))

	for _, payload := range payloads {
		externalIds = append(externalIds, payload.ExternalID)
		tenantIds = append(tenantIds, payload.TenantID)
		ids = append(ids, payload.ID)
		insertedAts = append(insertedAts, payload.InsertedAt)
		types = append(types, string(payload.Type))
		locations = append(locations, string(payload.Location))
		externalLocationKeys = append(externalLocationKeys, string(payload.ExternalLocationKey))
		inlineContents = append(inlineContents, payload.InlineContent)
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
						UNNEST($7::TEXT[]) AS external_location_key,
						UNNEST($8::JSONB[]) AS inline_content
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
						inline_content,
						NOW()
					FROM inputs
					ORDER BY tenant_id, inserted_at, id, type
					ON CONFLICT(tenant_id, id, inserted_at, type) DO NOTHING
				)

				SELECT tenant_id, inserted_at, id, type
				FROM inputs
				ORDER BY tenant_id DESC, inserted_at DESC, id DESC, type DESC
				LIMIT 1
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
		inlineContents,
	)

	var insertRow InsertCutOverPayloadsIntoTempTableRow

	err := row.Scan(
		&insertRow.TenantId,
		&insertRow.InsertedAt,
		&insertRow.ID,
		&insertRow.Type,
	)

	return &insertRow, err
}

type PartitionRowCounts struct {
	SourcePartitionCount int64
	TempPartitionCount   int64
}

func ComparePartitionRowCounts(ctx context.Context, tx DBTX, tempPartitionName, sourcePartitionName string) (*PartitionRowCounts, error) {
	row := tx.QueryRow(
		ctx,
		fmt.Sprintf(
			`
				SELECT
					(SELECT COUNT(*) FROM %s) AS temp_partition_count,
					(SELECT COUNT(*) FROM %s) AS source_partition_count
			`,
			tempPartitionName,
			sourcePartitionName,
		),
	)

	var tempPartitionCount int64
	var sourcePartitionCount int64

	err := row.Scan(&tempPartitionCount, &sourcePartitionCount)

	if err != nil {
		return nil, err
	}

	return &PartitionRowCounts{
		SourcePartitionCount: sourcePartitionCount,
		TempPartitionCount:   tempPartitionCount,
	}, nil
}

const findV1PayloadPartitionsBeforeDate = `-- name: findV1PayloadPartitionsBeforeDate :many
WITH partitions AS (
    SELECT
        child.relname::text AS partition_name,
        SUBSTRING(pg_get_expr(child.relpartbound, child.oid) FROM 'FROM \(''([^'']+)')::DATE AS lower_bound,
        SUBSTRING(pg_get_expr(child.relpartbound, child.oid) FROM 'TO \(''([^'']+)')::DATE AS upper_bound
    FROM pg_inherits
    JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child ON pg_inherits.inhrelid = child.oid
    WHERE parent.relname = 'v1_payload'
    ORDER BY child.relname DESC
	LIMIT $1::INTEGER
)

SELECT partition_name, lower_bound AS partition_date
FROM partitions
WHERE lower_bound <= $2::DATE
`

type FindV1PayloadPartitionsBeforeDateRow struct {
	PartitionName string      `json:"partition_name"`
	PartitionDate pgtype.Date `json:"partition_date"`
}

func (q *Queries) FindV1PayloadPartitionsBeforeDate(ctx context.Context, db DBTX, maxPartitionsToProcess int32, date pgtype.Date) ([]*FindV1PayloadPartitionsBeforeDateRow, error) {
	rows, err := db.Query(ctx, findV1PayloadPartitionsBeforeDate,
		maxPartitionsToProcess,
		date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*FindV1PayloadPartitionsBeforeDateRow
	for rows.Next() {
		var i FindV1PayloadPartitionsBeforeDateRow
		if err := rows.Scan(
			&i.PartitionName,
			&i.PartitionDate,
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
