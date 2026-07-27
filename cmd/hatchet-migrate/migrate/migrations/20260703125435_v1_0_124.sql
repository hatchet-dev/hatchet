-- +goose Up
-- +goose StatementBegin

-- The outbox trigger functions joined v1_step_concurrency on id alone (unindexed, since the
-- primary key leads with workflow_id), sequentially scanning it on every concurrency slot
-- insert/update/delete. The slot row's own parent_strategy_id is populated from the same
-- v1_step_concurrency row, so we can filter on it directly and skip the join.

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        'concurrency.' || nt.tenant_id::text || '.' || nt.strategy_id::text,
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
    WHERE nt.parent_strategy_id IS NULL;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        'concurrency.' || dr.tenant_id::text || '.' || dr.strategy_id::text,
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
    WHERE dr.parent_strategy_id IS NULL;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

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
    WHERE nt.parent_strategy_id IS NULL
        AND nt.is_filled = FALSE;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        'concurrency.' || nt.tenant_id::text || '.' || nt.strategy_id::text,
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
    WHERE sc.parent_strategy_id IS NULL;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_outbox_function()
RETURNS trigger AS $$
BEGIN
    INSERT INTO outbox.messages (topic, payload)
    SELECT
        'concurrency.' || dr.tenant_id::text || '.' || dr.strategy_id::text,
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
    WHERE sc.parent_strategy_id IS NULL;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

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

-- +goose StatementEnd
