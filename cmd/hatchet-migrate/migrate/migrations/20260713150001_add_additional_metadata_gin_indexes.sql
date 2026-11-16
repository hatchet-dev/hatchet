-- +goose Up
-- +goose StatementBegin

-- GIN indexes for additional_metadata containment filters (@> / @> ANY) on the OLAP
-- run/task tables.
--
-- ON ONLY creates the index on the partitioned parent without building it on any
-- existing partition, so this migration is instant and takes no long-lived locks.
-- The parent index stays INVALID until every partition has an attached child index,
-- which is harmless: the planner uses each partition's own index independently.
--
-- New partitions get the index automatically: create_v1_range_partition creates
-- partitions with LIKE ... INCLUDING INDEXES and ATTACH PARTITION, which builds the
-- child index (on an empty table) and attaches it to this parent index. Existing
-- partitions are intentionally left unindexed and age out with retention; to backfill
-- one, run CREATE INDEX CONCURRENTLY on the partition and ALTER INDEX ... ATTACH
-- PARTITION it.
--
-- jsonb_path_ops only supports @> (not ? / ?| / ?&), which is all the queries use,
-- and is smaller and faster to search than the default jsonb_ops.

CREATE INDEX IF NOT EXISTS ix_v1_tasks_olap_additional_metadata_gin
    ON ONLY v1_tasks_olap USING gin (additional_metadata jsonb_path_ops);

CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_additional_metadata_gin
    ON ONLY v1_runs_olap USING gin (additional_metadata jsonb_path_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ix_v1_tasks_olap_additional_metadata_gin;
DROP INDEX IF EXISTS ix_v1_runs_olap_additional_metadata_gin;

-- +goose StatementEnd
