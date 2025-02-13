
-- CreateTable
CREATE TABLE v2_queue (
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    last_active TIMESTAMP(3),

    CONSTRAINT v2_queue_pkey PRIMARY KEY (tenant_id, name)
);

CREATE TYPE v2_sticky_strategy AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v2_task_initial_state AS ENUM ('QUEUED', 'CANCELLED', 'SKIPPED');

-- CreateTable
CREATE TABLE v2_task (
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
    sticky v2_sticky_strategy NOT NULL,
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
    initial_state v2_task_initial_state NOT NULL DEFAULT 'QUEUED',
    CONSTRAINT v2_task_pkey PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE OR REPLACE FUNCTION create_v2_task_partition(
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
    SELECT format('v2_task_%s', targetDateStr) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE v2_task INCLUDING INDEXES)', newTableName);
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
        format('ALTER TABLE v2_task ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION get_v2_task_partitions_before(
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
        inhparent = 'v2_task'::regclass
        AND substring(inhrelid::regclass::text, 'v2_task_(\d{8})') ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, 'v2_task_(\d{8})')::date) < targetDate;
END;
$$;

SELECT create_v2_task_partition(DATE 'today');

CREATE TYPE v2_task_event_type AS ENUM (
    'COMPLETED',
    'FAILED',
    'CANCELLED',
    'SIGNAL_CREATED',
    'SIGNAL_COMPLETED'
);

-- CreateTable
CREATE TABLE v2_task_event (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    event_type v2_task_event_type NOT NULL,
    event_key TEXT,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB,
    CONSTRAINT v2_task_event_pkey PRIMARY KEY (id)
);

-- Create unique index on (tenant_id, task_id, event_key) when event_key is not null
CREATE UNIQUE INDEX v2_task_event_event_key_unique_idx ON v2_task_event (
    tenant_id ASC,
    task_id ASC,
    event_type ASC,
    event_key ASC
) WHERE event_key IS NOT NULL;

-- CreateTable
CREATE TABLE v2_queue_item (
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
    sticky v2_sticky_strategy NOT NULL,
    desired_worker_id UUID,
    -- TODO: REMOVE is_queued
    is_queued BOOLEAN NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT v2_queue_item_pkey PRIMARY KEY (id)
);

alter table v2_queue_item set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE INDEX v2_queue_item_isQueued_priority_tenantId_queue_id_idx ON v2_queue_item (
    is_queued ASC,
    tenant_id ASC,
    queue ASC,
    priority DESC,
    id ASC
);

-- CreateTable
CREATE TABLE v2_task_runtime (
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    timeout_at TIMESTAMP(3) NOT NULL,

    CONSTRAINT v2_task_runtime_pkey PRIMARY KEY (task_id, retry_count)
);

CREATE INDEX v2_task_runtime_tenantId_workerId_idx ON v2_task_runtime (tenant_id ASC, worker_id ASC);

CREATE INDEX v2_task_runtime_tenantId_timeoutAt_idx ON v2_task_runtime (tenant_id ASC, timeout_at ASC);

alter table v2_task_runtime set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE TYPE v2_match_kind AS ENUM ('TRIGGER', 'SIGNAL');

CREATE TABLE v2_match (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    kind v2_match_kind NOT NULL,
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
    CONSTRAINT v2_match_pkey PRIMARY KEY (id)
);

CREATE TYPE v2_event_type AS ENUM ('USER', 'INTERNAL');

-- Provides information to the caller about the action to take. This is used to differentiate
-- negative conditions from positive conditions. For example, if a task is waiting for a set of
-- tasks to fail, the success of all tasks would be a CANCEL condition, and the failure of any
-- task would be a CREATE condition. Different actions are implicitly different groups of conditions.
CREATE TYPE v2_match_condition_action AS ENUM ('CREATE', 'CANCEL', 'SKIP');

CREATE TABLE v2_match_condition (
    v2_match_id bigint NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    registered_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type v2_event_type NOT NULL,
    event_key TEXT NOT NULL,
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,
    action v2_match_condition_action NOT NULL DEFAULT 'CREATE',
    or_group_id UUID NOT NULL,
    expression TEXT,
    data JSONB,
    CONSTRAINT v2_match_condition_pkey PRIMARY KEY (v2_match_id, id)
);

CREATE INDEX v2_match_condition_filter_idx ON v2_match_condition (
    tenant_id ASC,
    event_type ASC,
    event_key ASC,
    is_satisfied ASC
);

CREATE TABLE v2_dag (
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    CONSTRAINT v2_dag_pkey PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE TABLE v2_dag_to_task (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT v2_dag_to_task_pkey PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

CREATE TABLE v2_dag_data (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    input JSONB NOT NULL,
    additional_metadata JSONB,
    CONSTRAINT v2_dag_input_pkey PRIMARY KEY (dag_id, dag_inserted_at)
);

CREATE OR REPLACE FUNCTION v2_task_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_queue_item (
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
        is_queued,
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
        TRUE,
        retry_count
    FROM new_table
    WHERE 
        initial_state = 'QUEUED';

    INSERT INTO v2_dag_to_task (
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

CREATE TRIGGER v2_task_insert_trigger
AFTER INSERT ON v2_task
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v2_task_insert_function();

CREATE OR REPLACE FUNCTION v2_task_to_v2_queue_item_update_retry_count_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_queue_item (
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
        is_queued,
        retry_count
    )
    VALUES (
        NEW.tenant_id,
        NEW.queue,
        NEW.id,
        NEW.action_id,
        NEW.step_id,
        NEW.workflow_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(NEW.schedule_timeout),
        NEW.step_timeout,
        -- retries are always given priority=4
        4,
        NEW.sticky,
        NEW.desired_worker_id,
        TRUE,
        NEW.retry_count
    );
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v2_task_to_v2_queue_item_update_retry_count_trigger
AFTER UPDATE OF retry_count ON v2_task
FOR EACH ROW
WHEN (OLD.retry_count IS DISTINCT FROM NEW.retry_count)
EXECUTE PROCEDURE v2_task_to_v2_queue_item_update_retry_count_function();

CREATE OR REPLACE FUNCTION create_v2_dag_partition(
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
    SELECT format('v2_dag_%s', targetDateStr) INTO newTableName;
    -- exit if the table exists
    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = newTableName) THEN
        RETURN 0;
    END IF;

    EXECUTE
        format('CREATE TABLE %s (LIKE v2_dag INCLUDING INDEXES)', newTableName);
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
        format('ALTER TABLE v2_dag ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION get_v2_dag_partitions_before(
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
        inhparent = 'v2_dag'::regclass
        AND substring(inhrelid::regclass::text, 'v2_dag_(\d{8})') ~ '^\d{8}'
        AND (substring(inhrelid::regclass::text, 'v2_dag_(\d{8})')::date) < targetDate;
END;
$$;

SELECT create_v2_dag_partition(DATE 'today');
