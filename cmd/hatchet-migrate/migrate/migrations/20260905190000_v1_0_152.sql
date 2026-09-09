-- +goose Up

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
            -- NOTE: cap in log space before POWER. POWER(backoff_factor, app_retry_count)
            -- overflows float8 (SQLSTATE 22003) for large retry counts, and LEAST cannot
            -- prevent that because POWER is evaluated first. b^n > cap iff
            -- n * ln(b) > ln(cap), so compare logs and skip POWER when the result
            -- would exceed retry_max_backoff.
            NOW() + (
                LEAST(
                    COALESCE(nt.retry_max_backoff, 86400)::double precision,
                    CASE
                        WHEN nt.retry_backoff_factor <= 1 THEN
                            POWER(nt.retry_backoff_factor, nt.app_retry_count)
                        WHEN LN(nt.retry_backoff_factor) * nt.app_retry_count
                            >= LN(GREATEST(COALESCE(nt.retry_max_backoff, 86400), 1)::double precision) THEN
                            COALESCE(nt.retry_max_backoff, 86400)::double precision
                        ELSE
                            POWER(nt.retry_backoff_factor, nt.app_retry_count)
                    END
                ) * interval '1 second'
            ) AS retry_after
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

    -- A pending retry remains unfinished work for replacement policies. Keep its
    -- filled slot at the accepted invocation's position until retry admission.
    UPDATE v1_concurrency_slot cs
    SET task_retry_count = nt.retry_count
    FROM new_table nt
    JOIN old_table ot ON ot.id = nt.id AND ot.inserted_at = nt.inserted_at
    JOIN v1_step_concurrency sc ON sc.id = ANY(nt.concurrency_strategy_ids)
    WHERE cs.task_id = nt.id
        AND cs.task_inserted_at = nt.inserted_at
        AND cs.task_retry_count = ot.retry_count
        AND cs.strategy_id = sc.id
        AND sc.strategy = 'CANCEL_IN_PROGRESS'
        AND cs.is_filled
        AND nt.initial_state = 'QUEUED'
        AND nt.retry_backoff_factor IS NOT NULL
        AND ot.app_retry_count IS DISTINCT FROM nt.app_retry_count
        AND nt.app_retry_count != 0;

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
            nt.concurrency_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(nt.concurrency_max_runs, 1) > 1 THEN nt.concurrency_max_runs[2:array_length(nt.concurrency_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
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
            t.concurrency_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(t.concurrency_max_runs, 1) > 1 THEN t.concurrency_max_runs[2:array_length(t.concurrency_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
        queue,
        schedule_timeout_at
    FROM new_slot_rows
    ON CONFLICT (task_id, task_inserted_at, task_retry_count, strategy_id)
    DO UPDATE SET is_filled = FALSE, schedule_timeout_at = EXCLUDED.schedule_timeout_at;

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
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Removing a retained replacement slot must suppress retry admission in the
    -- same transaction, including retries whose deadline has already elapsed.
    UPDATE v1_retry_queue_item r
    SET retry_after = 'infinity'::timestamptz
    FROM deleted_rows d
    WHERE r.task_id = d.task_id
        AND r.task_inserted_at = d.task_inserted_at
        AND r.task_retry_count = d.task_retry_count;

    DELETE FROM v1_retry_queue_item r
    USING deleted_rows d
    WHERE r.task_id = d.task_id
        AND r.task_inserted_at = d.task_inserted_at
        AND r.task_retry_count = d.task_retry_count;

    FOR rec IN SELECT * FROM deleted_rows ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
        IF rec.parent_strategy_id IS NOT NULL THEN
            PERFORM cleanup_workflow_concurrency_slots(
                rec.parent_strategy_id,
                rec.workflow_version_id,
                rec.workflow_run_id
            );
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_concurrency_slot_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    -- Chained-slot upserts invoke statement-level UPDATE triggers even when
    -- there are no rows to advance.
    IF NOT EXISTS (SELECT 1 FROM new_table) THEN
        RETURN NULL;
    END IF;

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
            nt.next_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(nt.next_max_runs, 1) > 1 THEN nt.next_max_runs[2:array_length(nt.next_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
        schedule_timeout_at,
        queue
    FROM new_slot_rows
    ON CONFLICT (task_id, task_inserted_at, task_retry_count, strategy_id)
    DO UPDATE SET is_filled = FALSE, schedule_timeout_at = EXCLUDED.schedule_timeout_at;

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
-- +goose StatementEnd

-- +goose Down

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
            -- NOTE: cap in log space before POWER. POWER(backoff_factor, app_retry_count)
            -- overflows float8 (SQLSTATE 22003) for large retry counts, and LEAST cannot
            -- prevent that because POWER is evaluated first. b^n > cap iff
            -- n * ln(b) > ln(cap), so compare logs and skip POWER when the result
            -- would exceed retry_max_backoff.
            NOW() + (
                LEAST(
                    COALESCE(nt.retry_max_backoff, 86400)::double precision,
                    CASE
                        WHEN nt.retry_backoff_factor <= 1 THEN
                            POWER(nt.retry_backoff_factor, nt.app_retry_count)
                        WHEN LN(nt.retry_backoff_factor) * nt.app_retry_count
                            >= LN(GREATEST(COALESCE(nt.retry_max_backoff, 86400), 1)::double precision) THEN
                            COALESCE(nt.retry_max_backoff, 86400)::double precision
                        ELSE
                            POWER(nt.retry_backoff_factor, nt.app_retry_count)
                    END
                ) * interval '1 second'
            ) AS retry_after
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
            nt.concurrency_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(nt.concurrency_max_runs, 1) > 1 THEN nt.concurrency_max_runs[2:array_length(nt.concurrency_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
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
            t.concurrency_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(t.concurrency_max_runs, 1) > 1 THEN t.concurrency_max_runs[2:array_length(t.concurrency_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
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
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT * FROM deleted_rows ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
        IF rec.parent_strategy_id IS NOT NULL THEN
            PERFORM cleanup_workflow_concurrency_slots(
                rec.parent_strategy_id,
                rec.workflow_version_id,
                rec.workflow_run_id
            );
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
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
            nt.next_max_runs[1] AS max_runs,
            CASE
                WHEN array_length(nt.next_max_runs, 1) > 1 THEN nt.next_max_runs[2:array_length(nt.next_max_runs, 1)]
                ELSE '{}'::integer[]
            END AS next_max_runs,
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
        max_runs,
        next_max_runs,
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
        max_runs,
        next_max_runs,
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
-- +goose StatementEnd

-- Pending retries must be admitted without conflicting with retained slots.
DELETE FROM v1_concurrency_slot cs
USING v1_retry_queue_item r
WHERE cs.task_id = r.task_id
    AND cs.task_inserted_at = r.task_inserted_at
    AND cs.task_retry_count = r.task_retry_count;
