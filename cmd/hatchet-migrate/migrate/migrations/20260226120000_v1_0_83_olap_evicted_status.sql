-- +goose Up
-- +goose NO TRANSACTION

ALTER TYPE v1_readable_status_olap ADD VALUE IF NOT EXISTS 'EVICTED';

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_v1_olap_partition_with_date_and_status(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT format('%s_%s', targetTableName, targetDateStr) INTO newTableName;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        EXECUTE format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES) PARTITION BY LIST (readable_status)', newTableName, targetTableName);
    END IF;

    PERFORM create_v1_partition_with_status(newTableName, 'QUEUED');
    PERFORM create_v1_partition_with_status(newTableName, 'RUNNING');
    PERFORM create_v1_partition_with_status(newTableName, 'COMPLETED');
    PERFORM create_v1_partition_with_status(newTableName, 'CANCELLED');
    PERFORM create_v1_partition_with_status(newTableName, 'FAILED');
    PERFORM create_v1_partition_with_status(newTableName, 'EVICTED');

    -- If it's not already attached, attach the partition
    IF NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = newTableName::regclass) THEN
        EXECUTE format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    END IF;

    RETURN 1;
END;
$$;
-- +goose StatementEnd

WITH partitions AS (
    SELECT inhrelid::regclass::text AS partition_name
    FROM pg_inherits
    WHERE inhparent IN (
        'v1_tasks_olap'::regclass,
        'v1_dags_olap'::regclass,
        'v1_runs_olap'::regclass
    )
)
SELECT create_v1_partition_with_status(partition_name, 'EVICTED')
FROM partitions;

ALTER TABLE v1_task_events_olap ADD COLUMN IF NOT EXISTS durable_invocation_count INT NOT NULL DEFAULT 0;

ANALYZE v1_tasks_olap;
ANALYZE v1_dags_olap;
ANALYZE v1_runs_olap;

-- +goose Down
-- +goose NO TRANSACTION

ALTER TABLE v1_task_events_olap DROP COLUMN IF EXISTS durable_invocation_count;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION create_v1_olap_partition_with_date_and_status(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT format('%s_%s', targetTableName, targetDateStr) INTO newTableName;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        EXECUTE format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES) PARTITION BY LIST (readable_status)', newTableName, targetTableName);
    END IF;

    PERFORM create_v1_partition_with_status(newTableName, 'QUEUED');
    PERFORM create_v1_partition_with_status(newTableName, 'RUNNING');
    PERFORM create_v1_partition_with_status(newTableName, 'COMPLETED');
    PERFORM create_v1_partition_with_status(newTableName, 'CANCELLED');
    PERFORM create_v1_partition_with_status(newTableName, 'FAILED');

    IF NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = newTableName::regclass) THEN
        EXECUTE format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    END IF;

    RETURN 1;
END;
$$;
-- +goose StatementEnd

-- NOTE: Postgres does not support removing enum values.
-- The 'EVICTED' value in v1_readable_status_olap cannot be reverted.
-- Any EVICTED partitions created by this migration would need to be merged/dropped separately.
