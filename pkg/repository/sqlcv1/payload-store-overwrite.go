package sqlcv1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

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
