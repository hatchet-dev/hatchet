CREATE TYPE v1_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v1_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'CANCELLED',
    'FAILED',
    'COMPLETED',
    'EVICTED'
);

-- NOTE: enum ordering puts EVICTED after COMPLETED, but logically EVICTED is
-- non-terminal and should rank below terminal statuses. These functions provide
-- the canonical priority ordering for aggregation and comparison.
CREATE OR REPLACE FUNCTION v1_status_to_priority(s v1_readable_status_olap)
RETURNS int IMMUTABLE LANGUAGE sql AS $$
    SELECT CASE s
        WHEN 'QUEUED'    THEN 1
        WHEN 'RUNNING'   THEN 2
        WHEN 'EVICTED'   THEN 3
        WHEN 'CANCELLED' THEN 4
        WHEN 'FAILED'    THEN 5
        WHEN 'COMPLETED' THEN 6
    END;
$$;

CREATE OR REPLACE FUNCTION v1_status_from_priority(p int)
RETURNS v1_readable_status_olap IMMUTABLE LANGUAGE sql AS $$
    SELECT CASE p
        WHEN 1 THEN 'QUEUED'
        WHEN 2 THEN 'RUNNING'
        WHEN 3 THEN 'EVICTED'
        WHEN 4 THEN 'CANCELLED'
        WHEN 5 THEN 'FAILED'
        WHEN 6 THEN 'COMPLETED'
    END::v1_readable_status_olap;
$$;

-- Mirrors the body of the convert_duration_to_interval migration so sqlc can
-- resolve the function during codegen. Keep both definitions in sync.
CREATE OR REPLACE FUNCTION convert_duration_to_interval(duration text) RETURNS interval AS $$
DECLARE
    rest text;
    total_seconds double precision := 0;
    m text[];
    val double precision;
    unit text;
    factor double precision;
BEGIN
    IF duration IS NULL OR length(duration) = 0 THEN
        RETURN '5 minutes'::interval;
    END IF;

    m := regexp_match(duration, '^([0-9]{1,8})(d|w|y)$');
    IF m IS NOT NULL THEN
        val := m[1]::double precision;
        unit := m[2];
        CASE unit
            WHEN 'd' THEN RETURN make_interval(days => val::int);
            WHEN 'w' THEN RETURN make_interval(days => (val * 7)::int);
            WHEN 'y' THEN RETURN make_interval(months => (val * 12)::int);
        END CASE;
    END IF;

    rest := duration;

    LOOP
        EXIT WHEN length(rest) = 0;

        m := regexp_match(rest, '^([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(ms|s|m|h)');

        IF m IS NULL THEN
            RETURN '5 minutes'::interval;
        END IF;

        IF length(m[1]) > 15 THEN
            RAISE EXCEPTION 'duration % has a numeric component exceeding 15 digits', duration;
        END IF;

        val := m[1]::double precision;
        unit := m[2];

        CASE unit
            WHEN 'ms' THEN factor := 1e-3;
            WHEN 's' THEN factor := 1;
            WHEN 'm' THEN factor := 60;
            WHEN 'h' THEN factor := 3600;
        END CASE;

        total_seconds := total_seconds + val * factor;

        rest := substring(rest from length(m[1]) + length(m[2]) + 1);
    END LOOP;

    IF total_seconds > 9223372036 THEN
        RAISE EXCEPTION 'duration % exceeds maximum supported value (~292 years)', duration;
    END IF;

    RETURN make_interval(secs => total_seconds);
