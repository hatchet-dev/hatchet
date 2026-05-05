-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ADD COLUMN result_payload_external_id UUID;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    batch_size INT := 100; -- small batches, since each durable task can have many event log entries
    rows_updated INT;
    last_task_id BIGINT := -1;
    last_inserted_at TIMESTAMPTZ := '1970-01-01T00:00:00Z';
BEGIN
    LOOP
        WITH batch_keys AS (
            SELECT DISTINCT durable_task_id, durable_task_inserted_at
            FROM v1_durable_event_log_entry
            WHERE result_payload_external_id IS NULL
              AND (durable_task_id, durable_task_inserted_at) > (last_task_id, last_inserted_at)
            ORDER BY durable_task_id, durable_task_inserted_at
            LIMIT batch_size
        ), updated AS (
            UPDATE v1_durable_event_log_entry e
            SET result_payload_external_id = gen_random_uuid()
            FROM batch_keys b
            WHERE (e.durable_task_id, e.durable_task_inserted_at) = (b.durable_task_id, b.durable_task_inserted_at)
            RETURNING e.durable_task_id, e.durable_task_inserted_at
        )

        SELECT
            count(*) OVER (),
            durable_task_id,
            durable_task_inserted_at
        INTO rows_updated, last_task_id, last_inserted_at
        FROM updated
        ORDER BY durable_task_id DESC, durable_task_inserted_at DESC
        LIMIT 1;

        EXIT WHEN rows_updated IS NULL OR rows_updated = 0;
    END LOOP;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry ALTER COLUMN result_payload_external_id SET DEFAULT gen_random_uuid();
ALTER TABLE v1_durable_event_log_entry ALTER COLUMN result_payload_external_id SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_durable_event_log_entry DROP COLUMN result_payload_external_id;
-- +goose StatementEnd
