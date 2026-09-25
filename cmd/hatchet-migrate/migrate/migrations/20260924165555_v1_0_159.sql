-- +goose Up
-- +goose StatementBegin
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

DO $$
DECLARE
    next_month_start DATE := (date_trunc('month', NOW()) + INTERVAL '1 month')::DATE;
    next_month_partition_name TEXT := 'v1_lookup_table_' || to_char(next_month_start, 'YYYYMMDD');
BEGIN
    PERFORM create_v1_monthly_range_partition('v1_lookup_table', next_month_start);
    EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I UNIQUE (external_id)', next_month_partition_name, next_month_partition_name || '_external_id_uq');
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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    today_start DATE := (NOW() AT TIME ZONE 'UTC')::DATE;
    tomorrow_start DATE := today_start + 1;
    legacy_partition_name TEXT := 'v1_dag_to_task_' || to_char(today_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_dag_to_task RENAME CONSTRAINT v1_dag_to_task_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_dag_to_task RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_to_task_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, tomorrow_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_dag_to_task_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_dag_to_task_partitioned RENAME TO v1_dag_to_task;
ALTER INDEX v1_dag_to_task_partitioned_pkey RENAME TO v1_dag_to_task_pkey;

SELECT create_v1_range_partition('v1_dag_to_task', ((NOW() AT TIME ZONE 'UTC')::DATE + 1));
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    today_start DATE := (NOW() AT TIME ZONE 'UTC')::DATE;
    tomorrow_start DATE := today_start + 1;
    legacy_partition_name TEXT := 'v1_dag_data_' || to_char(today_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_dag_data RENAME CONSTRAINT v1_dag_input_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_dag_data RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_data_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, tomorrow_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_dag_data_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_dag_data_partitioned RENAME TO v1_dag_data;
ALTER INDEX v1_dag_data_partitioned_pkey RENAME TO v1_dag_data_pkey;

SELECT create_v1_range_partition('v1_dag_data', ((NOW() AT TIME ZONE 'UTC')::DATE + 1));
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    today_start DATE := (NOW() AT TIME ZONE 'UTC')::DATE;
    tomorrow_start DATE := today_start + 1;
    legacy_partition_name TEXT := 'v1_task_expression_eval_' || to_char(today_start, 'YYYYMMDD');
BEGIN
    EXECUTE format('ALTER TABLE v1_task_expression_eval RENAME CONSTRAINT v1_task_expression_eval_pkey TO %I', legacy_partition_name || '_pkey');
    EXECUTE format('ALTER TABLE v1_task_expression_eval RENAME TO %I', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_task_expression_eval_partitioned ATTACH PARTITION %I FOR VALUES FROM (MINVALUE) TO (%L)', legacy_partition_name, tomorrow_start);
    EXECUTE format('ALTER TABLE %I DROP CONSTRAINT v1_task_expression_eval_attach_bound', legacy_partition_name);
END $$;

ALTER TABLE v1_task_expression_eval_partitioned RENAME TO v1_task_expression_eval;
ALTER INDEX v1_task_expression_eval_partitioned_pkey RENAME TO v1_task_expression_eval_pkey;

SELECT create_v1_range_partition('v1_task_expression_eval', ((NOW() AT TIME ZONE 'UTC')::DATE + 1));
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
    WHERE i.inhparent = 'v1_task_expression_eval'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_task_expression_eval_original (
            key TEXT NOT NULL,
            task_id BIGINT NOT NULL,
            task_inserted_at TIMESTAMPTZ NOT NULL,
            value_str TEXT,
            value_int INTEGER,
            kind "StepExpressionKind" NOT NULL,
            CONSTRAINT v1_task_expression_eval_pkey PRIMARY KEY (task_id, task_inserted_at, kind, key)
        );

        INSERT INTO v1_task_expression_eval_original SELECT * FROM v1_task_expression_eval;

        DROP TABLE v1_task_expression_eval;
        ALTER TABLE v1_task_expression_eval_original RENAME TO v1_task_expression_eval;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_task_expression_eval DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I SELECT * FROM v1_task_expression_eval ON CONFLICT DO NOTHING', legacy_partition_name);
    DROP TABLE v1_task_expression_eval;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_task_expression_eval', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_task_expression_eval RENAME CONSTRAINT %I TO v1_task_expression_eval_pkey', legacy_partition_name || '_pkey');
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_dag_data'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_dag_data_original (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    input JSONB NOT NULL,
    additional_metadata JSONB,
    CONSTRAINT v1_dag_input_pkey PRIMARY KEY (dag_id, dag_inserted_at)
        );

        INSERT INTO v1_dag_data_original SELECT * FROM v1_dag_data;

        DROP TABLE v1_dag_data;
        ALTER TABLE v1_dag_data_original RENAME TO v1_dag_data;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_dag_data DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I SELECT * FROM v1_dag_data ON CONFLICT DO NOTHING', legacy_partition_name);
    DROP TABLE v1_dag_data;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_dag_data', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_data RENAME CONSTRAINT %I TO v1_dag_input_pkey', legacy_partition_name || '_pkey');
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$
DECLARE
    legacy_partition_name TEXT;
BEGIN
    SELECT c.relname
    INTO legacy_partition_name
    FROM pg_inherits i
    JOIN pg_class c ON c.oid = i.inhrelid
    WHERE i.inhparent = 'v1_dag_to_task'::regclass
    AND pg_get_expr(c.relpartbound, c.oid) LIKE 'FOR VALUES FROM (MINVALUE)%';

    IF legacy_partition_name IS NULL THEN
        CREATE TABLE v1_dag_to_task_original (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT v1_dag_to_task_pkey PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
        );

        INSERT INTO v1_dag_to_task_original SELECT * FROM v1_dag_to_task;

        DROP TABLE v1_dag_to_task;
        ALTER TABLE v1_dag_to_task_original RENAME TO v1_dag_to_task;
        RETURN;
    END IF;

    EXECUTE format('ALTER TABLE v1_dag_to_task DETACH PARTITION %I', legacy_partition_name);
    EXECUTE format('INSERT INTO %I SELECT * FROM v1_dag_to_task ON CONFLICT DO NOTHING', legacy_partition_name);
    DROP TABLE v1_dag_to_task;
    EXECUTE format('ALTER TABLE %I RENAME TO v1_dag_to_task', legacy_partition_name);
    EXECUTE format('ALTER TABLE v1_dag_to_task RENAME CONSTRAINT %I TO v1_dag_to_task_pkey', legacy_partition_name || '_pkey');
END $$;
-- +goose StatementEnd

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