END;
$$ LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE;

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
        partition_bounds.child_name
    FROM (
        SELECT
            i.inhrelid::regclass::text AS child_name,
            (regexp_match(pg_get_expr(c.relpartbound, c.oid), 'TO \(''([^'']+)''\)'))[1] AS upper_bound
        FROM
            pg_inherits i
        JOIN
            pg_class c ON c.oid = i.inhrelid
        WHERE
            i.inhparent = targetTableName::regclass
            AND substring(i.inhrelid::regclass::text, format('%s_(\d{8})', targetTableName)) ~ '^\d{8}'
    ) partition_bounds
    WHERE
        partition_bounds.upper_bound::timestamptz <= targetDate::timestamptz;
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
    additional_metadata JSONB, -- deprecated - access this through `v1_runs_olap.additional_metadata` instead
    readable_status v1_readable_status_olap NOT NULL DEFAULT 'QUEUED',
    latest_retry_count INT NOT NULL DEFAULT 0,
    latest_worker_id UUID,
    dag_id BIGINT,
    dag_inserted_at TIMESTAMPTZ,
    parent_task_external_id UUID,
    is_durable BOOLEAN NOT NULL DEFAULT FALSE,
    idempotency_key TEXT,
    payload_reference_id UUID,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_tasks_olap_workflow_id_idx ON v1_tasks_olap (tenant_id, workflow_id);
CREATE INDEX v1_tasks_olap_worker_id_idx ON v1_tasks_olap (tenant_id, latest_worker_id) WHERE latest_worker_id IS NOT NULL;
CREATE INDEX ix_v1_tasks_olap_idempotency_key ON v1_tasks_olap (idempotency_key, inserted_at) WHERE idempotency_key IS NOT NULL;

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
    additional_metadata JSONB, -- deprecated - access this through `v1_runs_olap.additional_metadata` instead
    parent_task_external_id UUID,
    total_tasks INT NOT NULL DEFAULT 1,
    idempotency_key TEXT,
    latest_retry_count INT NOT NULL DEFAULT 0,
    is_dag_operator BOOLEAN NOT NULL DEFAULT FALSE,
    payload_reference_id UUID,
    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_dags_olap_workflow_id_idx ON v1_dags_olap (tenant_id, workflow_id);

-- RUN DEFINITIONS --
CREATE TYPE v1_run_kind AS ENUM ('TASK', 'DAG');

-- v1_runs_olap represents an invocation of a workflow. it can either refer to a DAG or a task.
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
    idempotency_key TEXT,
    finished_at TIMESTAMPTZ,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX ix_v1_runs_olap_parent_task_external_id ON v1_runs_olap (parent_task_external_id) WHERE parent_task_external_id IS NOT NULL;
CREATE INDEX ix_v1_runs_olap_tenant_ins_at_status_wf ON v1_runs_olap (tenant_id, inserted_at DESC, readable_status, workflow_id);
CREATE INDEX ix_v1_runs_olap_idempotency_key ON v1_runs_olap (idempotency_key, inserted_at) WHERE idempotency_key IS NOT NULL;

-- Backs additional_metadata containment filters (@> / @> ANY). jsonb_path_ops only
-- supports containment, which is all the list/count queries use.
CREATE INDEX ix_v1_runs_olap_additional_metadata_gin ON v1_runs_olap USING gin (additional_metadata jsonb_path_ops);

CREATE INDEX ix_v1_runs_olap_tenant_failed_finished_at ON v1_runs_olap (tenant_id, finished_at) WHERE readable_status = 'FAILED';

-- LOOKUP TABLES --
CREATE TABLE v1_lookup_table_olap (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL, -- IMPORTANT: Each _partition_ of this table has a `UNIQUE` constraint on this column, but the parent does not
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id, inserted_at)
) PARTITION BY RANGE (inserted_at);

CREATE TABLE v1_dag_to_task_olap (
    dag_id BIGINT NOT NULL,
    dag_inserted_at TIMESTAMPTZ NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (dag_id, dag_inserted_at, task_id, task_inserted_at)
) PARTITION BY RANGE (dag_inserted_at);

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
) PARTITION BY RANGE (inserted_at);


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
    'COULD_NOT_SEND_TO_WORKER',
    'DURABLE_EVICTED',
    'DURABLE_RESTORING',
    'BATCH_BUFFERED',
    'WAITING_FOR_BATCH',
    'BATCH_FLUSHED'
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
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
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
    durable_invocation_count INT NOT NULL DEFAULT 0,
    is_dag_orchestrator BOOLEAN NOT NULL DEFAULT FALSE,
    payload_reference_id UUID,

    PRIMARY KEY (task_id, task_inserted_at, id)
) PARTITION BY RANGE (task_inserted_at);

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
    external_id UUID NOT NULL, -- IMPORTANT: Each _partition_ of this table has a `UNIQUE` constraint on this column, but the parent does not

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

