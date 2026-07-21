-- +goose Up
-- +goose StatementBegin
-- v0 schema alignment
ALTER TYPE "LeaseKind" ADD VALUE 'BATCH';

-- v1 batching propagation fields
ALTER TABLE v1_task
    ADD COLUMN batch_key TEXT;

ALTER TABLE v1_queue_item
    ADD COLUMN batch_key TEXT;

ALTER TABLE v1_rate_limited_queue_items
    ADD COLUMN batch_key TEXT;

ALTER TABLE v1_task_runtime
    ADD COLUMN batch_id UUID,
    ADD COLUMN batch_size INTEGER,
    ADD COLUMN batch_index INTEGER,
    ADD COLUMN batch_key TEXT;

-- Per-step batching configuration
CREATE TABLE v1_step_batch_config (
    step_id UUID NOT NULL,
    batch_max_size INTEGER NOT NULL,
    batch_max_interval INTEGER,
    batch_group_key TEXT,
    batch_group_max_runs INTEGER,
    broadcast_output BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT v1_step_batch_config_pkey PRIMARY KEY (step_id)
);

-- Batched queue items buffer table
CREATE TABLE v1_batched_queue_item (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    external_id UUID NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
    schedule_timeout_at TIMESTAMP(3),
    step_timeout TEXT,
    priority INTEGER NOT NULL DEFAULT 1,
    sticky v1_sticky_strategy NOT NULL,
    desired_worker_id UUID,
    retry_count INTEGER NOT NULL DEFAULT 0,
    batch_key TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    payload_size INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT v1_batched_queue_item_pkey PRIMARY KEY (id),
    CONSTRAINT v1_batched_queue_item_task_key UNIQUE (task_id, task_inserted_at, retry_count)
);

ALTER TABLE v1_batched_queue_item SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

CREATE INDEX v1_batched_queue_item_step_batch_id_idx
    ON v1_batched_queue_item (tenant_id ASC, step_id ASC, batch_key ASC, id ASC);

CREATE TABLE v1_batch_runtime (
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    action_id TEXT NOT NULL,
    batch_key TEXT NOT NULL,
    batch_id UUID NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_batch_runtime_pkey PRIMARY KEY (tenant_id, batch_id)
);

CREATE INDEX v1_batch_runtime_key_idx
    ON v1_batch_runtime (tenant_id, step_id, batch_key);

-- OLAP enum additions for batching lifecycle events
ALTER TYPE v1_event_type_olap ADD VALUE 'BATCH_BUFFERED';
ALTER TYPE v1_event_type_olap ADD VALUE 'WAITING_FOR_BATCH';
ALTER TYPE v1_event_type_olap ADD VALUE 'BATCH_FLUSHED';

-- Update trigger functions to match current canonical definitions in sql/schema/v1-core.sql (batch_key propagation)
CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Only insert if there's a single task with initial_state = 'QUEUED' and concurrency_strategy_ids is not null
    IF (SELECT COUNT(*) FROM new_table WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL) > 0 THEN
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
    END IF;

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
        desired_worker_label,
        batch_key
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
        retry_count,
        desired_worker_label,
        batch_key
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    -- Only insert into v1_dag and v1_dag_to_task if dag_id and dag_inserted_at are not null
    IF (SELECT COUNT(*) FROM new_table WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL) > 0 THEN
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
    END IF;

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
        desired_worker_label,
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
        nt.desired_worker_label,
        nt.batch_key
    FROM new_table nt
    JOIN old_table ot ON ot.id = nt.id
    WHERE nt.initial_state = 'QUEUED'
        AND nt.concurrency_strategy_ids[1] IS NULL
        AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
        AND ot.retry_count IS DISTINCT FROM nt.retry_count
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

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
        retry_count,
        desired_worker_label,
        batch_key
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
        retry_count,
        desired_worker_label,
        batch_key
    FROM tasks
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

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
        retry_count,
        desired_worker_label,
        batch_key
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
        retry_count,
        desired_worker_label,
        batch_key
    FROM tasks
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION after_v1_task_runtime_delete_cleanup_batch_runtime_fn()
RETURNS trigger AS $$
BEGIN
    -- Consider only batch-associated runtime rows which were deleted in this statement.
    -- Delete the corresponding v1_batch_runtime reservation iff no runtimes remain for that batch.
    WITH deleted_batches AS (
        SELECT DISTINCT
            d.tenant_id,
            d.batch_id
        FROM
            deleted_rows d
        WHERE
            d.batch_id IS NOT NULL
    ), deletable AS (
        SELECT
            br.tenant_id,
            br.batch_id
        FROM
            v1_batch_runtime br
        JOIN
            deleted_batches db ON db.tenant_id = br.tenant_id AND db.batch_id = br.batch_id
        WHERE NOT EXISTS (
            SELECT 1
            FROM v1_task_runtime tr
            WHERE tr.tenant_id = br.tenant_id
              AND tr.batch_id = br.batch_id
        )
        ORDER BY br.batch_id
        FOR UPDATE
    )
    DELETE FROM
        v1_batch_runtime br
    WHERE
        (br.tenant_id, br.batch_id) IN (SELECT tenant_id, batch_id FROM deletable);

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_task_runtime_delete_cleanup_batch_runtime
AFTER DELETE ON v1_task_runtime
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_task_runtime_delete_cleanup_batch_runtime_fn();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER after_v1_task_runtime_delete_cleanup_batch_runtime ON v1_task_runtime;
DROP FUNCTION after_v1_task_runtime_delete_cleanup_batch_runtime_fn();

-- Drop batch buffer table and indexes
DROP TABLE v1_batched_queue_item;

-- Drop per-step batching configuration
DROP TABLE v1_step_batch_config;

-- Drop v1 batch runtime table
DROP TABLE v1_batch_runtime;

-- Drop runtime batch metadata
ALTER TABLE v1_task_runtime
    DROP COLUMN batch_key,
    DROP COLUMN batch_index,
    DROP COLUMN batch_size,
    DROP COLUMN batch_id;

ALTER TABLE v1_task
    DROP COLUMN batch_key;

ALTER TABLE v1_queue_item
    DROP COLUMN batch_key;

ALTER TABLE v1_rate_limited_queue_items
    DROP COLUMN batch_key;

-- Restore trigger functions to pre-batching versions (from v1_0_106 baseline), so down migrations remain functional.
CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Only insert if there's a single task with initial_state = 'QUEUED' and concurrency_strategy_ids is not null
    IF (SELECT COUNT(*) FROM new_table WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL) > 0 THEN
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
    END IF;

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
        desired_worker_label
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
        retry_count,
        desired_worker_label
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    -- Only insert into v1_dag and v1_dag_to_task if dag_id and dag_inserted_at are not null
    IF (SELECT COUNT(*) FROM new_table WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL) > 0 THEN
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
    END IF;

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
        desired_worker_label,
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
        nt.desired_worker_label,
        nt.batch_key
    FROM new_table nt
    JOIN old_table ot ON ot.id = nt.id
    WHERE nt.initial_state = 'QUEUED'
        AND nt.concurrency_strategy_ids[1] IS NULL
        AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
        AND ot.retry_count IS DISTINCT FROM nt.retry_count
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

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
        retry_count,
        desired_worker_label
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
        retry_count,
        desired_worker_label
    FROM tasks
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

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
        retry_count,
        desired_worker_label
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
        retry_count,
        desired_worker_label
    FROM tasks
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    ;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd
