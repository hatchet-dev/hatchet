package sqlcv1

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type CutoverPayloadToInsert struct {
	TenantID            pgtype.UUID
	ID                  int64
	InsertedAt          pgtype.Timestamptz
	ExternalID          pgtype.UUID
	Type                V1PayloadType
	ExternalLocationKey string
}

func InsertCutOverPayloadsIntoTempTable(ctx context.Context, tx DBTX, tableName string, payloads []CutoverPayloadToInsert) (int64, error) {
	tenantIds := make([]pgtype.UUID, 0, len(payloads))
	ids := make([]int64, 0, len(payloads))
	insertedAts := make([]pgtype.Timestamptz, 0, len(payloads))
	externalIds := make([]pgtype.UUID, 0, len(payloads))
	types := make([]string, 0, len(payloads))
	locations := make([]string, 0, len(payloads))
	externalLocationKeys := make([]string, 0, len(payloads))

	for _, payload := range payloads {
		externalIds = append(externalIds, payload.ExternalID)
		tenantIds = append(tenantIds, payload.TenantID)
		ids = append(ids, payload.ID)
		insertedAts = append(insertedAts, payload.InsertedAt)
		types = append(types, string(payload.Type))
		locations = append(locations, string(V1PayloadLocationEXTERNAL))
		externalLocationKeys = append(externalLocationKeys, string(payload.ExternalLocationKey))
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
	err := row.Scan(&copyCount)

	return copyCount, err
}

func ComparePartitionRowCounts(ctx context.Context, tx DBTX, tempPartitionName, sourcePartitionName string) (bool, error) {
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
		return false, err
	}

	return tempPartitionCount == sourcePartitionCount, nil
}