CREATE TABLE v1_payloads_olap_offloaded_block_index (
    payload_inserted_at_date DATE NOT NULL,
    block_external_id_range uuidrange NOT NULL,
    index_file_key TEXT NOT NULL,
    CONSTRAINT v1_payloads_olap_offloaded_block_index_date_range_excl
        EXCLUDE USING GIST (payload_inserted_at_date WITH =, block_external_id_range WITH &&)
);

CREATE UNIQUE INDEX v1_payloads_olap_offloaded_block_index_uq_index_key ON v1_payloads_olap_offloaded_block_index (index_file_key);

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
    payload_reference_id UUID,

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

CREATE TYPE v1_cel_evaluation_failure_source AS ENUM ('FILTER', 'WEBHOOK', 'IDEMPOTENCY_KEY');

CREATE TABLE v1_cel_evaluation_failures_olap (
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    tenant_id UUID NOT NULL,

    source v1_cel_evaluation_failure_source NOT NULL,

    error TEXT NOT NULL,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE TYPE v1_audit_log_actor_type AS ENUM ('user', 'api_key');

CREATE TABLE v1_audit_log (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    actor_type v1_audit_log_actor_type NOT NULL,
    actor_id UUID NOT NULL,
    action TEXT NOT NULL,
    correlation_id TEXT,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,

    PRIMARY KEY (tenant_id, inserted_at, id)
) PARTITION BY RANGE(inserted_at);

CREATE INDEX v1_audit_log_tenant_id_correlation_id_idx ON v1_audit_log (tenant_id, correlation_id DESC);

CREATE TABLE v1_audit_log_archive_cursor (
    partition_date DATE PRIMARY KEY,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    lease_process_id UUID NOT NULL,
    lease_expires_at TIMESTAMPTZ NOT NULL,

    next_file_index INTEGER NOT NULL,
    last_tenant_id UUID NOT NULL,
    last_inserted_at TIMESTAMPTZ NOT NULL,
    last_id UUID NOT NULL
);

-- TRIGGERS TO LINK TASKS, DAGS AND EVENTS --
CREATE OR REPLACE FUNCTION v1_tasks_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

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
        readable_status = n.readable_status,
        -- same status and attempt keeps finished_at, so unrelated updates never mark old rows newly finished
        finished_at = CASE
            WHEN n.readable_status = r.readable_status AND n.latest_retry_count = o.latest_retry_count THEN r.finished_at
            WHEN n.readable_status IN ('COMPLETED', 'FAILED', 'CANCELLED') THEN clock_timestamp()
            ELSE NULL
        END
    FROM new_rows n
    JOIN old_rows o ON o.id = n.id AND o.inserted_at = n.inserted_at
    WHERE
        r.id = n.id
        AND r.inserted_at = n.inserted_at
        AND r.kind = 'TASK';

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_tasks_olap_status_update_trigger
AFTER UPDATE ON v1_tasks_olap
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_tasks_olap_status_update_function();

CREATE OR REPLACE FUNCTION v1_dags_olap_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
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
    ON CONFLICT (external_id, inserted_at) DO NOTHING;

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
        readable_status = n.readable_status,
        parent_task_external_id = n.parent_task_external_id,
        idempotency_key = n.idempotency_key,
        -- same status and attempt keeps finished_at, so unrelated updates never mark old rows newly finished
        finished_at = CASE
            WHEN n.readable_status = r.readable_status AND n.latest_retry_count = o.latest_retry_count THEN r.finished_at
            WHEN n.readable_status IN ('COMPLETED', 'FAILED', 'CANCELLED') THEN clock_timestamp()
            ELSE NULL
        END
    FROM new_rows n
    JOIN old_rows o ON o.id = n.id AND o.inserted_at = n.inserted_at
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
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_dags_olap_status_update_function();

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

CREATE TYPE v1_otel_span_kind AS ENUM ('UNSPECIFIED', 'INTERNAL', 'SERVER', 'CLIENT', 'PRODUCER', 'CONSUMER');

CREATE TYPE v1_otel_status_code AS ENUM ('UNSET', 'OK', 'ERROR');

CREATE TABLE v1_otel_trace_olap (
    tenant_id       UUID NOT NULL,
    trace_id        BYTEA NOT NULL,
    span_id         BYTEA NOT NULL,
    parent_span_id  TEXT,
    span_name       TEXT NOT NULL,
    span_kind       v1_otel_span_kind NOT NULL DEFAULT 'UNSPECIFIED',
    service_name    TEXT NOT NULL DEFAULT 'unknown',
    status_code     v1_otel_status_code NOT NULL DEFAULT 'UNSET',
    status_message  TEXT,
    duration_ns     BIGINT NOT NULL DEFAULT 0,
    resource_attributes JSONB,
    span_attributes     JSONB,
    scope_name      TEXT,
    scope_version   TEXT,
    task_run_external_id    UUID,
    workflow_run_external_id UUID,
    retry_count     INT NOT NULL DEFAULT 0,
    start_time      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, trace_id, start_time, span_id)
) PARTITION BY RANGE (start_time);

