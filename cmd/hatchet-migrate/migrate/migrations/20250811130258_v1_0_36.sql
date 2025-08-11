-- +goose Up
-- +goose StatementBegin

UPDATE v1_task
SET priority = 3
WHERE priority > 3;

UPDATE v1_queue_item
SET priority = 3
WHERE priority > 3 AND retry_count = 0;

UPDATE v1_workflow_concurrency_slot
SET priority = 3
WHERE priority > 3;

UPDATE v1_concurrency_slot
SET priority = 3
WHERE priority > 3 AND task_retry_count = 0;

ALTER TABLE v1_task
ADD CONSTRAINT v1_task_priority_user_limit
CHECK (priority >= 1 AND priority <= 3);

ALTER TABLE v1_workflow_concurrency_slot
ADD CONSTRAINT v1_workflow_concurrency_slot_priority_user_limit
CHECK (priority >= 1 AND priority <= 3);

CREATE OR REPLACE FUNCTION create_v1_range_partition(
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
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_weekly_range_partition(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneWeekStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(date_trunc('week', targetDate), 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(date_trunc('week', targetDate) + INTERVAL '1 week', 'YYYYMMDD') INTO targetDatePlusOneWeekStr;
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneWeekStr);
    RETURN 1;
END;
$$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE v1_task DROP CONSTRAINT IF EXISTS v1_task_priority_user_limit;
ALTER TABLE v1_workflow_concurrency_slot DROP CONSTRAINT IF EXISTS v1_workflow_concurrency_slot_priority_user_limit;

-- +goose StatementEnd
