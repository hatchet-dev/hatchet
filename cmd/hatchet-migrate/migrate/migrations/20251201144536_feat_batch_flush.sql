-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Step"
    ADD COLUMN batch_size INTEGER,
    ADD COLUMN batch_flush_interval_ms INTEGER,
    ADD COLUMN batch_key_expression TEXT,
    ADD COLUMN batch_max_runs INTEGER;

ALTER TABLE v1_task
    ADD COLUMN IF NOT EXISTS batch_key TEXT;

ALTER TABLE v1_queue_item
    ADD COLUMN IF NOT EXISTS batch_key TEXT;

ALTER TABLE v1_rate_limited_queue_items
    ADD COLUMN IF NOT EXISTS batch_key TEXT;

ALTER TABLE v1_task_runtime
    ADD COLUMN batch_id UUID,
    ADD COLUMN batch_size INTEGER,
    ADD COLUMN batch_index INTEGER,
    ADD COLUMN batch_key TEXT;

CREATE TABLE IF NOT EXISTS v1_task_batch_run (
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    action_id TEXT NOT NULL,
    batch_key TEXT NOT NULL,
    batch_id UUID NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ,
    CONSTRAINT v1_task_batch_run_pkey PRIMARY KEY (tenant_id, batch_id)
);

CREATE INDEX IF NOT EXISTS v1_task_batch_run_active_key_idx
    ON v1_task_batch_run (tenant_id, step_id, batch_key)
    WHERE completed_at IS NULL;