CREATE INDEX idx_v1_otel_trace_olap_task_lookup
    ON v1_otel_trace_olap (tenant_id, task_run_external_id)
    WHERE task_run_external_id IS NOT NULL;

CREATE INDEX idx_v1_otel_trace_olap_workflow_lookup
    ON v1_otel_trace_olap (tenant_id, workflow_run_external_id)
    WHERE workflow_run_external_id IS NOT NULL;

CREATE TABLE v1_otel_trace_lookup_olap (
    tenant_id       UUID NOT NULL,
    external_id     UUID NOT NULL,
    retry_count     INT NOT NULL,
    trace_id        BYTEA NOT NULL,
    start_time      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, external_id, retry_count, start_time)
) PARTITION BY RANGE (start_time);

-- OLAP aggregates: per-minute counts of task statuses, task events and events.
--
-- Statement-level triggers on the source tables group each statement's rows by aggregate key
-- and append the counts to the aggregate's delta table, in the writing transaction. The OLAP
-- controller folds delta rows into the aggregate table in the transaction that deletes them.
-- An aggregate's value is therefore its row plus the delta rows for the same key, and a query
-- that reads both in one statement sees exactly the source writes committed in its snapshot.
--
-- Task statuses are counted from v1_runs_olap: one row per task or DAG run, whose status the
-- task and DAG triggers keep current. A status change moves the run between counts, and a
-- deleted run leaves them.
--
-- Writers never update aggregate rows, so hot keys (one tenant's current minute) are not a
-- point of contention on the write path.
--
-- Aggregate tables are partitioned daily like the other OLAP tables, and the OLAP repository
-- creates and drops their partitions with the same retention. Rows written before these triggers
-- existed are not counted.

CREATE TABLE v1_agg_task_statuses_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    evicted_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket, workflow_id)
) PARTITION BY RANGE (bucket);

CREATE TABLE v1_agg_task_events_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count BIGINT NOT NULL DEFAULT 0,
    running_count BIGINT NOT NULL DEFAULT 0,
    completed_count BIGINT NOT NULL DEFAULT 0,
    cancelled_count BIGINT NOT NULL DEFAULT 0,
    failed_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket, workflow_id)
) PARTITION BY RANGE (bucket);

CREATE TABLE v1_agg_events_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    event_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket)
) PARTITION BY RANGE (bucket);

