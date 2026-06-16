-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION rename_partitions(
    parent_table_name TEXT,
    new_prefix TEXT
)
RETURNS TABLE(old_name TEXT, new_name TEXT)
LANGUAGE plpgsql
AS $$
DECLARE
    partition_record RECORD;
    old_partition_name TEXT;
    new_partition_name TEXT;
    partition_suffix TEXT;
BEGIN
    FOR partition_record IN
        SELECT c.relname AS partition_name
        FROM pg_inherits i
        JOIN pg_class c ON i.inhrelid = c.oid
        JOIN pg_namespace n ON c.relnamespace = n.oid
        JOIN pg_class parent ON i.inhparent = parent.oid
        WHERE parent.relname = parent_table_name
        ORDER BY c.relname
    LOOP
        old_partition_name := partition_record.partition_name;
        partition_suffix := replace(old_partition_name, parent_table_name || '_', '');
        new_partition_name := new_prefix || '_' || partition_suffix;

        EXECUTE format('ALTER TABLE %I RENAME TO %I', old_partition_name, new_partition_name);
        EXECUTE format('ALTER INDEX %I RENAME TO %I',
            old_partition_name || '_pkey',
            new_partition_name || '_pkey'
        );

        RETURN NEXT;
        RAISE NOTICE 'Renamed: % -> %', old_partition_name, new_partition_name;
    END LOOP;

    RETURN;
END;
$$;

CREATE TABLE IF NOT EXISTS v1_lookup_table_partitioned (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);

DO $$
DECLARE
    oldest_date DATE;
    d DATE;
    partition_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO partition_count
    FROM pg_class c
    JOIN pg_inherits i ON c.oid = i.inhrelid
    JOIN pg_class parent ON i.inhparent = parent.oid
    WHERE parent.relname = 'v1_lookup_table_partitioned'
    AND c.relkind = 'r';

    IF partition_count > 0 THEN
        RAISE NOTICE 'Partitions already exist for v1_lookup_table_partitioned, skipping creation';
        RETURN;
    END IF;

    SELECT (WITH t AS (SELECT * FROM v1_task ORDER BY id LIMIT 1) SELECT inserted_at::DATE FROM t)
    INTO oldest_date;

    IF oldest_date IS NULL THEN
        oldest_date := (NOW() - INTERVAL '30 day')::DATE;
    END IF;

    d := oldest_date;
    WHILE d <= (NOW() + INTERVAL '1 day')::DATE LOOP
        PERFORM create_v1_weekly_range_partition('v1_lookup_table_partitioned', d);
        d := d + INTERVAL '1 day';
    END LOOP;
END $$;

-- Replication trigger: inserts into old table are forwarded to the new partitioned table
-- during the backfill window so no writes are lost between backfill and cutover.
CREATE OR REPLACE FUNCTION v1_lookup_table_partitioned_insert_function()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO v1_lookup_table_partitioned (
        tenant_id,
        external_id,
        task_id,
        dag_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        task_id,
        dag_id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_lookup_table_partitioned_insert_trigger
AFTER INSERT ON v1_lookup_table
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_lookup_table_partitioned_insert_function();
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO v1_lookup_table_partitioned (
    tenant_id,
    external_id,
    task_id,
    dag_id,
    inserted_at
)
SELECT
    tenant_id,
    external_id,
    id,
    NULL::BIGINT,
    inserted_at
FROM v1_task
ON CONFLICT (external_id, inserted_at) DO NOTHING;

INSERT INTO v1_lookup_table_partitioned (
    tenant_id,
    external_id,
    task_id,
    dag_id,
    inserted_at
)
SELECT
    tenant_id,
    external_id,
    NULL::BIGINT,
    id,
    inserted_at
FROM v1_dag
ON CONFLICT (external_id, inserted_at) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS v1_lookup_table;
DROP FUNCTION IF EXISTS v1_lookup_table_partitioned_insert_function;

SELECT rename_partitions('v1_lookup_table_partitioned', 'v1_lookup_table');

ALTER TABLE v1_lookup_table_partitioned RENAME TO v1_lookup_table;
ALTER INDEX v1_lookup_table_partitioned_pkey RENAME TO v1_lookup_table_pkey;

-- Update v1_task_insert_function to use new composite PK (external_id, inserted_at)
CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
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
CREATE TABLE IF NOT EXISTS v1_lookup_table_original (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (external_id)
);
-- +goose StatementEnd

-- +goose StatementBegin
INSERT INTO v1_lookup_table_original (
    tenant_id,
    external_id,
    task_id,
    dag_id,
    inserted_at
)
SELECT
    tenant_id,
    external_id,
    id,
    NULL::BIGINT,
    inserted_at
FROM v1_task
ON CONFLICT (external_id) DO NOTHING;

INSERT INTO v1_lookup_table_original (
    tenant_id,
    external_id,
    task_id,
    dag_id,
    inserted_at
)
SELECT
    tenant_id,
    external_id,
    NULL::BIGINT,
    id,
    inserted_at
FROM v1_dag
ON CONFLICT (external_id) DO NOTHING;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS v1_lookup_table;

ALTER TABLE v1_lookup_table_original RENAME TO v1_lookup_table;
ALTER INDEX v1_lookup_table_original_pkey RENAME TO v1_lookup_table_pkey;

CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
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

DROP FUNCTION IF EXISTS rename_partitions(TEXT, TEXT);
-- +goose StatementEnd