CREATE INDEX IF NOT EXISTS v1_task_runtime_batch_id_idx
    ON v1_task_runtime USING BTREE (batch_id)
    WHERE batch_id IS NOT NULL;

ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'BATCH_BUFFERED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'WAITING_FOR_BATCH';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'BATCH_FLUSHED';
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    WITH new_slot_rows AS (
        SELECT
            id,
            inserted_at,
            retry_count,
            tenant_id,
            priority,
            concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(concurrency_parent_strategy_ids, 1) > 1 THEN concurrency_parent_strategy_ids[2:array_length(concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            concurrency_strategy_ids[1] AS strategy_id,
            external_id,
            workflow_run_id,
            CASE
                WHEN array_length(concurrency_strategy_ids, 1) > 1 THEN concurrency_strategy_ids[2:array_length(concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            concurrency_keys[1] AS key,
            CASE
                WHEN array_length(concurrency_keys, 1) > 1 THEN concurrency_keys[2:array_length(concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            workflow_id,
            workflow_version_id,
            queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
        FROM new_table
        WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        COALESCE(priority, 1),
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
    )
    SELECT
        nt.tenant_id,
        nt.queue,
        nt.id,
        nt.inserted_at,
        nt.external_id,
        nt.action_id,
        nt.step_id,
        nt.workflow_id,
        nt.workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout),
        nt.step_timeout,
        COALESCE(nt.priority, 1),
        nt.sticky,
        nt.desired_worker_id,
        nt.retry_count,
        COALESCE(nt.batch_key, t.batch_key)
    FROM new_table nt
    LEFT JOIN v1_task t
        ON t.id = nt.id
        AND t.inserted_at = nt.inserted_at
    WHERE nt.initial_state = 'QUEUED' AND nt.concurrency_strategy_ids[1] IS NULL;

    INSERT INTO v1_dag_to_task (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        id,
        inserted_at
    FROM new_table
    WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL;

    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        task_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_concurrency_slot_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    -- If the concurrency slot has next_keys, insert a new slot for the next key
    WITH new_slot_rows AS (
        SELECT
            t.id,
            t.inserted_at,
            t.retry_count,
            t.tenant_id,
            t.priority,
            t.queue,
            t.workflow_run_id,
            t.external_id,
            nt.next_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(nt.next_parent_strategy_ids, 1) > 1 THEN nt.next_parent_strategy_ids[2:array_length(nt.next_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            nt.next_strategy_ids[1] AS strategy_id,
            CASE
                WHEN array_length(nt.next_strategy_ids, 1) > 1 THEN nt.next_strategy_ids[2:array_length(nt.next_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            nt.next_keys[1] AS key,
            CASE
                WHEN array_length(nt.next_keys, 1) > 1 THEN nt.next_keys[2:array_length(nt.next_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            t.workflow_id,
            t.workflow_version_id,
            CURRENT_TIMESTAMP + convert_duration_to_interval(t.schedule_timeout) AS schedule_timeout_at
        FROM new_table nt
        JOIN old_table ot USING (task_id, task_inserted_at, task_retry_count, key)
        JOIN v1_task t ON t.id = nt.task_id AND t.inserted_at = nt.task_inserted_at
        WHERE
            COALESCE(array_length(nt.next_keys, 1), 0) != 0
            AND nt.is_filled = TRUE
            AND nt.is_filled IS DISTINCT FROM ot.is_filled
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        schedule_timeout_at,
        queue_to_notify
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        COALESCE(priority, 1),
        key,
        next_keys,
        schedule_timeout_at,
        queue
    FROM new_slot_rows;

    -- If the concurrency slot does not have next_keys, insert an item into v1_queue_item
    WITH tasks AS (
        SELECT
            t.*
        FROM
            new_table nt
        JOIN old_table ot USING (task_id, task_inserted_at, task_retry_count, key)
        JOIN v1_task t ON t.id = nt.task_id AND t.inserted_at = nt.task_inserted_at
        WHERE
            COALESCE(array_length(nt.next_keys, 1), 0) = 0
            AND nt.is_filled = TRUE
            AND nt.is_filled IS DISTINCT FROM ot.is_filled
    )
    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        tenant_id,
        queue,
        id,
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        COALESCE(priority, 1),
        sticky,
        desired_worker_id,
        retry_count
    FROM tasks;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_retry_queue_item_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_slot_rows AS (
        SELECT
            t.id,
            t.inserted_at,
            t.retry_count,
            t.tenant_id,
            t.workflow_run_id,
            t.external_id,
            t.concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(t.concurrency_parent_strategy_ids, 1) > 1 THEN t.concurrency_parent_strategy_ids[2:array_length(t.concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            t.concurrency_strategy_ids[1] AS strategy_id,
            CASE
                WHEN array_length(t.concurrency_strategy_ids, 1) > 1 THEN t.concurrency_strategy_ids[2:array_length(t.concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            t.concurrency_keys[1] AS key,
            CASE
                WHEN array_length(t.concurrency_keys, 1) > 1 THEN t.concurrency_keys[2:array_length(t.concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            t.workflow_id,
            t.workflow_version_id,
            t.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(t.schedule_timeout) AS schedule_timeout_at
        FROM deleted_rows dr
        JOIN
            v1_task t ON t.id = dr.task_id AND t.inserted_at = dr.task_inserted_at
        WHERE
            dr.retry_after <= NOW()
            AND t.initial_state = 'QUEUED'
            -- Check to see if the task has a concurrency strategy
            AND t.concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        4,
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    WITH tasks AS (
        SELECT
            t.*
        FROM
            deleted_rows dr
        JOIN v1_task t ON t.id = dr.task_id AND t.inserted_at = dr.task_inserted_at
        WHERE
            dr.retry_after <= NOW()
            AND t.initial_state = 'QUEUED'
            AND t.concurrency_strategy_ids[1] IS NULL
    )
    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        tenant_id,
        queue,
        id,
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        4,
        sticky,
        desired_worker_id,
        retry_count
    FROM tasks;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_task_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_retry_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            -- Convert the retry_after based on min(retry_backoff_factor ^ retry_count, retry_max_backoff)
            NOW() + (LEAST(nt.retry_max_backoff, POWER(nt.retry_backoff_factor, nt.app_retry_count)) * interval '1 second') AS retry_after
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            AND nt.retry_backoff_factor IS NOT NULL
            AND ot.app_retry_count IS DISTINCT FROM nt.app_retry_count
            AND nt.app_retry_count != 0
    )
    INSERT INTO v1_retry_queue_item (
        task_id,
        task_inserted_at,
        task_retry_count,
        retry_after,
        tenant_id
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        retry_after,
        tenant_id
    FROM new_retry_rows;

    WITH new_slot_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            nt.workflow_run_id,
            nt.external_id,
            nt.concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(nt.concurrency_parent_strategy_ids, 1) > 1 THEN nt.concurrency_parent_strategy_ids[2:array_length(nt.concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            nt.concurrency_strategy_ids[1] AS strategy_id,
            CASE
                WHEN array_length(nt.concurrency_strategy_ids, 1) > 1 THEN nt.concurrency_strategy_ids[2:array_length(nt.concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            nt.concurrency_keys[1] AS key,
            CASE
                WHEN array_length(nt.concurrency_keys, 1) > 1 THEN nt.concurrency_keys[2:array_length(nt.concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            nt.workflow_id,
            nt.workflow_version_id,
            nt.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout) AS schedule_timeout_at
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            -- Concurrency strategy id should never be null
            AND nt.concurrency_strategy_ids[1] IS NOT NULL
            AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
            AND ot.retry_count IS DISTINCT FROM nt.retry_count
    ), updated_slot AS (
        UPDATE
            v1_concurrency_slot cs
        SET
            task_retry_count = nt.retry_count,
            schedule_timeout_at = nt.schedule_timeout_at,
            is_filled = FALSE,
            priority = 4
        FROM
            new_slot_rows nt
        WHERE
            cs.task_id = nt.id
            AND cs.task_inserted_at = nt.inserted_at
            AND cs.strategy_id = nt.strategy_id
        RETURNING cs.*
    ), slots_to_insert AS (
        -- select the rows that were not updated
        SELECT
            nt.*
        FROM
            new_slot_rows nt
        LEFT JOIN
            updated_slot cs ON cs.task_id = nt.id AND cs.task_inserted_at = nt.inserted_at AND cs.strategy_id = nt.strategy_id
        WHERE
            cs.task_id IS NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        4,
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM slots_to_insert;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count,
        batch_key
    )
    SELECT
        nt.tenant_id,
        nt.queue,
        nt.id,
        nt.inserted_at,
        nt.external_id,
        nt.action_id,
        nt.step_id,
        nt.workflow_id,
        nt.workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout),
        nt.step_timeout,
        4,
        nt.sticky,
        nt.desired_worker_id,
        nt.retry_count,
        nt.batch_key
    FROM new_table nt
    JOIN old_table ot ON ot.id = nt.id
    WHERE nt.initial_state = 'QUEUED'
        AND nt.concurrency_strategy_ids[1] IS NULL
        AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
        AND ot.retry_count IS DISTINCT FROM nt.retry_count;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Step"
    DROP COLUMN IF EXISTS batch_max_runs,
    DROP COLUMN IF EXISTS batch_key_expression,
    DROP COLUMN IF EXISTS batch_flush_interval_ms,
    DROP COLUMN IF EXISTS batch_size;

DROP INDEX IF EXISTS v1_task_runtime_batch_id_idx;

ALTER TABLE v1_task_runtime
    DROP COLUMN IF EXISTS batch_key,
    DROP COLUMN IF EXISTS batch_index,
    DROP COLUMN IF EXISTS batch_size,
    DROP COLUMN IF EXISTS batch_id;

ALTER TABLE v1_task
    DROP COLUMN IF EXISTS batch_key;

ALTER TABLE v1_queue_item
    DROP COLUMN IF EXISTS batch_key;

ALTER TABLE v1_rate_limited_queue_items
    DROP COLUMN IF EXISTS batch_key;

DROP INDEX IF EXISTS v1_task_batch_run_active_key_idx;

DROP TABLE IF EXISTS v1_task_batch_run;

SELECT 'no-op: cannot remove enum values from v1_event_type_olap';
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    WITH new_slot_rows AS (
        SELECT
            id,
            inserted_at,
            retry_count,
            tenant_id,
            priority,
            concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(concurrency_parent_strategy_ids, 1) > 1 THEN concurrency_parent_strategy_ids[2:array_length(concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            concurrency_strategy_ids[1] AS strategy_id,
            external_id,
            workflow_run_id,
            CASE
                WHEN array_length(concurrency_strategy_ids, 1) > 1 THEN concurrency_strategy_ids[2:array_length(concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            concurrency_keys[1] AS key,
            CASE
                WHEN array_length(concurrency_keys, 1) > 1 THEN concurrency_keys[2:array_length(concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            workflow_id,
            workflow_version_id,
            queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
        FROM new_table
        WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        COALESCE(priority, 1),
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        tenant_id,
        queue,
        id,
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        COALESCE(priority, 1),
        sticky,
        desired_worker_id,
        retry_count
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL;

    INSERT INTO v1_dag_to_task (
        dag_id,
        dag_inserted_at,
        task_id,
        task_inserted_at
    )
    SELECT
        dag_id,
        dag_inserted_at,
        id,
        inserted_at
    FROM new_table
    WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL;

    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        task_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_task_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_retry_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            NOW() + (LEAST(nt.retry_max_backoff, POWER(nt.retry_backoff_factor, nt.app_retry_count)) * interval '1 second') AS retry_after
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            AND nt.retry_backoff_factor IS NOT NULL
            AND ot.app_retry_count IS DISTINCT FROM nt.app_retry_count
            AND nt.app_retry_count != 0
    )
    INSERT INTO v1_retry_queue_item (
        task_id,
        task_inserted_at,
        task_retry_count,
        retry_after,
        tenant_id
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        retry_after,
        tenant_id
    FROM new_retry_rows;

    WITH new_slot_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            nt.workflow_run_id,
            nt.external_id,
            nt.concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(nt.concurrency_parent_strategy_ids, 1) > 1 THEN nt.concurrency_parent_strategy_ids[2:array_length(nt.concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            nt.concurrency_strategy_ids[1] AS strategy_id,
            CASE
                WHEN array_length(nt.concurrency_strategy_ids, 1) > 1 THEN nt.concurrency_strategy_ids[2:array_length(nt.concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            nt.concurrency_keys[1] AS key,
            CASE
                WHEN array_length(nt.concurrency_keys, 1) > 1 THEN nt.concurrency_keys[2:array_length(nt.concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            nt.workflow_id,
            nt.workflow_version_id,
            nt.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout) AS schedule_timeout_at
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            AND nt.concurrency_strategy_ids[1] IS NOT NULL
            AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
            AND ot.retry_count IS DISTINCT FROM nt.retry_count
    ), updated_slot AS (
        UPDATE
            v1_concurrency_slot cs
        SET
            task_retry_count = nt.retry_count,
            schedule_timeout_at = nt.schedule_timeout_at,
            is_filled = FALSE,
            priority = 4
        FROM
            new_slot_rows nt
        WHERE
            cs.task_id = nt.id
            AND cs.task_inserted_at = nt.inserted_at
            AND cs.strategy_id = nt.strategy_id
        RETURNING cs.*
    ), slots_to_insert AS (
        -- select the rows that were not updated
        SELECT
            nt.*
        FROM
            new_slot_rows nt
        LEFT JOIN
            updated_slot cs ON cs.task_id = nt.id AND cs.task_inserted_at = nt.inserted_at AND cs.strategy_id = nt.strategy_id
        WHERE
            cs.task_id IS NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        priority,
        key,
        next_keys,
        queue_to_notify,
        schedule_timeout_at
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        4,
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM slots_to_insert;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        retry_count
    )
    SELECT
        nt.tenant_id,
        nt.queue,
        nt.id,
        nt.inserted_at,
        nt.external_id,
        nt.action_id,
        nt.step_id,
        nt.workflow_id,
        nt.workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout),
        nt.step_timeout,
        4,
        nt.sticky,
        nt.desired_worker_id,
        nt.retry_count
    FROM new_table nt
    JOIN old_table ot ON ot.id = nt.id
    WHERE nt.initial_state = 'QUEUED'
        AND nt.concurrency_strategy_ids[1] IS NULL
        AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
        AND ot.retry_count IS DISTINCT FROM nt.retry_count;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
