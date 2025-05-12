CREATE TYPE v1_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v1_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'CANCELLED',
    'FAILED',
    'COMPLETED'
);

-- HELPER FUNCTIONS FOR PARTITIONED TABLES --
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
    workflow_version_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
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
    parent_task_external_id UUID,

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
    parent_task_external_id UUID,
    total_tasks INT NOT NULL DEFAULT 1,
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
    workflow_version_id UUID NOT NULL,
    additional_metadata JSONB,
    parent_task_external_id UUID,

    PRIMARY KEY (inserted_at, id, readable_status, kind)
) PARTITION BY RANGE(inserted_at);

SELECT create_v1_olap_partition_with_date_and_status('v1_runs_olap', CURRENT_DATE);

CREATE INDEX ix_v1_runs_olap_parent_task_external_id ON v1_runs_olap (parent_task_external_id) WHERE parent_task_external_id IS NOT NULL;

-- LOOKUP TABLES --
CREATE TABLE v1_lookup_table_olap (
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

-- Events tables
CREATE TABLE v1_events_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    seen_at TIMESTAMPTZ NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    additional_metadata JSONB,

    PRIMARY KEY (tenant_id, id, seen_at)
) PARTITION BY RANGE(seen_at);

CREATE INDEX v1_events_olap_key_idx ON v1_events_olap (tenant_id, key);

CREATE TABLE v1_event_lookup_table_olap (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (tenant_id, external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

CREATE TABLE v1_event_to_run_olap (
    run_id BIGINT NOT NULL,
    run_inserted_at TIMESTAMPTZ NOT NULL,
    event_id UUID NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (event_id, event_seen_at, run_id, run_inserted_at)
) PARTITION BY RANGE(event_seen_at);


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
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    FROM new_rows
    WHERE dag_id IS NULL;

    INSERT INTO v1_lookup_table_olap (
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
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

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
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id,
        workflow_version_id,
        additional_metadata,
        parent_task_external_id
    FROM new_rows;

    INSERT INTO v1_lookup_table_olap (
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
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

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

CREATE OR REPLACE FUNCTION v1_events_lookup_table_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table_olap (
        tenant_id,
        external_id,
        event_id,
        event_seen_at
    )
    SELECT
        tenant_id,
        external_id,
        id,
        inserted_at
    FROM new_rows
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_olap_insert_trigger
AFTER INSERT ON v1_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_events_lookup_table_olap_insert_function();