-- Durable stream payload bytes, written by the usage flush rather than by triggers, so it has no
-- delta table: each process upserts its totals once per flush interval.
CREATE TABLE v1_agg_stream_bytes_minute_olap (
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    published_bytes BIGINT NOT NULL DEFAULT 0,
    delivered_bytes BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, bucket)
) PARTITION BY RANGE (bucket);

-- Delta tables: the counts of each source statement, not yet folded into the aggregate. The
-- columns match the aggregate's so the two can be read with UNION ALL. A key can have many
-- delta rows.
--
-- Delta tables are hash partitioned by tenant. Each partition has its own compactor and is
-- vacuumed on its own, and a read for one tenant only touches that tenant's partition.
-- Compactors claim rows in id order, so the oldest deltas of a partition are folded first
-- whichever tenant wrote them. The (tenant_id, bucket) index serves reads, and keeps them cheap
-- if compaction falls far behind. The OLAP repository creates the partitions, with the same
-- autovacuum settings as the other hash partitioned OLAP tables.
CREATE TABLE v1_agg_task_statuses_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count INTEGER NOT NULL,
    running_count INTEGER NOT NULL,
    completed_count INTEGER NOT NULL,
    cancelled_count INTEGER NOT NULL,
    failed_count INTEGER NOT NULL,
    evicted_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE TABLE v1_agg_task_events_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    workflow_id UUID NOT NULL,
    queued_count INTEGER NOT NULL,
    running_count INTEGER NOT NULL,
    completed_count INTEGER NOT NULL,
    cancelled_count INTEGER NOT NULL,
    failed_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE TABLE v1_agg_events_minute_delta_olap (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    bucket TIMESTAMPTZ NOT NULL,
    event_count INTEGER NOT NULL,
    PRIMARY KEY (id, tenant_id)
) PARTITION BY HASH (tenant_id);

CREATE INDEX v1_agg_task_statuses_minute_delta_olap_idx ON v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket);
CREATE INDEX v1_agg_task_events_minute_delta_olap_idx ON v1_agg_task_events_minute_delta_olap (tenant_id, bucket);
CREATE INDEX v1_agg_events_minute_delta_olap_idx ON v1_agg_events_minute_delta_olap (tenant_id, bucket);

-- claim_agg_deltas deletes the oldest batch_size rows of one partition of a delta table, skipping
-- rows another compactor has claimed, and returns them for the caller to fold into the aggregate.
-- The delta table is given by the type of delta_table_row, e.g. NULL::v1_agg_events_minute_delta_olap,
-- which is also the row type it returns.
-- DELETE has no LIMIT, so the rows are claimed in a subquery and deleted by ctid, which looks
-- each one up directly. Delta rows are never updated, so a claimed row keeps its ctid.
CREATE OR REPLACE FUNCTION claim_agg_deltas(
    delta_table_row anyelement,
    partition_number INT,
    batch_size INT
) RETURNS SETOF anyelement
LANGUAGE plpgsql AS
$$
DECLARE
    partition_table text;
BEGIN
    partition_table := pg_typeof(delta_table_row)::text || '_' || partition_number::text;
    RETURN QUERY EXECUTE format(
        'DELETE FROM %I
         WHERE ctid IN (
             SELECT ctid
             FROM %I
             ORDER BY id
             LIMIT $1
             FOR UPDATE SKIP LOCKED
         )
         RETURNING *',
         partition_table, partition_table)
    USING batch_size;
END;
$$;

-- Each trigger writes one delta row per key the statement touched. A statement that changes
-- nothing returns before planning the insert.
CREATE OR REPLACE FUNCTION v1_agg_task_statuses_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT
        tenant_id,
        date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        COUNT(*) FILTER (WHERE readable_status = 'FAILED'),
        COUNT(*) FILTER (WHERE readable_status = 'EVICTED')
    FROM new_rows
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

