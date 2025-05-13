-- +goose Up
-- +goose StatementBegin
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
    SELECT to_char(date_trunc('week', current_date), 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(date_trunc('week', current_date) + INTERVAL '1 week', 'YYYYMMDD') INTO targetDatePlusOneWeekStr;
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
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
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneWeekStr);
    RETURN 1;
END;
$$;

CREATE TABLE v1_events_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    seen_at TIMESTAMPTZ NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    additional_metadata JSONB,

    PRIMARY KEY (tenant_id, id, seen_at)
) PARTITION BY RANGE(seen_at);

CREATE INDEX v1_events_olap_key_idx ON v1_events_olap (tenant_id, key);

CREATE TABLE v1_event_lookup_table_olap (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (tenant_id, external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

CREATE TABLE v1_event_to_run_olap (
    run_id BIGINT NOT NULL,
    run_inserted_at TIMESTAMPTZ NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (event_id, event_seen_at, run_id, run_inserted_at)
) PARTITION BY RANGE(event_seen_at);


SELECT create_v1_range_partition('v1_events_olap', DATE 'today');
SELECT create_v1_range_partition('v1_event_to_run_olap', DATE 'today');
SELECT create_v1_weekly_range_partition('v1_event_lookup_table_olap'::text, DATE 'today');

CREATE OR REPLACE FUNCTION v1_events_lookup_table_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table_olap (
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        seen_at
    FROM new_rows
    ON CONFLICT (tenant_id, external_id, event_seen_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_olap_insert_trigger
AFTER INSERT ON v1_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_events_lookup_table_olap_insert_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_event_to_run_olap;
DROP TABLE v1_events_olap;
DROP TABLE v1_event_lookup_table_olap;

DROP FUNCTION create_v1_weekly_range_partition(text, date);
DROP FUNCTION v1_events_lookup_table_olap_insert_function();
-- +goose StatementEnd
