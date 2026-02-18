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

-- Clean up any spurious 'default' slots that were created for durable tasks
-- by the old trigger. These are rows where a task has both a 'default' and another
-- slot type (like 'durable'), and the step only requested the non-default type.
DELETE FROM v1_task_runtime_slot d
USING v1_task_runtime_slot other
JOIN v1_task_runtime tr
  ON tr.task_id = other.task_id
  AND tr.task_inserted_at = other.task_inserted_at
  AND tr.retry_count = other.retry_count
JOIN v1_task t
  ON t.id = tr.task_id
  AND t.inserted_at = tr.task_inserted_at
JOIN v1_step_slot_request req
  ON req.step_id = t.step_id
  AND req.tenant_id = t.tenant_id
WHERE d.task_id = other.task_id
  AND d.task_inserted_at = other.task_inserted_at
  AND d.retry_count = other.retry_count
  AND d.slot_type = 'default'
  AND other.slot_type <> 'default'
  AND NOT EXISTS (
      SELECT 1 FROM v1_step_slot_request r
      WHERE r.step_id = t.step_id
        AND r.tenant_id = t.tenant_id
        AND r.slot_type = 'default'
  );

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
