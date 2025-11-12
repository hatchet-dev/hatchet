-- +goose Up
-- +goose NO TRANSACTION
-- +goose StatementBegin
BEGIN;

CREATE TABLE v1_event_lookup_table_olap_new (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

WITH existing_partitions AS (
    SELECT
        child.relname AS partition_name,
        regexp_replace(
            pg_get_expr(child.relpartbound, child.oid),
            '.*FROM \(''([^'']+)''\) TO \(''([^'']+)''\).*',
            '\1'
        )::timestamp AS range_start
    FROM pg_inherits
    JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child ON pg_inherits.inhrelid = child.oid
    WHERE parent.relname = 'v1_event_lookup_table_olap'
    ORDER BY range_start
)

SELECT create_v1_weekly_range_partition('v1_event_lookup_table_olap_new'::text, range_start::DATE)
FROM existing_partitions;

CREATE OR REPLACE FUNCTION v1_event_lookup_table_olap_new_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table_olap_new (
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    )
    SELECT
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    FROM new_rows
    ON CONFLICT DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_olap_new_insert_trigger
AFTER INSERT ON v1_event_lookup_table_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_event_lookup_table_olap_new_insert_function();

COMMIT;

-- +goose StatementEnd
-- +goose StatementBegin

INSERT INTO v1_event_lookup_table_olap_new (tenant_id, external_id, event_id, event_seen_at)
SELECT tenant_id, external_id, event_id, event_seen_at
FROM v1_event_lookup_table_olap
ON CONFLICT DO NOTHING;

-- +goose StatementEnd
-- +goose StatementBegin

BEGIN;

DROP TRIGGER v1_event_lookup_table_olap_new_insert_trigger ON v1_event_lookup_table_olap;
DROP FUNCTION v1_event_lookup_table_olap_new_insert_function();
DROP TABLE v1_event_lookup_table_olap;
ALTER TABLE v1_event_lookup_table_olap_new RENAME TO v1_event_lookup_table_olap;
ALTER INDEX v1_event_lookup_table_olap_new_pkey RENAME TO v1_event_lookup_table_olap_pkey;

DO $$
DECLARE
    partition_record RECORD;
    new_name TEXT;
BEGIN
    FOR partition_record IN
        SELECT tablename
        FROM pg_tables
        WHERE tablename LIKE 'v1_event_lookup_table_olap_new_%'
        AND schemaname = 'public'
    LOOP
        new_name := REPLACE(partition_record.tablename, 'v1_event_lookup_table_olap_new_', 'v1_event_lookup_table_olap_');
        EXECUTE format('ALTER TABLE %I RENAME TO %I', partition_record.tablename, new_name);
        RAISE NOTICE 'Renamed % to %', partition_record.tablename, new_name;
    END LOOP;
END $$;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose NO TRANSACTION
-- +goose StatementBegin
BEGIN;

CREATE TABLE v1_event_lookup_table_olap_new (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (tenant_id, external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

WITH existing_partitions AS (
    SELECT
        child.relname AS partition_name,
        regexp_replace(
            pg_get_expr(child.relpartbound, child.oid),
            '.*FROM \(''([^'']+)''\) TO \(''([^'']+)''\).*',
            '\1'
        )::timestamp AS range_start
    FROM pg_inherits
    JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
    JOIN pg_class child ON pg_inherits.inhrelid = child.oid
    WHERE parent.relname = 'v1_event_lookup_table_olap'
    ORDER BY range_start
)

SELECT create_v1_weekly_range_partition('v1_event_lookup_table_olap_new'::text, range_start::DATE)
FROM existing_partitions;

CREATE OR REPLACE FUNCTION v1_event_lookup_table_olap_new_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table_olap_new (
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    )
    SELECT
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    FROM new_rows
    ON CONFLICT DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_olap_new_insert_trigger
AFTER INSERT ON v1_event_lookup_table_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_event_lookup_table_olap_new_insert_function();

COMMIT;

-- +goose StatementEnd
-- +goose StatementBegin

INSERT INTO v1_event_lookup_table_olap_new (tenant_id, external_id, event_id, event_seen_at)
SELECT tenant_id, external_id, event_id, event_seen_at
FROM v1_event_lookup_table_olap
ON CONFLICT DO NOTHING;

-- +goose StatementEnd
-- +goose StatementBegin

BEGIN;

DROP TRIGGER v1_event_lookup_table_olap_new_insert_trigger ON v1_event_lookup_table_olap;
DROP FUNCTION v1_event_lookup_table_olap_new_insert_function();
DROP TABLE v1_event_lookup_table_olap;
ALTER TABLE v1_event_lookup_table_olap_new RENAME TO v1_event_lookup_table_olap;
ALTER INDEX v1_event_lookup_table_olap_new_pkey RENAME TO v1_event_lookup_table_olap_pkey;

DO $$
DECLARE
    partition_record RECORD;
    new_name TEXT;
BEGIN
    FOR partition_record IN
        SELECT tablename
        FROM pg_tables
        WHERE tablename LIKE 'v1_event_lookup_table_olap_new_%'
        AND schemaname = 'public'
    LOOP
        new_name := REPLACE(partition_record.tablename, 'v1_event_lookup_table_olap_new_', 'v1_event_lookup_table_olap_');
        EXECUTE format('ALTER TABLE %I RENAME TO %I', partition_record.tablename, new_name);
        RAISE NOTICE 'Renamed % to %', partition_record.tablename, new_name;
    END LOOP;
END $$;

COMMIT;
-- +goose StatementEnd
