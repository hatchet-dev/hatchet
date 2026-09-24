-- +goose Up
-- +goose StatementBegin
-- The existing table is named after the current month so retention drops it once the whole month
-- is past the retention cutoff. Its old single-column primary key stays as the per-partition unique
-- constraint on external_id, using the same name the engine gives new partitions.
DO $$
DECLARE
    current_month_start DATE := date_trunc('month', NOW())::DATE;
    next_month_start DATE := (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE;
    legacy_partition_name TEXT := 'v1_lookup_table_' || to_char(current_month_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_lookup_table RENAME CONSTRAINT v1_lookup_table_external_id_inserted_at_uq TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_lookup_table RENAME CONSTRAINT v1_lookup_table_pkey TO %I', legacy_partition_name || '_external_id_uq');
    EXECUTE format('ALTER TABLE v1_lookup_table RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_lookup_table_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, next_month_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_lookup_table_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_lookup_table_partitioned RENAME TO v1_lookup_table;
ALTER INDEX v1_lookup_table_partitioned_pkey RENAME TO v1_lookup_table_pkey;

-- Update v1_task_insert_function to use new composite PK (external_id, inserted_at)
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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

-- Update v1_dag_insert_function to use new composite PK (external_id, inserted_at)
CREATE OR REPLACE FUNCTION v1_dag_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        dag_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_lookup_table'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_lookup_table_original (
            tenant_id UUID NOT NULL,
            external_id UUID NOT NULL,
            task_id BIGINT,
            dag_id BIGINT,
            inserted_at TIMESTAMPTZ NOT NULL,
            PRIMARY KEY (external_id)
        );

        INSERT INTO v1_lookup_table_original
        SELECT tenant_id, external_id, task_id, dag_id, inserted_at FROM v1_lookup_table
        ON CONFLICT (external_id) DO NOTHING;

        DROP TABLE v1_lookup_table;
        ALTER TABLE v1_lookup_table_original RENAME TO v1_lookup_table;
        ALTER INDEX v1_lookup_table_original_pkey RENAME TO v1_lookup_table_pkey;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_lookup_table DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I SELECT tenant_id, external_id, task_id, dag_id, inserted_at FROM v1_lookup_table ON CONFLICT (external_id) DO NOTHING', legacy_partition_name);
    DROP TABLE v1_lookup_table;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_lookup_table', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_lookup_table RENAME CONSTRAINT %I TO v1_lookup_table_pkey', legacy_partition_name || '_external_id_uq');
    EXECUTE format('ALTER TABLE v1_lookup_table RENAME CONSTRAINT %I TO v1_lookup_table_external_id_inserted_at_uq', legacy_partition_name || '_pkey');
END $$;

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

CREATE OR REPLACE FUNCTION v1_dag_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        dag_id,
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
