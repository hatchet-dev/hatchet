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
CREATE INDEX ix_v1_runs_olap_tenant_id ON v1_runs_olap (tenant_id, inserted_at, id, readable_status, kind);

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
    'SKIPPED',
    'COULD_NOT_SEND_TO_WORKER'
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
    external_id UUID,
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

CREATE TABLE v1_incoming_webhook_validation_failures_olap (
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    tenant_id UUID NOT NULL,

    -- webhook names are tenant-unique
    incoming_webhook_name TEXT NOT NULL,

    error TEXT NOT NULL,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_incoming_webhook_validation_failures_olap_tenant_id_incoming_webhook_name_idx ON v1_incoming_webhook_validation_failures_olap (tenant_id, incoming_webhook_name);

-- IMPORTANT: Keep these values in sync with `v1_payload_type` in the core db
CREATE TYPE v1_payload_location_olap AS ENUM ('INLINE', 'EXTERNAL');

CREATE TABLE v1_payloads_olap (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,

    location v1_payload_location_olap NOT NULL,
    external_location_key TEXT,
    inline_content JSONB,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, external_id, inserted_at),
    CHECK (
        location = 'INLINE'
        OR
        (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
    )
) PARTITION BY RANGE(inserted_at);

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

CREATE OR REPLACE FUNCTION find_matching_tenants_in_task_status_updates_tmp_partition(
    partition_number INT,
    tenant_ids UUID[]
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_task_status_updates_tmp_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE e.tenant_id = ANY($1)
              AND e.requeue_after <= CURRENT_TIMESTAMP
        )',
        partition_table)
    USING tenant_ids
    INTO result;

    RETURN result;
END;
$$;

CREATE OR REPLACE FUNCTION find_matching_tenants_in_task_events_tmp_partition(
    partition_number INT,
    tenant_ids UUID[]
) RETURNS UUID[]
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
    result UUID[];
BEGIN
    partition_table := 'v1_task_events_olap_tmp_' || partition_number::text;

    EXECUTE format(
        'SELECT ARRAY(
            SELECT DISTINCT e.tenant_id
            FROM %I e
            WHERE e.tenant_id = ANY($1)
              AND e.requeue_after <= CURRENT_TIMESTAMP
        )',
        partition_table)
    USING tenant_ids
    INTO result;

    RETURN result;
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
    scope TEXT,
    triggering_webhook_name TEXT,

    PRIMARY KEY (tenant_id, seen_at, id)
) PARTITION BY RANGE(seen_at);

CREATE INDEX v1_events_olap_key_idx ON v1_events_olap (tenant_id, key);
CREATE INDEX v1_events_olap_scope_idx ON v1_events_olap (tenant_id, scope) WHERE scope IS NOT NULL;

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
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,
    filter_id UUID,

    PRIMARY KEY (event_id, event_seen_at, run_id, run_inserted_at)
) PARTITION BY RANGE(event_seen_at);

CREATE TYPE v1_cel_evaluation_failure_source AS ENUM ('FILTER', 'WEBHOOK');

