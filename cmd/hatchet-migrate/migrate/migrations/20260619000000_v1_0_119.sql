-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        nt.tenant_id::text || '.' || nt.strategy_id::text,
        jsonb_build_object(
            'operation', 'INSERT',
            'key', nt.key,
            'priority', nt.priority,
            'taskId', nt.task_id,
            'taskInsertedAt', nt.task_inserted_at,
            'taskRetryCount', nt.task_retry_count,
            'scheduleTimeoutAtMs', (EXTRACT(EPOCH FROM nt.schedule_timeout_at) * 1000)::bigint
        )
    FROM new_table nt
    JOIN v1_step_concurrency sc ON sc.id = nt.strategy_id
    WHERE sc.parent_strategy_id IS NULL
      AND sc.strategy IN ('GROUP_ROUND_ROBIN', 'CANCEL_IN_PROGRESS', 'CANCEL_NEWEST');

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_insert_outbox
AFTER INSERT ON v1_concurrency_slot
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_insert_outbox_function();

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        dr.tenant_id::text || '.' || dr.strategy_id::text,
        jsonb_build_object(
            'operation', 'DELETE',
            'key', dr.key,
            'priority', dr.priority,
            'taskId', dr.task_id,
            'taskInsertedAt', dr.task_inserted_at,
            'taskRetryCount', dr.task_retry_count,
            'scheduleTimeoutAtMs', (EXTRACT(EPOCH FROM dr.schedule_timeout_at) * 1000)::bigint
        )
    FROM deleted_rows dr
    JOIN v1_step_concurrency sc ON sc.id = dr.strategy_id
    WHERE sc.parent_strategy_id IS NULL
      AND sc.strategy IN ('GROUP_ROUND_ROBIN', 'CANCEL_IN_PROGRESS', 'CANCEL_NEWEST');

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_delete_outbox
AFTER DELETE ON v1_concurrency_slot
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_delete_outbox_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS after_v1_concurrency_slot_insert_outbox ON v1_concurrency_slot;
DROP TRIGGER IF EXISTS after_v1_concurrency_slot_delete_outbox ON v1_concurrency_slot;
DROP FUNCTION IF EXISTS after_v1_concurrency_slot_insert_outbox_function();
DROP FUNCTION IF EXISTS after_v1_concurrency_slot_delete_outbox_function();
-- +goose StatementEnd