-- A status change moves the run from the old status's count to the new one's. Updates that
-- leave every key's counts unchanged write nothing.
CREATE OR REPLACE FUNCTION v1_agg_task_statuses_update_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT *
    FROM (
        SELECT
            tenant_id,
            date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS bucket,
            workflow_id,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'QUEUED'), 0) AS queued,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'RUNNING'), 0) AS running,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'COMPLETED'), 0) AS completed,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'CANCELLED'), 0) AS cancelled,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'FAILED'), 0) AS failed,
            COALESCE(SUM(sign) FILTER (WHERE readable_status = 'EVICTED'), 0) AS evicted
        FROM (
            SELECT tenant_id, workflow_id, inserted_at, readable_status, 1 AS sign FROM new_rows
            UNION ALL
            SELECT tenant_id, workflow_id, inserted_at, readable_status, -1 AS sign FROM old_rows
        ) c
        GROUP BY 1, 2, 3
    ) g
    WHERE queued <> 0 OR running <> 0 OR completed <> 0 OR cancelled <> 0 OR failed <> 0 OR evicted <> 0;

    RETURN NULL;
END;
$$;

-- Deleting a run removes it from its status's count.
CREATE OR REPLACE FUNCTION v1_agg_task_statuses_delete_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM old_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_statuses_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count, evicted_count)
    SELECT
        tenant_id,
        date_bin('1 minute', inserted_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        -COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        -COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        -COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        -COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        -COUNT(*) FILTER (WHERE readable_status = 'FAILED'),
        -COUNT(*) FILTER (WHERE readable_status = 'EVICTED')
    FROM old_rows
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

-- A task event is counted in the minute it happened. Event timestamps can come from worker
-- clocks, so they are capped at the inserting transaction's time: a delta never lands in a
-- future minute, which may not have an aggregate partition yet.
CREATE OR REPLACE FUNCTION v1_agg_task_events_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_task_events_minute_delta_olap (tenant_id, bucket, workflow_id, queued_count, running_count, completed_count, cancelled_count, failed_count)
    SELECT
        tenant_id,
        date_bin('1 minute', LEAST(event_timestamp, now()), TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        workflow_id,
        COUNT(*) FILTER (WHERE readable_status = 'QUEUED'),
        COUNT(*) FILTER (WHERE readable_status = 'RUNNING'),
        COUNT(*) FILTER (WHERE readable_status = 'COMPLETED'),
        COUNT(*) FILTER (WHERE readable_status = 'CANCELLED'),
        COUNT(*) FILTER (WHERE readable_status = 'FAILED')
    FROM new_rows
    WHERE NOT is_dag_orchestrator
    GROUP BY 1, 2, 3;

    RETURN NULL;
END;
$$;

CREATE OR REPLACE FUNCTION v1_agg_events_insert_fn() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM new_rows LIMIT 1;
    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    INSERT INTO v1_agg_events_minute_delta_olap (tenant_id, bucket, event_count)
    SELECT
        tenant_id,
        date_bin('1 minute', seen_at, TIMESTAMPTZ '1970-01-01 00:00:00+00'),
        COUNT(*)
    FROM new_rows
    GROUP BY 1, 2;

    RETURN NULL;
END;
$$;

CREATE TRIGGER v1_agg_task_statuses_insert_trigger
AFTER INSERT ON v1_runs_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_insert_fn();

CREATE TRIGGER v1_agg_task_statuses_update_trigger
AFTER UPDATE ON v1_runs_olap
REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_update_fn();

CREATE TRIGGER v1_agg_task_statuses_delete_trigger
AFTER DELETE ON v1_runs_olap
REFERENCING OLD TABLE AS old_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_statuses_delete_fn();

CREATE TRIGGER v1_agg_task_events_insert_trigger
AFTER INSERT ON v1_task_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_task_events_insert_fn();

CREATE TRIGGER v1_agg_events_insert_trigger
AFTER INSERT ON v1_events_olap
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_agg_events_insert_fn();
