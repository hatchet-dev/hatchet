-- +goose Up
-- +goose StatementBegin
DROP TRIGGER v1_event_lookup_table_insert_trigger ON v1_event;
DROP FUNCTION v1_event_lookup_table_insert_function();

DROP TABLE v1_event_lookup_table;
TRUNCATE TABLE v1_event_to_run;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE v1_event_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id, event_seen_at)
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
    ON CONFLICT (external_id, event_seen_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_insert_trigger
AFTER INSERT ON v1_event
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_event_lookup_table_insert_function();
-- +goose StatementEnd
