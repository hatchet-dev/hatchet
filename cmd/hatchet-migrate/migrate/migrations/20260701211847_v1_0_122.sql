-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_update_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        'concurrency.' || nt.tenant_id::text || '.' || nt.strategy_id::text,
        jsonb_build_object(
            'operation', 'UPDATE',
            'key', nt.key,
            'priority', nt.priority,
            'taskId', nt.task_id,
            'taskInsertedAt', nt.task_inserted_at,
            'taskRetryCount', nt.task_retry_count,
            'scheduleTimeoutAtMs', (EXTRACT(EPOCH FROM nt.schedule_timeout_at) * 1000)::bigint,
            'isFilled', nt.is_filled
        )
    FROM new_table nt
    JOIN v1_step_concurrency sc ON sc.id = nt.strategy_id
    WHERE sc.parent_strategy_id IS NULL
        AND nt.is_filled = FALSE;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_update_outbox
AFTER UPDATE ON v1_concurrency_slot
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_update_outbox_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS after_v1_concurrency_slot_update_outbox ON v1_concurrency_slot;
DROP FUNCTION IF EXISTS after_v1_concurrency_slot_update_outbox_function();
-- +goose StatementEnd
