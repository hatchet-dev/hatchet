-- +goose no transaction
-- +goose Up
-- +goose StatementBegin
BEGIN;
CREATE TABLE IF NOT EXISTS v1_lookup_table_partitioned (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);

-- check if partitions already exist, and create them if not
DO $$
DECLARE
    partition_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO partition_count
    FROM pg_class c
    JOIN pg_inherits i ON c.oid = i.inhrelid
    JOIN pg_class parent ON i.inhparent = parent.oid
    WHERE
        parent.relname = 'v1_lookup_table_partitioned'
        AND c.relkind = 'r';

    IF partition_count > 0 THEN
        RAISE NOTICE 'Table has % partitions. Exiting.', partition_count;
        RETURN;
    END IF;

    RAISE NOTICE 'No partitions found. Creating partitions...';

    PERFORM create_v1_weekly_range_partition('v1_lookup_table_partitioned', NOW()::DATE);
    PERFORM create_v1_weekly_range_partition('v1_lookup_table_partitioned', (NOW() - INTERVAL '1 week')::DATE);
END $$;


DO $$
DECLARE
    partition_count INTEGER;
    start_date_str varchar;
    end_date_str varchar;
    target_table_name CONSTANT varchar := 'v1_lookup_table_partitioned';
    new_table_name varchar;
BEGIN
    -- if the partition containing the old data already exists, exit
    SELECT COUNT(*)
    INTO partition_count
    FROM pg_class c
    JOIN pg_inherits i ON c.oid = i.inhrelid
    JOIN pg_class parent ON i.inhparent = parent.oid
    WHERE
        c.relname = 'v1_lookup_table_partitioned_19700101';

    IF partition_count > 0 THEN
        RAISE NOTICE 'Table has % partitions. Exiting.', partition_count;
        RETURN;
    END IF;

    SELECT '19700101' INTO start_date_str;
    SELECT TO_CHAR(date_trunc('week', (NOW() - INTERVAL '8 days')::DATE), 'YYYYMMDD') INTO end_date_str;
    SELECT LOWER(FORMAT('%s_%s', target_table_name, start_date_str)) INTO new_table_name;

    EXECUTE
        format('CREATE TABLE IF NOT EXISTS %s (LIKE %s INCLUDING INDEXES)', new_table_name, target_table_name);

    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', new_table_name);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', target_table_name, new_table_name, start_date_str, end_date_str);
END$$;


CREATE OR REPLACE FUNCTION v1_lookup_table_partitioned_insert_function()
RETURNS TRIGGER AS
$$
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
        id,
        dag_id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER v1_lookup_table_partitioned_insert_trigger
AFTER INSERT ON v1_lookup_table
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_lookup_table_partitioned_insert_function();

COMMIT;
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
BEGIN;
DROP TABLE IF EXISTS v1_lookup_table;
ALTER TABLE v1_lookup_table_partitioned
    RENAME TO v1_lookup_table;

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

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;



ALTER INDEX v1_lookup_table_partitioned_pkey
    RENAME TO v1_lookup_table_pkey;

DROP TRIGGER IF EXISTS v1_lookup_table_partitioned_insert_trigger ON v1_lookup_table_partitioned;
DROP FUNCTION IF EXISTS v1_lookup_table_partitioned_insert_function;
COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
CREATE TABLE IF NOT EXISTS v1_lookup_table_original (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id)
);
COMMIT;
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
FROM v1_task;


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
FROM v1_dag;
-- +goose StatementEnd

-- +goose StatementBegin
BEGIN;
DROP TABLE IF EXISTS v1_lookup_table;
ALTER TABLE v1_lookup_table_original
    RENAME TO v1_lookup_table;

ALTER INDEX v1_lookup_table_original_pkey
    RENAME TO v1_lookup_table_pkey;

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

COMMIT;
-- +goose StatementEnd
