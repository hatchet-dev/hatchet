package olapagg

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func CompactionLockKey(aggregateTable string, partition int) int64 {
	return sqlchelpers.AdvisoryLockKey(fmt.Sprintf("olap-aggregate-compaction:%s:%d", aggregateTable, partition))
}

// Aggregate is one aggregate table and the query that compacts its delta table.
type Aggregate struct {
	// Table is the aggregate table, used to find its partitions.
	Table string

	compact func(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, p compactParams) (int64, error)
}

type compactParams struct {
	partitionNumber int32
	batchSize       int32
	retainFrom      pgtype.Timestamptz
}

var Aggregates = []Aggregate{
	{
		Table: "v1_agg_task_statuses_minute_olap",
		compact: func(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, p compactParams) (int64, error) {
			return q.CompactTaskStatusesMinuteDeltas(ctx, db, sqlcv1.CompactTaskStatusesMinuteDeltasParams{
				Partitionnumber: p.partitionNumber, Batchsize: p.batchSize, Retainfrom: p.retainFrom,
			})
		},
	},
	{
		Table: "v1_agg_task_events_minute_olap",
		compact: func(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, p compactParams) (int64, error) {
			return q.CompactTaskEventsMinuteDeltas(ctx, db, sqlcv1.CompactTaskEventsMinuteDeltasParams{
				Partitionnumber: p.partitionNumber, Batchsize: p.batchSize, Retainfrom: p.retainFrom,
			})
		},
	},
	{
		Table: "v1_agg_events_minute_olap",
		compact: func(ctx context.Context, q *sqlcv1.Queries, db sqlcv1.DBTX, p compactParams) (int64, error) {
			return q.CompactEventsMinuteDeltas(ctx, db, sqlcv1.CompactEventsMinuteDeltasParams{
				Partitionnumber: p.partitionNumber, Batchsize: p.batchSize, Retainfrom: p.retainFrom,
			})
		},
	},
}
