-- +goose Up
-- The queue-item and payload row mappings of a task move from the v1_task insert trigger
-- (queue items) and the CreateTasks statement into schema functions that both call.
-- Deploy order: apply this migration before running an engine that calls the functions.
-- The old engine keeps working on the new schema (the trigger still writes queue items
-- for a statement that does not).

-- +goose StatementBegin
-- store_queue_items_for_tasks writes the queue item of every task in the array that starts
-- QUEUED without a concurrency strategy (a task with one gets its item when a slot is
-- granted) and returns the rows it wrote. It is the one definition of the v1_task to
-- v1_queue_item mapping: v1_task_insert_function calls it for the new rows that have no
-- item yet, and CreateTasks calls it for the rows it inserts so that their ids come back
-- with the tasks. Set-based on purpose (one call per statement, not one per row) and
-- plpgsql rather than sql so that its plan is cached per connection: Postgres 15 plans a
-- sql-language body again on every call, which costs more than the rows it writes at
-- small batch sizes. ON CONFLICT DO NOTHING keeps a second call for the same tasks
-- harmless. STRICT: a NULL array (no tasks) writes nothing without calling the body.
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
-- store_payloads_for_tasks writes the TASK_INPUT payload row of every task in the array
-- whose input (the element at the same position of inputs) is not NULL and returns the
-- rows it wrote. v1_task.input is always '{}': the v1_payload row is the only copy of the
-- input, and an empty input writes no row. CreateTasks calls it for the rows it inserts;
-- v1_task_insert_function cannot, because the input bytes are not on the task row.
-- Set-based, plpgsql and STRICT for the reasons given on store_queue_items_for_tasks.
CREATE OR REPLACE FUNCTION store_payloads_for_tasks(tasks v1_task[], inputs jsonb[])
RETURNS SETOF v1_payload
LANGUAGE plpgsql
STRICT
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO v1_payload (
        tenant_id,
        id,
        inserted_at,
        external_id,
        type,
        location,
        external_location_key,
        inline_content
    )
    SELECT
        (tasks[n]).tenant_id,
        (tasks[n]).id,
        (tasks[n]).inserted_at,
        (tasks[n]).external_id,
        'TASK_INPUT'::v1_payload_type,
        'INLINE'::v1_payload_location,
        NULL,
        inputs[n]
    FROM generate_subscripts(tasks, 1) AS n
    WHERE inputs[n] IS NOT NULL
    ON CONFLICT DO NOTHING
    RETURNING *;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
-- store_payloads_for_events is store_payloads_for_tasks for events: the USER_EVENT_INPUT
-- payload row of every event in the array whose input is not NULL. BulkCreateEvents calls
-- it for the rows it inserts.
CREATE OR REPLACE FUNCTION store_payloads_for_events(events v1_event[], inputs jsonb[])
RETURNS SETOF v1_payload
LANGUAGE plpgsql
STRICT
AS $$
BEGIN
    RETURN QUERY
    INSERT INTO v1_payload (
        tenant_id,
        id,
        inserted_at,
        external_id,
        type,
        location,
        external_location_key,
        inline_content
    )
    SELECT
        (events[n]).tenant_id,
        (events[n]).id,
        (events[n]).seen_at,
        (events[n]).external_id,
        'USER_EVENT_INPUT'::v1_payload_type,
        'INLINE'::v1_payload_location,
        NULL,
        inputs[n]
    FROM generate_subscripts(events, 1) AS n
    WHERE inputs[n] IS NOT NULL
    ON CONFLICT DO NOTHING
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

    -- Queue items: the row mapping lives in store_queue_items_for_tasks, which CreateTasks
    -- calls itself for the rows it inserts so that it can return them (a statement cannot
    -- see what its AFTER trigger writes). Only the new rows that have no queue item yet are
    -- passed on, so the common path evaluates the mapping once per row, and this call is
    -- the path for every other INSERT into v1_task.
    PERFORM store_queue_items_for_tasks((
        SELECT array_agg(t::v1_task)
        FROM new_table t
        WHERE NOT EXISTS (
            SELECT 1
            FROM v1_queue_item qi
            WHERE qi.task_id = t.id AND qi.task_inserted_at = t.inserted_at AND qi.retry_count = t.retry_count
        )
    ));

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
DROP FUNCTION store_payloads_for_tasks(v1_task[], jsonb[]);
DROP FUNCTION store_payloads_for_events(v1_event[], jsonb[]);
-- +goose StatementEnd
