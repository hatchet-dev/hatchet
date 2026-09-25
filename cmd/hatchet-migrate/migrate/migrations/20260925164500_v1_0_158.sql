-- +goose Up
-- +goose StatementBegin

-- workflow_id is a trailing key so workflow-filtered run lists check it on the index
-- entry instead of fetching every tenant row in the window from the heap, while the
-- scan still returns rows in inserted_at order for ORDER BY ... LIMIT. The key is a
-- superset of ix_v1_runs_olap_tenant_ins_at_status, so it also serves unfiltered lists.
--
-- ON ONLY creates the index on the partitioned parent without building it on any
-- existing partition, so this migration is instant and takes no long-lived locks.
-- New partitions get the index automatically via create_v1_range_partition. Existing
-- partitions are left unindexed and age out with retention; to backfill one, run
-- CREATE INDEX CONCURRENTLY on the partition and ALTER INDEX ... ATTACH PARTITION it.

CREATE INDEX IF NOT EXISTS ix_v1_runs_olap_tenant_ins_at_status_wf
    ON ONLY v1_runs_olap (tenant_id, inserted_at DESC, readable_status, workflow_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS ix_v1_runs_olap_tenant_ins_at_status_wf;

-- +goose StatementEnd
