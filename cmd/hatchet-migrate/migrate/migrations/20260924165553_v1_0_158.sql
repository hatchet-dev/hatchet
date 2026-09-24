-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION get_v1_monthly_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        -- only drop a monthly partition once every row in it is older than the target date
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date + INTERVAL '1 month') <= targetDate
    ;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_monthly_range_partition(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    monthStartStr varchar;
    nextMonthStartStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(date_trunc('month', targetDate), 'YYYYMMDD') INTO monthStartStr;
    SELECT to_char(date_trunc('month', targetDate) + INTERVAL '1 month', 'YYYYMMDD') INTO nextMonthStartStr;
    SELECT lower(format('%s_%s', targetTableName, monthStartStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES)', newTableName, targetTableName);
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
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, monthStartStr, nextMonthStartStr);
    RETURN 1;
END;
$$;

CREATE TABLE IF NOT EXISTS v1_lookup_table_partitioned (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS v1_lookup_table_external_id_inserted_at_idx ON v1_lookup_table (external_id, inserted_at);
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_lookup_table_external_id_inserted_at_uq') THEN
        ALTER TABLE v1_lookup_table ADD CONSTRAINT v1_lookup_table_external_id_inserted_at_uq UNIQUE USING INDEX v1_lookup_table_external_id_inserted_at_idx;
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    next_month_start TIMESTAMPTZ := date_trunc('month', NOW()) + INTERVAL '1 month';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'v1_lookup_table_attach_bound') THEN
        EXECUTE format('ALTER TABLE v1_lookup_table ADD CONSTRAINT v1_lookup_table_attach_bound CHECK (inserted_at < %L) NOT VALID', next_month_start);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_lookup_table VALIDATE CONSTRAINT v1_lookup_table_attach_bound;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_lookup_table DROP CONSTRAINT IF EXISTS v1_lookup_table_attach_bound;
ALTER TABLE v1_lookup_table DROP CONSTRAINT IF EXISTS v1_lookup_table_external_id_inserted_at_uq;
DROP INDEX IF EXISTS v1_lookup_table_external_id_inserted_at_idx;
DROP TABLE IF EXISTS v1_lookup_table_partitioned;
-- +goose StatementEnd
