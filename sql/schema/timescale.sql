CREATE TYPE v2_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v2_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'COMPLETED',
    'CANCELLED',
    'FAILED'
);

-- HELPER FUNCTIONS FOR PARTITIONED TABLES --
CREATE OR REPLACE FUNCTION create_v2_partition_with_status(
    newTableName text,
    status v2_readable_status_olap
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

CREATE OR REPLACE FUNCTION create_v2_olap_partition_with_date_and_status(
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

    PERFORM create_v2_partition_with_status(newTableName, 'QUEUED');
    PERFORM create_v2_partition_with_status(newTableName, 'RUNNING');
    PERFORM create_v2_partition_with_status(newTableName, 'COMPLETED');
    PERFORM create_v2_partition_with_status(newTableName, 'CANCELLED');
    PERFORM create_v2_partition_with_status(newTableName, 'FAILED');

    -- If it's not already attached, attach the partition
    IF NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhrelid = newTableName::regclass) THEN
        EXECUTE format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    END IF;

    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION get_v2_partitions_before_date(
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

CREATE OR REPLACE FUNCTION create_v2_hash_partitions(
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
CREATE TABLE v2_tasks_olap (
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
    sticky v2_sticky_strategy_olap NOT NULL,
    desired_worker_id UUID,
    display_name TEXT NOT NULL,
    input JSONB NOT NULL,
    additional_metadata JSONB,
    readable_status v2_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    latest_retry_count INT NOT NULL DEFAULT 0,
    latest_worker_id UUID,
    dag_id BIGINT,
    dag_inserted_at TIMESTAMPTZ,

    PRIMARY KEY (inserted_at, id, readable_status)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v2_tasks_olap_workflow_id_idx ON v2_tasks_olap (tenant_id, workflow_id);

CREATE INDEX v2_tasks_olap_worker_id_idx ON v2_tasks_olap (tenant_id, latest_worker_id) WHERE latest_worker_id IS NOT NULL;

SELECT create_v2_olap_partition_with_date_and_status('v2_tasks_olap', CURRENT_DATE);

-- DAG DEFINITIONS --
CREATE TABLE v2_dags_olap (
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    readable_status v2_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    input JSONB NOT NULL,
    additional_metadata JSONB,
    PRIMARY KEY (inserted_at, id, readable_status)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v2_dags_olap_workflow_id_idx ON v2_dags_olap (tenant_id, workflow_id);

SELECT create_v2_olap_partition_with_date_and_status('v2_dags_olap', CURRENT_DATE);

-- RUN DEFINITIONS --
CREATE TYPE v2_run_kind AS ENUM ('TASK', 'DAG');

-- v2_runs_olap represents an invocation of a workflow. it can either refer to a DAG or a task.
-- we partition this table on status to allow for efficient querying of tasks in different states.
CREATE TABLE v2_runs_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    readable_status v2_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    kind v2_run_kind NOT NULL,
    workflow_id UUID NOT NULL,

    PRIMARY KEY (inserted_at, id, readable_status, kind)
) PARTITION BY RANGE(inserted_at);

SELECT create_v2_olap_partition_with_date_and_status('v2_runs_olap', CURRENT_DATE);

-- LOOKUP TABLES --
CREATE TABLE v2_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id)
);

CREATE TABLE v2_dag_to_task_olap (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
);

-- STATUS DEFINITION --
CREATE TYPE v2_status_kind AS ENUM ('TASK', 'DAG');

CREATE TABLE v2_statuses_olap (
    external_id UUID NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    kind v2_run_kind NOT NULL,
    readable_status v2_readable_status_olap NOT NULL DEFAULT 'QUEUED',

    PRIMARY KEY (external_id, inserted_at)
);

SELECT * from create_hypertable('v2_statuses_olap', by_range('inserted_at',  INTERVAL '1 day'));

CREATE  MATERIALIZED VIEW v2_cagg_status_metrics
   WITH (timescaledb.continuous, timescaledb.materialized_only = false)
   AS
      SELECT
        time_bucket('5 minutes', inserted_at) AS bucket,
        tenant_id,
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED') AS queued_count,
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING') AS running_count,
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED') AS cancelled_count,
        COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count
      FROM v2_statuses_olap
      GROUP BY tenant_id, workflow_id, bucket
      ORDER BY bucket DESC
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_status_metrics',
  start_offset => NULL,
  end_offset => INTERVAL '5 minutes',
  schedule_interval => INTERVAL '15 seconds');

-- EVENT DEFINITIONS --
CREATE TYPE v2_event_type_olap AS ENUM (
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
CREATE TABLE v2_task_events_olap_tmp (
    tenant_id UUID NOT NULL,
    requeue_after TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requeue_retries INT NOT NULL DEFAULT 0,
    id bigint GENERATED ALWAYS AS IDENTITY,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    event_type v2_event_type_olap NOT NULL,
    readable_status v2_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    worker_id UUID,

    PRIMARY KEY (tenant_id, requeue_after, task_id, id)
) PARTITION BY HASH(task_id);

CREATE OR REPLACE FUNCTION list_task_events_tmp(
    partition_number INT,
    tenant_id UUID,
    event_limit INT
) RETURNS SETOF v2_task_events_olap_tmp
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := 'v2_task_events_olap_tmp_' || partition_number::text;
    RETURN QUERY EXECUTE format(
        'SELECT e.*
         FROM %I e
         WHERE e.tenant_id = $1
           AND e.requeue_after <= CURRENT_TIMESTAMP
         ORDER BY e.task_id
         LIMIT $2
         FOR UPDATE SKIP LOCKED',
         partition_table)
    USING tenant_id, event_limit;
END;
$$;

CREATE TABLE v2_task_events_olap (
    tenant_id UUID NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    event_type v2_event_type_olap NOT NULL,
    workflow_id UUID NOT NULL,
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    readable_status v2_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    output JSONB,
    worker_id UUID,
    additional__event_data TEXT,
    additional__event_message TEXT,

    PRIMARY KEY (task_id, task_inserted_at, id)
);

SELECT * from create_hypertable('v2_task_events_olap', by_range('task_inserted_at',  INTERVAL '1 day'));

CREATE INDEX v2_task_events_olap_task_id_idx ON v2_task_events_olap (task_id);

CREATE MATERIALIZED VIEW v2_cagg_task_events_minute
WITH (timescaledb.continuous, timescaledb.materialized_only = false) AS
SELECT
    time_bucket('1 minute', task_inserted_at) AS bucket,
    tenant_id,
    workflow_id,
    COUNT(*) FILTER (WHERE readable_status = 'QUEUED') AS queued_count,
    COUNT(*) FILTER (WHERE readable_status = 'RUNNING') AS running_count,
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
    COUNT(*) FILTER (WHERE readable_status = 'CANCELLED') AS cancelled_count,
    COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count
FROM v2_task_events_olap
GROUP BY bucket, tenant_id, workflow_id
ORDER BY bucket
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_task_events_minute',
  start_offset => NULL,
  end_offset => INTERVAL '1 minute',
  schedule_interval => INTERVAL '15 seconds');

-- this is a hash-partitioned table on the dag_id, so that we can process batches of events in parallel
-- without needing to place conflicting locks on dags.
CREATE TABLE v2_task_status_updates_tmp (
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
) RETURNS SETOF v2_task_status_updates_tmp
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := 'v2_task_status_updates_tmp_' || partition_number::text;
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
CREATE OR REPLACE FUNCTION v2_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'TASK',
        workflow_id
    FROM new_rows
    WHERE dag_id IS NULL;

    INSERT INTO v2_lookup_table (
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
    INSERT INTO v2_dag_to_task_olap (
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

CREATE TRIGGER v2_tasks_olap_status_insert_trigger
AFTER INSERT ON v2_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_tasks_olap_insert_function();

CREATE OR REPLACE FUNCTION v2_tasks_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v2_runs_olap r
    SET
        readable_status = n.readable_status
    FROM new_rows n
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'TASK';

    -- insert tmp events into task status updates table if we have a dag_id
    INSERT INTO v2_task_status_updates_tmp (
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

CREATE TRIGGER v2_tasks_olap_status_update_trigger
AFTER UPDATE ON v2_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_tasks_olap_status_update_function();

CREATE OR REPLACE FUNCTION v2_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_runs_olap (
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        kind,
        workflow_id
    )
    SELECT
        tenant_id,
        id,
        inserted_at,
        external_id,
        readable_status,
        'DAG',
        workflow_id
    FROM new_rows;

    INSERT INTO v2_lookup_table (
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

CREATE TRIGGER v2_dags_olap_status_insert_trigger
AFTER INSERT ON v2_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_dags_olap_insert_function();

CREATE OR REPLACE FUNCTION v2_dags_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v2_runs_olap r
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

CREATE TRIGGER v2_dags_olap_status_update_trigger
AFTER UPDATE ON v2_dags_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_dags_olap_status_update_function();

CREATE OR REPLACE FUNCTION v2_runs_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_statuses_olap (
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

CREATE TRIGGER v2_runs_olap_status_insert_trigger
AFTER INSERT ON v2_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_runs_olap_insert_function();

CREATE OR REPLACE FUNCTION v2_runs_olap_status_update_function()
RETURNS TRIGGER AS
$$
BEGIN
    UPDATE
        v2_statuses_olap s
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

CREATE TRIGGER v2_runs_olap_status_update_trigger
AFTER UPDATE ON v2_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v2_runs_olap_status_update_function();
