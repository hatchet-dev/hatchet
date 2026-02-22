-- +goose Up
-- +goose StatementBegin

-- The original trigger always inserted a 'default' slot into v1_task_runtime_slot
-- when a row was inserted into v1_task_runtime. This caused durable tasks to also
-- acquire a default slot because the new code path (assigned_slots CTE) already
-- inserts the correct slot type, but the trigger would fire afterwards and add
-- an extra 'default' row. The NOT EXISTS guard skips the trigger insert when the
-- new code path has already written slots for the task.
CREATE OR REPLACE FUNCTION v1_task_runtime_slot_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_task_runtime_slot (
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        slot_type,
        units
    )
    SELECT
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        'default'::text,
        1
    FROM new_rows nr
    WHERE nr.worker_id IS NOT NULL
    AND NOT EXISTS (
        SELECT 1 FROM v1_task_runtime_slot s
        WHERE s.task_id = nr.task_id
          AND s.task_inserted_at = nr.task_inserted_at
          AND s.retry_count = nr.retry_count
    )
    ON CONFLICT (task_id, task_inserted_at, retry_count, slot_type) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_task_runtime_slot_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_task_runtime_slot (
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        slot_type,
        units
    )
    SELECT
        tenant_id,
        task_id,
        task_inserted_at,
        retry_count,
        worker_id,
        'default'::text,
        1
    FROM new_rows
    WHERE worker_id IS NOT NULL
    ON CONFLICT (task_id, task_inserted_at, retry_count, slot_type) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
