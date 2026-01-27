-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS v1_event (
    id bigint GENERATED ALWAYS AS IDENTITY,
    seen_at TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    additional_metadata JSONB,
    scope TEXT,
    triggering_webhook_name TEXT,

    PRIMARY KEY (tenant_id, seen_at, id)
) PARTITION BY RANGE(seen_at);

CREATE INDEX IF NOT EXISTS v1_event_key_idx ON v1_event (tenant_id, key);

CREATE TABLE IF NOT EXISTS v1_event_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (tenant_id, external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

CREATE OR REPLACE FUNCTION v1_event_lookup_table_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table (
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

DROP TRIGGER IF EXISTS v1_event_lookup_table_insert_trigger ON v1_event;

CREATE TRIGGER v1_event_lookup_table_insert_trigger
AFTER INSERT ON v1_event
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_event_lookup_table_insert_function();

CREATE TABLE IF NOT EXISTS v1_event_to_run (
    run_external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,
    filter_id UUID,

    PRIMARY KEY (event_id, event_seen_at, run_external_id)
) PARTITION BY RANGE(event_seen_at);

SELECT create_v1_range_partition('v1_event', DATE 'today');
SELECT create_v1_weekly_range_partition('v1_event_lookup_table', DATE 'today');
SELECT create_v1_weekly_range_partition('v1_event_to_run', DATE 'today');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS v1_event_to_run;
DROP TABLE IF EXISTS v1_event_lookup_table;
DROP TRIGGER IF EXISTS v1_event_lookup_table_insert_trigger ON v1_event;
DROP FUNCTION IF EXISTS v1_event_lookup_table_insert_function();
DROP INDEX IF EXISTS v1_event_key_idx;
DROP TABLE IF EXISTS v1_event;
-- +goose StatementEnd
