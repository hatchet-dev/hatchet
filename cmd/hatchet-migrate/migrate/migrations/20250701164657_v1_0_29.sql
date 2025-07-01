-- +goose Up
-- +goose StatementBegin

-- +goose NO TRANSACTION

DO $$
DECLARE
    snapshot_xmin BIGINT;
    partition_record RECORD;
    create_sql TEXT;
    rename_sql TEXT;
    old_name TEXT;
    new_name TEXT;
BEGIN
    SELECT txid_current() INTO snapshot_xmin;

    CREATE TABLE v1_lookup_table_new (
        tenant_id UUID NOT NULL,
        external_id UUID NOT NULL,
        task_id BIGINT,
        dag_id BIGINT,
        inserted_at TIMESTAMPTZ NOT NULL,

        PRIMARY KEY (external_id, inserted_at)
    ) PARTITION BY RANGE(inserted_at);

    FOR partition_record IN
        WITH wk AS (
            SELECT DISTINCT
                DATE_TRUNC('week', inserted_at)::DATE AS week_start
            FROM v1_lookup_table
        )

        SELECT
            'v1_lookup_table_' || TO_CHAR(week_start, 'YYYYMMDD') AS partition_name,
            'FOR VALUES FROM (''' || week_start || ''') TO (''' || (week_start + INTERVAL '1 week')::DATE || ''')' AS partition_bound
        FROM wk
        ORDER BY week_start
    LOOP
        create_sql := format(
            'CREATE TABLE %I PARTITION OF v1_lookup_table_new %s',
            partition_record.partition_name || '_new',
            partition_record.partition_bound
        );
        EXECUTE create_sql;
    END LOOP;

    INSERT INTO v1_lookup_table_new (tenant_id, external_id, task_id, dag_id, inserted_at)
    SELECT tenant_id, external_id, task_id, dag_id, inserted_at
    FROM v1_lookup_table
    WHERE xmin::text::bigint < snapshot_xmin;

    LOCK TABLE v1_lookup_table IN ACCESS EXCLUSIVE MODE;

    INSERT INTO v1_lookup_table_new (tenant_id, external_id, task_id, dag_id, inserted_at)
    SELECT tenant_id, external_id, task_id, dag_id, inserted_at
    FROM v1_lookup_table
    WHERE xmin::text::bigint >= snapshot_xmin;

    DROP TABLE v1_lookup_table;
    ALTER TABLE v1_lookup_table_new RENAME TO v1_lookup_table;

    FOR partition_record IN
        SELECT c.relname as partition_name
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname = 'v1_lookup_table'
          AND c.relname LIKE '%_new'
        ORDER BY c.relname
    LOOP
        old_name := partition_record.partition_name;
        new_name := regexp_replace(old_name, '_new$', '');
        rename_sql := format('ALTER TABLE %I RENAME TO %I', old_name, new_name);
        EXECUTE rename_sql;
    END LOOP;

    FOR partition_record IN
        SELECT
            c.relname as partition_name,
            i.relname as index_name
        FROM pg_class c
        JOIN pg_inherits inh ON c.oid = inh.inhrelid
        JOIN pg_class parent ON inh.inhparent = parent.oid
        JOIN pg_index idx ON c.oid = idx.indrelid
        JOIN pg_class i ON idx.indexrelid = i.oid
        WHERE parent.relname = 'v1_lookup_table'
        AND i.relname LIKE '%_new_%'
        ORDER BY c.relname, i.relname
    LOOP
        old_name := partition_record.index_name;
        new_name := regexp_replace(old_name, '_new_', '_');
        rename_sql := format('ALTER INDEX %I RENAME TO %I', old_name, new_name);
        EXECUTE rename_sql;
    END LOOP;

    ALTER INDEX v1_lookup_table_new_pkey RENAME TO v1_lookup_table_pkey;
END $$;

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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    -- NOTE: this comes after the insert into v1_dag_to_task and v1_lookup_table, because we case on these tables for cleanup
    FOR rec IN SELECT UNNEST(concurrency_parent_strategy_ids) AS parent_strategy_id, workflow_version_id, workflow_run_id FROM new_table WHERE initial_state != 'QUEUED' ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
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
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_task_insert_trigger
AFTER INSERT ON v1_task
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_task_insert_function();

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

CREATE OR REPLACE TRIGGER v1_dag_insert_trigger
AFTER INSERT ON v1_dag
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_dag_insert_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose NO TRANSACTION

DO $$
DECLARE
    snapshot_xmin BIGINT;
    partition_record RECORD;
BEGIN
    SELECT txid_current() INTO snapshot_xmin;

    CREATE TABLE v1_lookup_table_unpartitioned (
        tenant_id UUID NOT NULL,
        external_id UUID NOT NULL,
        task_id BIGINT,
        dag_id BIGINT,
        inserted_at TIMESTAMPTZ NOT NULL,

        PRIMARY KEY (external_id)
    );

    FOR partition_record IN
        SELECT c.relname as partition_name
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname = 'v1_lookup_table'
        ORDER BY c.relname
    LOOP
        EXECUTE format(
            'INSERT INTO v1_lookup_table_unpartitioned (tenant_id, external_id, task_id, dag_id, inserted_at)
             SELECT tenant_id, external_id, task_id, dag_id, inserted_at
             FROM %I
             WHERE xmin::text::bigint < %s',
            partition_record.partition_name,
            snapshot_xmin
        );
    END LOOP;

    LOCK TABLE v1_lookup_table IN ACCESS EXCLUSIVE MODE;

    FOR partition_record IN
        SELECT c.relname as partition_name
        FROM pg_class c
        JOIN pg_inherits i ON c.oid = i.inhrelid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname = 'v1_lookup_table'
        ORDER BY c.relname
    LOOP
        EXECUTE format(
            'INSERT INTO v1_lookup_table_unpartitioned (tenant_id, external_id, task_id, dag_id, inserted_at)
             SELECT tenant_id, external_id, task_id, dag_id, inserted_at
             FROM %I
             WHERE xmin::text::bigint >= %s',
            partition_record.partition_name,
            snapshot_xmin
        );
    END LOOP;

    DROP TABLE v1_lookup_table;

    ALTER TABLE v1_lookup_table_unpartitioned RENAME TO v1_lookup_table;
    ALTER INDEX v1_lookup_table_unpartitioned_pkey RENAME TO v1_lookup_table_pkey;
END $$;

CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $func$
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

    -- NOTE: this comes after the insert into v1_dag_to_task and v1_lookup_table, because we case on these tables for cleanup
    FOR rec IN SELECT UNNEST(concurrency_parent_strategy_ids) AS parent_strategy_id, workflow_version_id, workflow_run_id FROM new_table WHERE initial_state != 'QUEUED' ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
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
$func$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_task_insert_trigger
AFTER INSERT ON v1_task
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_task_insert_function();

CREATE OR REPLACE FUNCTION v1_dag_insert_function()
RETURNS TRIGGER AS
$func2$
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
$func2$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_dag_insert_trigger
AFTER INSERT ON v1_dag
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_dag_insert_function();

-- +goose StatementEnd
