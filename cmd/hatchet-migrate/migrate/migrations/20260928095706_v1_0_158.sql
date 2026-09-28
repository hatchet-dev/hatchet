-- +goose Up
-- The queue-item row mapping of a task moves from the v1_task insert trigger into a schema
-- function that the trigger and CreateTasks both call. Deploy order: apply this migration
-- before running an engine that calls the function. The old engine keeps working on the
-- new schema (the trigger still writes queue items for a statement that does not).

-- +goose StatementBegin
-- store_queue_items_for_tasks writes the queue item of every task in the array that starts
-- QUEUED without a concurrency strategy (a task with one gets its item when a slot is
-- granted) and returns the rows it wrote. It is the one definition of the v1_task to
-- v1_queue_item mapping: v1_task_insert_function calls it for every new row, and
-- CreateTasks calls it for the rows it inserts so that their ids come back with the
-- tasks; ON CONFLICT DO NOTHING on the task's unique index makes the trigger's call for
-- those rows a no-op. Set-based on purpose (one call per statement, not one per row) and
-- plpgsql rather than sql so that its plan is cached per connection: Postgres 15 plans a
-- sql-language body again on every call, which costs more than the rows it writes at
-- small batch sizes. STRICT: a NULL array (no tasks) writes nothing without calling the
-- body.
CREATE OR REPLACE FUNCTION store_queue_items_for_tasks(tasks v1_task[])
RETURNS SETOF v1_queue_item
LANGUAGE plpgsql
STRICT
AS $$
BEGIN
    RETURN QUERY
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
        t.tenant_id,
        t.queue,
        t.id,
        t.inserted_at,
        t.external_id,
        t.action_id,
        t.step_id,
        t.workflow_id,
        t.workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(t.schedule_timeout),
        t.step_timeout,
        COALESCE(t.priority, 1),
        t.sticky,
        t.desired_worker_id,
        t.retry_count,
        t.desired_worker_label,
        t.batch_key
    FROM unnest(tasks) t
    WHERE t.initial_state = 'QUEUED' AND t.concurrency_strategy_ids[1] IS NULL
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING
    RETURNING *;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
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
                concurrency_max_runs[1] AS max_runs,
                CASE
                    WHEN array_length(concurrency_max_runs, 1) > 1 THEN concurrency_max_runs[2:array_length(concurrency_max_runs, 1)]
                    ELSE '{}'::integer[]
                END AS next_max_runs,
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
            COALESCE(priority, 1),
            key,
            next_keys,
            max_runs,
            next_max_runs,
            queue,
            schedule_timeout_at
        FROM new_slot_rows;
    END IF;

    -- Queue items: the row mapping lives in store_queue_items_for_tasks. CreateTasks calls
    -- it for the rows it inserts so that it can return them (a statement cannot see what
    -- its AFTER trigger writes), so for those rows this call is a no-op on the function's
    -- ON CONFLICT DO NOTHING; for every other INSERT into v1_task it is the write.
    PERFORM store_queue_items_for_tasks((SELECT array_agg(t::v1_task) FROM new_table t));

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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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
                concurrency_max_runs[1] AS max_runs,
                CASE
                    WHEN array_length(concurrency_max_runs, 1) > 1 THEN concurrency_max_runs[2:array_length(concurrency_max_runs, 1)]
                    ELSE '{}'::integer[]
                END AS next_max_runs,
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
            COALESCE(priority, 1),
            key,
            next_keys,
            max_runs,
            next_max_runs,
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
-- +goose StatementEnd

-- +goose StatementBegin
DROP FUNCTION store_queue_items_for_tasks(v1_task[]);
-- +goose StatementEnd