CREATE TABLE v1_cel_evaluation_failures_olap (
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    tenant_id UUID NOT NULL,

    source v1_cel_evaluation_failure_source NOT NULL,

    error TEXT NOT NULL,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

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
    WHERE dag_id IS NULL
    ON CONFLICT (inserted_at, id, readable_status, kind) DO NOTHING;

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
    WHERE dag_id IS NOT NULL
    ON CONFLICT (dag_id, dag_inserted_at, task_id, task_inserted_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_insert_trigger
AFTER INSERT ON v1_tasks_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_insert_function();

CREATE OR REPLACE FUNCTION v1_tasks_olap_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    DELETE FROM v1_runs_olap r
    USING old_rows o
    WHERE
        r.inserted_at = o.inserted_at
        AND r.id = o.id
        AND r.readable_status = o.readable_status
        AND r.kind = 'TASK'
        AND o.dag_id IS NULL;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_delete_trigger
AFTER DELETE ON v1_tasks_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_delete_function();

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
    FROM new_rows
    ON CONFLICT (inserted_at, id, readable_status, kind) DO NOTHING;

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

CREATE OR REPLACE FUNCTION v1_dags_olap_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    DELETE FROM v1_runs_olap r
    USING old_rows o
    WHERE
        r.inserted_at = o.inserted_at
        AND r.id = o.id
        AND r.readable_status = o.readable_status
        AND r.kind = 'DAG';

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_dags_olap_status_delete_trigger
AFTER DELETE ON v1_dags_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_delete_function();

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
    FROM new_rows
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

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
        seen_at
    FROM new_rows
    ON CONFLICT (tenant_id, external_id, event_seen_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_olap_insert_trigger
AFTER INSERT ON v1_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_events_lookup_table_olap_insert_function();

CREATE TABLE v1_payloads_olap_cutover_job_offset (
    key DATE PRIMARY KEY,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    lease_process_id UUID NOT NULL DEFAULT gen_random_uuid(),
    lease_expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
    last_external_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
    last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00'
);

CREATE OR REPLACE FUNCTION copy_v1_payloads_olap_partition_structure(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    target_table_name varchar;
    trigger_function_name varchar;
    trigger_name varchar;
    partition_start date;
    partition_end date;
BEGIN
    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO target_table_name;
    SELECT format('sync_to_%s', target_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', target_table_name) INTO trigger_name;
    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Source partition % does not exist', source_partition_name;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = target_table_name) THEN
        RAISE NOTICE 'Target table % already exists, skipping creation', target_table_name;
        RETURN target_table_name;
    END IF;

    EXECUTE format(
        'CREATE TABLE %I (LIKE %I INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING INDEXES)',
        target_table_name,
        source_partition_name
    );

    EXECUTE format('
        ALTER TABLE %I
        ADD CONSTRAINT %I
        CHECK (
            inserted_at IS NOT NULL
            AND inserted_at >= %L::TIMESTAMPTZ
            AND inserted_at < %L::TIMESTAMPTZ
        )
        ',
        target_table_name,
        target_table_name || '_iat_chk_bounds',
        partition_start,
        partition_end
    );

    EXECUTE format('
        CREATE OR REPLACE FUNCTION %I() RETURNS trigger
            LANGUAGE plpgsql AS $func$
        BEGIN
            IF TG_OP = ''INSERT'' THEN
                INSERT INTO %I (tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at)
                VALUES (NEW.tenant_id, NEW.external_id, NEW.location, NEW.external_location_key, NEW.inline_content, NEW.inserted_at, NEW.updated_at)
                ON CONFLICT (tenant_id, external_id, inserted_at) DO UPDATE
                SET
                    location = EXCLUDED.location,
                    external_location_key = EXCLUDED.external_location_key,
                    inline_content = EXCLUDED.inline_content,
                    updated_at = EXCLUDED.updated_at;
                RETURN NEW;
            ELSIF TG_OP = ''UPDATE'' THEN
                UPDATE %I
                SET
                    location = NEW.location,
                    external_location_key = NEW.external_location_key,
                    inline_content = NEW.inline_content,
                    updated_at = NEW.updated_at
                WHERE
                    tenant_id = NEW.tenant_id
                    AND external_id = NEW.external_id
                    AND inserted_at = NEW.inserted_at
                    ;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                DELETE FROM %I
                WHERE
                    tenant_id = OLD.tenant_id
                    AND external_id = OLD.external_id
                    AND inserted_at = OLD.inserted_at
                    ;
                RETURN OLD;
            END IF;
            RETURN NULL;
        END;
        $func$;
    ', trigger_function_name, target_table_name, target_table_name, target_table_name);

    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    EXECUTE format('
        CREATE TRIGGER %I
        AFTER INSERT OR UPDATE OR DELETE ON %I
        FOR EACH ROW
        EXECUTE FUNCTION %I();
    ', trigger_name, source_partition_name, trigger_function_name);

    RAISE NOTICE 'Created table % as a copy of partition % with sync trigger', target_table_name, source_partition_name;

    RETURN target_table_name;
END;
$$;

CREATE OR REPLACE FUNCTION list_paginated_olap_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz,
    next_tenant_id uuid,
    next_external_id uuid,
    next_inserted_at timestamptz,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    inserted_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS MATERIALIZED (
            SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
            FROM %I
            WHERE
                (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
            ORDER BY tenant_id, external_id, inserted_at

            -- Multiplying by two here to handle an edge case. There is a small chance we miss a row
            -- when a different row is inserted before it, in between us creating the chunks and selecting
            -- them. By multiplying by two to create a "candidate" set, we significantly reduce the chance of us missing
            -- rows in this way, since if a row is inserted before one of our last rows, we will still have
            -- the next row after it in the candidate set.
            LIMIT $7 * 2
        )

        SELECT tenant_id, external_id, location, external_location_key, inline_content, inserted_at, updated_at
        FROM candidates
        WHERE
            (tenant_id, external_id, inserted_at) >= ($1, $2, $3)
            AND (tenant_id, external_id, inserted_at) <= ($4, $5, $6)
        ORDER BY tenant_id, external_id, inserted_at
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, next_tenant_id, next_external_id, next_inserted_at, batch_size;
END;
$$;

CREATE OR REPLACE FUNCTION create_olap_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_external_id uuid,
    last_inserted_at timestamptz
) RETURNS TABLE (
    lower_tenant_id UUID,
    lower_external_id UUID,
    lower_inserted_at TIMESTAMPTZ,
    upper_tenant_id UUID,
    upper_external_id UUID,
    upper_inserted_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH paginated AS (
            SELECT tenant_id, external_id, inserted_at, ROW_NUMBER() OVER (ORDER BY tenant_id, external_id, inserted_at) AS rn
            FROM %I
            WHERE (tenant_id, external_id, inserted_at) > ($1, $2, $3)
            ORDER BY tenant_id, external_id, inserted_at
            LIMIT $4
        ), lower_bounds AS (
            SELECT rn::INTEGER / $5::INTEGER AS batch_ix, tenant_id::UUID, external_id::UUID, inserted_at::TIMESTAMPTZ
            FROM paginated
            WHERE MOD(rn, $5::INTEGER) = 1
        ), upper_bounds AS (
            SELECT
                -- Using `CEIL` and subtracting 1 here to make the `batch_ix` zero indexed like the `lower_bounds` one is.
                -- We need the `CEIL` to handle the case where the number of rows in the window is not evenly divisible by the batch size,
                -- because without CEIL if e.g. there were 5 rows in the window and a batch size of two and we did integer division, we would end
                -- up with batches of index 0, 1, and 1 after dividing and subtracting. With float division and `CEIL`, we get 0, 1, and 2 as expected.
                -- Then we need to subtract one because we compute the batch index by using integer division on the lower bounds, which are all zero indexed.
                CEIL(rn::FLOAT / $5::FLOAT) - 1 AS batch_ix,
                tenant_id::UUID,
                external_id::UUID,
                inserted_at::TIMESTAMPTZ
            FROM paginated
            -- We want to include either the last row of each batch, or the last row of the entire paginated set, which may not line up with a batch end.
            WHERE MOD(rn, $5::INTEGER) = 0 OR rn = (SELECT MAX(rn) FROM paginated)
        )

        SELECT
            lb.tenant_id AS lower_tenant_id,
            lb.external_id AS lower_external_id,
            lb.inserted_at AS lower_inserted_at,
            ub.tenant_id AS upper_tenant_id,
            ub.external_id AS upper_external_id,
            ub.inserted_at AS upper_inserted_at
        FROM lower_bounds lb
        JOIN upper_bounds ub ON lb.batch_ix = ub.batch_ix
        ORDER BY lb.tenant_id, lb.external_id, lb.inserted_at
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_external_id, last_inserted_at, window_size, chunk_size;
END;
$$;

CREATE OR REPLACE FUNCTION diff_olap_payload_source_and_target_partitions(
    partition_date date
) RETURNS TABLE (
    tenant_id UUID,
    external_id UUID,
    inserted_at TIMESTAMPTZ,
    location v1_payload_location_olap,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ
)
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_partition_name varchar;
    query text;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO temp_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, external_id, inserted_at, location, external_location_key, inline_content, updated_at
        FROM %I source
        WHERE NOT EXISTS (
            SELECT 1
            FROM %I AS target
            WHERE
                source.tenant_id = target.tenant_id
                AND source.external_id = target.external_id
                AND source.inserted_at = target.inserted_at
        )
    ', source_partition_name, temp_partition_name);

    RETURN QUERY EXECUTE query;
END;
$$;

CREATE OR REPLACE FUNCTION compute_olap_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_external_id UUID,
    last_inserted_at TIMESTAMPTZ,
    batch_size INTEGER
) RETURNS BIGINT
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str TEXT;
    source_partition_name TEXT;
    query TEXT;
    result_size BIGINT;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS (
            SELECT *
            FROM %I
            WHERE (tenant_id, external_id, inserted_at) >= ($1::UUID, $2::UUID, $3::TIMESTAMPTZ)
            ORDER BY tenant_id, external_id, inserted_at
            LIMIT $4::INT
        )

        SELECT COALESCE(SUM(pg_column_size(inline_content)), 0) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size USING last_tenant_id, last_external_id, last_inserted_at, batch_size;

    RETURN result_size;
END;
$$;

CREATE OR REPLACE FUNCTION swap_v1_payloads_olap_partition_with_temp(
    partition_date date
) RETURNS text
    LANGUAGE plpgsql AS
$$
DECLARE
    partition_date_str varchar;
    source_partition_name varchar;
    temp_table_name varchar;
    old_pk_name varchar;
    new_pk_name varchar;
    partition_start date;
    partition_end date;
    trigger_function_name varchar;
    trigger_name varchar;
BEGIN
    IF partition_date IS NULL THEN
        RAISE EXCEPTION 'partition_date parameter cannot be NULL';
    END IF;

    SELECT to_char(partition_date, 'YYYYMMDD') INTO partition_date_str;
    SELECT format('v1_payloads_olap_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s', partition_date_str) INTO temp_table_name;
    SELECT format('v1_payloads_olap_offload_tmp_%s_pkey', partition_date_str) INTO old_pk_name;
    SELECT format('v1_payloads_olap_%s_pkey', partition_date_str) INTO new_pk_name;
    SELECT format('sync_to_%s', temp_table_name) INTO trigger_function_name;
    SELECT format('trigger_sync_to_%s', temp_table_name) INTO trigger_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = temp_table_name) THEN
        RAISE EXCEPTION 'Temp table % does not exist', temp_table_name;
    END IF;

    partition_start := partition_date;
    partition_end := partition_date + INTERVAL '1 day';

    EXECUTE format(
        'ALTER TABLE %I SET (
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor = ''0.05'',
            autovacuum_vacuum_threshold = ''25'',
            autovacuum_analyze_threshold = ''25'',
            autovacuum_vacuum_cost_delay = ''10'',
            autovacuum_vacuum_cost_limit = ''1000''
        )',
        temp_table_name
    );
    RAISE NOTICE 'Set autovacuum settings on partition %', temp_table_name;

    LOCK TABLE v1_payloads_olap IN ACCESS EXCLUSIVE MODE;

    RAISE NOTICE 'Dropping trigger from partition %', source_partition_name;
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    RAISE NOTICE 'Dropping trigger function %', trigger_function_name;
    EXECUTE format('DROP FUNCTION IF EXISTS %I()', trigger_function_name);

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE NOTICE 'Dropping old partition %', source_partition_name;
        EXECUTE format('ALTER TABLE v1_payloads_olap DETACH PARTITION %I', source_partition_name);
        EXECUTE format('DROP TABLE %I CASCADE', source_partition_name);
    END IF;

    RAISE NOTICE 'Renaming primary key % to %', old_pk_name, new_pk_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_pk_name, new_pk_name);

    RAISE NOTICE 'Renaming temp table % to %', temp_table_name, source_partition_name;
    EXECUTE format('ALTER TABLE %I RENAME TO %I', temp_table_name, source_partition_name);

    RAISE NOTICE 'Attaching new partition % to v1_payloads_olap', source_partition_name;
    EXECUTE format(
        'ALTER TABLE v1_payloads_olap ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
        source_partition_name,
        partition_start,
        partition_end
    );

    RAISE NOTICE 'Dropping hack check constraint';
    EXECUTE format(
        'ALTER TABLE %I DROP CONSTRAINT %I',
        source_partition_name,
        temp_table_name || '_iat_chk_bounds'
    );

    RAISE NOTICE 'Successfully swapped partition %', source_partition_name;
    RETURN source_partition_name;
END;
$$;
