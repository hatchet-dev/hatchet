CREATE OR REPLACE FUNCTION get_v1_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < targetDate;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_range_partition(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT lower(format('%s_%s', targetTableName, targetDateStr)) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

-- https://stackoverflow.com/questions/8137112/unnest-array-by-one-level
CREATE OR REPLACE FUNCTION unnest_nd_1d(a anyarray, OUT a_1d anyarray)
  RETURNS SETOF anyarray
  LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE STRICT AS
$func$
BEGIN                -- null is covered by STRICT
   IF a = '{}' THEN  -- empty
      a_1d = '{}';
      RETURN NEXT;
   ELSE              --  all other cases
      FOREACH a_1d SLICE 1 IN ARRAY a LOOP
         RETURN NEXT;
      END LOOP;
   END IF;
END
$func$;

-- CreateTable
CREATE TABLE v1_queue (
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    last_active TIMESTAMP(3),

    CONSTRAINT v1_queue_pkey PRIMARY KEY (tenant_id, name)
);

CREATE TYPE v1_sticky_strategy AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v1_task_initial_state AS ENUM ('QUEUED', 'CANCELLED', 'SKIPPED', 'FAILED');

-- We need a NONE strategy to allow for tasks which were previously using a concurrency strategy to
-- enqueue if the strategy is removed.
CREATE TYPE v1_concurrency_strategy AS ENUM ('NONE', 'GROUP_ROUND_ROBIN', 'CANCEL_IN_PROGRESS', 'CANCEL_NEWEST');

CREATE TABLE v1_step_concurrency (
    -- We need an id used for stable ordering to prevent deadlocks. We must process all concurrency
    -- strategies on a step in the same order.
    id bigint GENERATED ALWAYS AS IDENTITY,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    step_id UUID NOT NULL,
    -- If the strategy is NONE and we've removed all concurrency slots, we can set is_active to false
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    strategy v1_concurrency_strategy NOT NULL,
    expression TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    max_concurrency INTEGER NOT NULL,
    CONSTRAINT v1_step_concurrency_pkey PRIMARY KEY (workflow_id, workflow_version_id, step_id, id)
);

CREATE OR REPLACE FUNCTION create_v1_step_concurrency()
RETURNS trigger AS $$
BEGIN
  IF NEW."concurrencyGroupExpression" IS NOT NULL THEN
    WITH steps AS (
        -- Select only steps which don't have a parent according to _StepOrder
        SELECT
            s."id",
            wf."id" AS "workflowId",
            wv."id" AS "workflowVersionId",
            wf."tenantId"
        FROM "Step" s
        JOIN "Job" j ON s."jobId" = j."id"
        JOIN "WorkflowVersion" wv ON j."workflowVersionId" = wv."id"
        JOIN "Workflow" wf ON wv."workflowId" = wf."id"
        WHERE
            wv."id" = NEW."workflowVersionId"
            AND j."kind" = 'DEFAULT'
    )
    INSERT INTO v1_step_concurrency (
      workflow_id,
      workflow_version_id,
      step_id,
      strategy,
      expression,
      tenant_id,
      max_concurrency
    )
    SELECT
      s."workflowId",
      s."workflowVersionId",
      s."id",
      NEW."limitStrategy"::VARCHAR::v1_concurrency_strategy,
      NEW."concurrencyGroupExpression",
      s."tenantId",
      NEW."maxRuns"
    FROM steps s;

  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_create_v1_step_concurrency
AFTER INSERT ON "WorkflowConcurrency"
FOR EACH ROW
EXECUTE FUNCTION create_v1_step_concurrency();

-- CreateTable
CREATE TABLE v1_task (
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    step_readable_id TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    schedule_timeout TEXT NOT NULL,
    step_timeout TEXT,
    priority INTEGER DEFAULT 1,
    sticky v1_sticky_strategy NOT NULL,
    desired_worker_id UUID,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    input JSONB NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    internal_retry_count INTEGER NOT NULL DEFAULT 0,
    app_retry_count INTEGER NOT NULL DEFAULT 0,
    additional_metadata JSONB,
    dag_id BIGINT,
    dag_inserted_at TIMESTAMPTZ,
    parent_external_id UUID,
    child_index INTEGER,
    child_key TEXT,
    initial_state v1_task_initial_state NOT NULL DEFAULT 'QUEUED',
    initial_state_reason TEXT,
    concurrency_strategy_ids BIGINT[],
    concurrency_keys TEXT[],
    retry_backoff_factor DOUBLE PRECISION,
    retry_max_backoff INTEGER,
    CONSTRAINT v1_task_pkey PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE TYPE v1_task_event_type AS ENUM (
    'COMPLETED',
    'FAILED',
    'CANCELLED',
    'SIGNAL_CREATED',
    'SIGNAL_COMPLETED'
);

-- CreateTable
CREATE TABLE v1_task_event (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    event_type v1_task_event_type NOT NULL,
    event_key TEXT,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB,
    CONSTRAINT v1_task_event_pkey PRIMARY KEY (id)
);

-- Create unique index on (tenant_id, task_id, event_key) when event_key is not null
CREATE UNIQUE INDEX v1_task_event_event_key_unique_idx ON v1_task_event (
    tenant_id ASC,
    task_id ASC,
    event_type ASC,
    event_key ASC
) WHERE event_key IS NOT NULL;

-- CreateTable
CREATE TABLE v1_queue_item (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id bigint NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    schedule_timeout_at TIMESTAMP(3),
    step_timeout TEXT,
    priority INTEGER NOT NULL DEFAULT 1,
    sticky v1_sticky_strategy NOT NULL,
    desired_worker_id UUID,
    retry_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT v1_queue_item_pkey PRIMARY KEY (id)
);

alter table v1_queue_item set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE INDEX v1_queue_item_list_idx ON v1_queue_item (
    tenant_id ASC,
    queue ASC,
    priority DESC,
    id ASC
);

CREATE INDEX v1_queue_item_task_idx ON v1_queue_item (
    task_id ASC,
    retry_count ASC
);

-- CreateTable
CREATE TABLE v1_task_runtime (
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    timeout_at TIMESTAMP(3) NOT NULL,

    CONSTRAINT v1_task_runtime_pkey PRIMARY KEY (task_id, retry_count)
);

CREATE INDEX v1_task_runtime_tenantId_workerId_idx ON v1_task_runtime (tenant_id ASC, worker_id ASC);

CREATE INDEX v1_task_runtime_tenantId_timeoutAt_idx ON v1_task_runtime (tenant_id ASC, timeout_at ASC);

alter table v1_task_runtime set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE TYPE v1_match_kind AS ENUM ('TRIGGER', 'SIGNAL');

CREATE TABLE v1_match (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    kind v1_match_kind NOT NULL,
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,
    signal_target_id bigint,
    signal_key TEXT,
    -- references the parent DAG for the task, which we can use to get input + additional metadata
    trigger_dag_id bigint,
    trigger_dag_inserted_at timestamptz,
    -- references the step id to instantiate the task
    trigger_step_id UUID,
    -- references the external id for the new task
    trigger_external_id UUID,
    CONSTRAINT v1_match_pkey PRIMARY KEY (id)
);

CREATE TYPE v1_event_type AS ENUM ('USER', 'INTERNAL');

-- Provides information to the caller about the action to take. This is used to differentiate
-- negative conditions from positive conditions. For example, if a task is waiting for a set of
-- tasks to fail, the success of all tasks would be a CANCEL condition, and the failure of any
-- task would be a CREATE condition. Different actions are implicitly different groups of conditions.
CREATE TYPE v1_match_condition_action AS ENUM ('CREATE', 'CANCEL', 'SKIP');

CREATE TABLE v1_match_condition (
    v1_match_id bigint NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    registered_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type v1_event_type NOT NULL,
    event_key TEXT NOT NULL,
    -- readable_data_key is used as the key when constructing the aggregated data for the v1_match
    readable_data_key TEXT NOT NULL,
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,
    action v1_match_condition_action NOT NULL DEFAULT 'CREATE',
    or_group_id UUID NOT NULL,
    expression TEXT,
    data JSONB,
    CONSTRAINT v1_match_condition_pkey PRIMARY KEY (v1_match_id, id)
);

CREATE INDEX v1_match_condition_filter_idx ON v1_match_condition (
    tenant_id ASC,
    event_type ASC,
    event_key ASC,
    is_satisfied ASC
);

CREATE TABLE v1_dag (
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    CONSTRAINT v1_dag_pkey PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE TABLE v1_dag_to_task (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT v1_dag_to_task_pkey PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

CREATE TABLE v1_dag_data (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    input JSONB NOT NULL,
    additional_metadata JSONB,
    CONSTRAINT v1_dag_input_pkey PRIMARY KEY (dag_id, dag_inserted_at)
);

-- CreateTable
CREATE TABLE v1_concurrency_slot (
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    task_retry_count INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    strategy_id BIGINT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 1,
    key TEXT NOT NULL,
    is_filled BOOLEAN NOT NULL DEFAULT FALSE,
    next_strategy_ids BIGINT[],
    next_keys TEXT[],
    queue_to_notify TEXT NOT NULL,
    schedule_timeout_at TIMESTAMP(3) NOT NULL,
    CONSTRAINT v1_concurrency_slot_pkey PRIMARY KEY (task_id, task_inserted_at, task_retry_count, strategy_id)
) PARTITION BY RANGE(task_inserted_at);

CREATE INDEX v1_concurrency_slot_query_idx ON v1_concurrency_slot (tenant_id, strategy_id ASC, key ASC, priority DESC);

CREATE OR REPLACE FUNCTION delete_concurrency_slots_on_v1_task_runtime_delete()
RETURNS trigger AS $$
BEGIN
  WITH slots_to_delete AS (
    SELECT
        cs.task_inserted_at, cs.task_id, cs.task_retry_count, cs.key
    FROM
        deleted_rows d
    JOIN
        v1_task t ON t.id = d.task_id
    JOIN v1_concurrency_slot cs ON cs.task_id = t.id AND cs.task_inserted_at = t.inserted_at AND cs.task_retry_count = d.retry_count
  )
  DELETE FROM
    v1_concurrency_slot cs
  WHERE
    (task_inserted_at, task_id, task_retry_count, key) IN (
        SELECT
            task_inserted_at, task_id, task_retry_count, key
        FROM
            slots_to_delete
    );

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_task_runtime_delete
AFTER DELETE ON v1_task_runtime
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION delete_concurrency_slots_on_v1_task_runtime_delete();

CREATE TABLE v1_retry_queue_item (
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    task_retry_count INTEGER NOT NULL,
    retry_after TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,

    CONSTRAINT v1_retry_queue_item_pkey PRIMARY KEY (task_id, task_inserted_at, task_retry_count)
);

CREATE INDEX v1_retry_queue_item_tenant_id_retry_after_idx ON v1_retry_queue_item (tenant_id ASC, retry_after ASC);

CREATE OR REPLACE FUNCTION v1_task_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_slot_rows AS (
        SELECT
            id,
            inserted_at,
            retry_count,
            tenant_id,
            priority,
            concurrency_strategy_ids[1] AS strategy_id,
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
            queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
        FROM new_table
        WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        workflow_id,
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
        tenant_id,
        workflow_id,
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
        action_id,
        step_id,
        workflow_id,
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
        action_id,
        step_id,
        workflow_id,
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

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_task_insert_trigger
AFTER INSERT ON v1_task
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_task_insert_function();

CREATE OR REPLACE FUNCTION v1_task_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_slot_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            nt.priority,
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
            nt.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout) AS schedule_timeout_at
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            AND nt.concurrency_strategy_ids[1] IS NOT NULL
            AND ot.retry_count IS DISTINCT FROM nt.retry_count
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        tenant_id,
        workflow_id,
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
        tenant_id,
        workflow_id,
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
        action_id,
        step_id,
        workflow_id,
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
        nt.action_id,
        nt.step_id,
        nt.workflow_id,
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
        AND ot.retry_count IS DISTINCT FROM nt.retry_count;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_task_update_trigger
AFTER UPDATE ON v1_task
REFERENCING NEW TABLE AS new_table OLD TABLE AS old_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_task_update_function();

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
        tenant_id,
        workflow_id,
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
        tenant_id,
        workflow_id,
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
        action_id,
        step_id,
        workflow_id,
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
        action_id,
        step_id,
        workflow_id,
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

CREATE TRIGGER v1_concurrency_slot_update_trigger
AFTER UPDATE ON v1_concurrency_slot
REFERENCING NEW TABLE AS new_table OLD TABLE AS old_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_concurrency_slot_update_function();

CREATE TYPE v1_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v1_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'COMPLETED',
    'CANCELLED',
    'FAILED'
);

-- HELPER FUNCTIONS FOR PARTITIONED TABLES --
CREATE OR REPLACE FUNCTION create_v1_partition_with_status(
    newTableName text,
    status v1_readable_status_olap
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetNameWithStatus varchar;
BEGIN
    SELECT lower(format('%s_%s', newTableName, status::text)) INTO targetNameWithStatus;

    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = targetNameWithStatus) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES)', targetNameWithStatus, newTableName);
    EXECUTE
        format('ALTER TABLE %s SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', targetNameWithStatus);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES IN (''%s'')', newTableName, targetNameWithStatus, status);
    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_olap_partition_with_date_and_status(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneDayStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(targetDate, 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(targetDate + INTERVAL '1 day', 'YYYYMMDD') INTO targetDatePlusOneDayStr;
    SELECT format('%s_%s', targetTableName, targetDateStr) INTO newTableName;
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        EXECUTE format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES) PARTITION BY LIST (readable_status)', newTableName, targetTableName);
    END IF;

    PERFORM create_v1_partition_with_status(newTableName, 'QUEUED');
    PERFORM create_v1_partition_with_status(newTableName, 'RUNNING');
    PERFORM create_v1_partition_with_status(newTableName, 'COMPLETED');
    PERFORM create_v1_partition_with_status(newTableName, 'CANCELLED');
    PERFORM create_v1_partition_with_status(newTableName, 'FAILED');

    -- If it's not already attached, attach the partition
    IF NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = newTableName::regclass) THEN
        EXECUTE format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    END IF;

    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION get_v1_partitions_before_date(
    targetTableName text,
    targetDate date
) RETURNS TABLE(partition_name text)
    LANGUAGE plpgsql AS
$$
BEGIN
    RETURN QUERY
    SELECT
        inhrelid::regclass::text AS partition_name
    FROM
        pg_inherits
    WHERE
        inhparent = targetTableName::regclass
        AND substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < targetDate;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_hash_partitions(
    targetTableName text,
    num_partitions INT
) RETURNS integer
LANGUAGE plpgsql AS
$$
DECLARE
    existing_count INT;
    partition_name text;
    created_count INT := 0;
    i INT;
BEGIN
    SELECT count(*) INTO existing_count
    FROM pg_inherits
    WHERE inhparent = targetTableName::regclass;

    IF existing_count > num_partitions THEN
        RAISE EXCEPTION 'Cannot decrease the number of partitions: we already have % partitions which is more than the target %', existing_count, num_partitions;
    END IF;

    FOR i IN 0..(num_partitions - 1) LOOP
        partition_name := format('%s_%s', targetTableName, i);
        IF to_regclass(partition_name) IS NULL THEN
            EXECUTE format('CREATE TABLE %I (LIKE %s INCLUDING INDEXES)', partition_name, targetTableName);
            EXECUTE format('ALTER TABLE %I SET (
                autovacuum_vacuum_scale_factor = ''0.1'',
                autovacuum_analyze_scale_factor = ''0.05'',
                autovacuum_vacuum_threshold = ''25'',
                autovacuum_analyze_threshold = ''25'',
                autovacuum_vacuum_cost_delay = ''10'',
                autovacuum_vacuum_cost_limit = ''1000''
            )', partition_name);
            EXECUTE format('ALTER TABLE %s ATTACH PARTITION %I FOR VALUES WITH (modulus %s, remainder %s)', targetTableName, partition_name, num_partitions, i);
            created_count := created_count + 1;
        END IF;
    END LOOP;
    RETURN created_count;
END;
$$;

-- TASKS DEFINITIONS --
CREATE TABLE v1_tasks_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    queue TEXT NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    schedule_timeout TEXT NOT NULL,
    step_timeout TEXT,
    priority INTEGER DEFAULT 1,
    sticky v1_sticky_strategy_olap NOT NULL,
    desired_worker_id UUID,
    display_name TEXT NOT NULL,
    input JSONB NOT NULL,
    additional_metadata JSONB,
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    latest_retry_count INT NOT NULL DEFAULT 0,
    latest_worker_id UUID,
    dag_id BIGINT,
    dag_inserted_at TIMESTAMPTZ,

    PRIMARY KEY (inserted_at, id, readable_status)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_tasks_olap_workflow_id_idx ON v1_tasks_olap (tenant_id, workflow_id);

CREATE INDEX v1_tasks_olap_worker_id_idx ON v1_tasks_olap (tenant_id, latest_worker_id) WHERE latest_worker_id IS NOT NULL;

SELECT create_v1_olap_partition_with_date_and_status('v1_tasks_olap', CURRENT_DATE);

-- DAG DEFINITIONS --
CREATE TABLE v1_dags_olap (
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    input JSONB NOT NULL,
    additional_metadata JSONB,
    PRIMARY KEY (inserted_at, id, readable_status)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_dags_olap_workflow_id_idx ON v1_dags_olap (tenant_id, workflow_id);

SELECT create_v1_olap_partition_with_date_and_status('v1_dags_olap', CURRENT_DATE);

-- RUN DEFINITIONS --
CREATE TYPE v1_run_kind AS ENUM ('TASK', 'DAG');

-- v1_runs_olap represents an invocation of a workflow. it can either refer to a DAG or a task.
-- we partition this table on status to allow for efficient querying of tasks in different states.
CREATE TABLE v1_runs_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    kind v1_run_kind NOT NULL,
    workflow_id UUID NOT NULL,
    additional_metadata JSONB,

    PRIMARY KEY (inserted_at, id, readable_status, kind)
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_olap_partition_with_date_and_status('v1_runs_olap', CURRENT_DATE);

-- LOOKUP TABLES --
CREATE TABLE v1_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id)
);

CREATE TABLE v1_dag_to_task_olap (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

-- STATUS DEFINITION --
CREATE TYPE v1_status_kind AS ENUM ('TASK', 'DAG');

CREATE TABLE v1_statuses_olap (
    external_id UUID NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    kind v1_run_kind NOT NULL,
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',

    PRIMARY KEY (external_id, inserted_at)
);


-- EVENT DEFINITIONS --
CREATE TYPE v1_event_type_olap AS ENUM (
    'RETRYING',
    'REASSIGNED',
    'RETRIED_BY_USER',
    'CREATED',
    'QUEUED',
    'REQUEUED_NO_WORKER',
    'REQUEUED_RATE_LIMIT',
    'ASSIGNED',
    'ACKNOWLEDGED',
    'SENT_TO_WORKER',
    'SLOT_RELEASED',
    'STARTED',
    'TIMEOUT_REFRESHED',
    'SCHEDULING_TIMED_OUT',
    'FINISHED',
    'FAILED',
    'CANCELLED',
    'TIMED_OUT',
    'RATE_LIMIT_ERROR',
    'SKIPPED'
);

-- this is a hash-partitioned table on the task_id, so that we can process batches of events in parallel
-- without needing to place conflicting locks on tasks.
CREATE TABLE v1_task_events_olap_tmp (
    tenant_id UUID NOT NULL,
    requeue_after TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requeue_retries INT NOT NULL DEFAULT 0,
    id bigint GENERATED ALWAYS AS IDENTITY,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    event_type v1_event_type_olap NOT NULL,
    readable_status v1_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    worker_id UUID,

    PRIMARY KEY (tenant_id, requeue_after, task_id, id)
) PARTITION BY HASH(task_id);

CREATE OR REPLACE FUNCTION list_task_events_tmp(
    partition_number INT,
    tenant_id UUID,
    event_limit INT
) RETURNS SETOF v1_task_events_olap_tmp
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := 'v1_task_events_olap_tmp_' || partition_number::text;
    RETURN QUERY EXECUTE format(
        'SELECT e.*
         FROM %I e
         WHERE e.tenant_id = $1
           AND e.requeue_after <= CURRENT_TIMESTAMP
         ORDER BY e.requeue_after, e.task_id, e.id
         LIMIT $2
         FOR UPDATE SKIP LOCKED',
         partition_table)
    USING tenant_id, event_limit;
END;
$$;

CREATE TABLE v1_task_events_olap (
    tenant_id UUID NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    event_type v1_event_type_olap NOT NULL,
    workflow_id UUID NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    readable_status v1_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    output JSONB,
    worker_id UUID,
    additional__event_data TEXT,
    additional__event_message TEXT,

    PRIMARY KEY (task_id, task_inserted_at, id)
);

CREATE INDEX v1_task_events_olap_task_id_idx ON v1_task_events_olap (task_id);

-- this is a hash-partitioned table on the dag_id, so that we can process batches of events in parallel
-- without needing to place conflicting locks on dags.
CREATE TABLE v1_task_status_updates_tmp (
    tenant_id UUID NOT NULL,
    requeue_after TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requeue_retries INT NOT NULL DEFAULT 0,
    id bigint GENERATED ALWAYS AS IDENTITY,
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (tenant_id, requeue_after, dag_id, id)
) PARTITION BY HASH(dag_id);

CREATE OR REPLACE FUNCTION list_task_status_updates_tmp(
    partition_number INT,
    tenant_id UUID,
    event_limit INT
) RETURNS SETOF v1_task_status_updates_tmp
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := 'v1_task_status_updates_tmp_' || partition_number::text;
    RETURN QUERY EXECUTE format(
        'SELECT e.*
         FROM %I e
         WHERE e.tenant_id = $1
           AND e.requeue_after <= CURRENT_TIMESTAMP
         ORDER BY e.dag_id
         LIMIT $2
         FOR UPDATE SKIP LOCKED',
         partition_table)
    USING tenant_id, event_limit;
END;
$$;

-- TRIGGERS TO LINK TASKS, DAGS AND EVENTS --
CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        additional_metadata
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id,
        additional_metadata
    FROM new_rows
    WHERE dag_id IS NULL;

    INSERT INTO v1_lookup_table (
        tenant_id,
        external_id,
        task_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows;

    -- If the task has a dag_id and dag_inserted_at, insert into the lookup table
    INSERT INTO v1_dag_to_task_olap (
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
    FROM new_rows
    WHERE dag_id IS NOT NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_insert_trigger
AFTER INSERT ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_insert_function();

CREATE OR REPLACE FUNCTION v1_tasks_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_runs_olap r
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'TASK';

    -- insert tmp events into task status updates table if we have a dag_id
    INSERT INTO v1_task_status_updates_tmp (
        tenant_id,
        dag_id,
        dag_inserted_at
    )
    SELECT
        tenant_id,
        dag_id,
        dag_inserted_at
    FROM new_rows
    WHERE dag_id IS NOT NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_update_trigger
AFTER UPDATE ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_status_update_function();

CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id,
        additional_metadata
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id,
        additional_metadata
    FROM new_rows;

    INSERT INTO v1_lookup_table (
        tenant_id,
        external_id,
        dag_id,
        inserted_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_dags_olap_status_insert_trigger
AFTER INSERT ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_insert_function();

CREATE OR REPLACE FUNCTION v1_dags_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_runs_olap r
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'DAG';

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_dags_olap_status_update_trigger
AFTER UPDATE ON v1_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_status_update_function();

CREATE OR REPLACE FUNCTION v1_runs_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_statuses_olap (
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    )
    SELECT
        external_id,
        inserted_at,
        tenant_id,
        workflow_id,
        kind,
        readable_status
    FROM new_rows;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_runs_olap_status_insert_trigger
AFTER INSERT ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_insert_function();

CREATE OR REPLACE FUNCTION v1_runs_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v1_statuses_olap s
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        s.external_id = n.external_id
        AND s.inserted_at = n.inserted_at;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_runs_olap_status_update_trigger
AFTER UPDATE ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_runs_olap_status_update_function();
