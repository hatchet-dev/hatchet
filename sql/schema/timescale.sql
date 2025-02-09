CREATE TYPE v2_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

CREATE TYPE v2_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'COMPLETED',
    'CANCELLED',
    'FAILED'
);

CREATE TABLE v2_tasks_olap (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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

    PRIMARY KEY (tenant_id, id, inserted_at)
);

SELECT * from create_hypertable('v2_tasks_olap', by_range('inserted_at',  INTERVAL '1 day'));

CREATE INDEX v2_tasks_olap_status_workflow_id_idx ON v2_tasks_olap (tenant_id, inserted_at DESC, readable_status, workflow_id);

CREATE INDEX v2_tasks_olap_worker_id_idx ON v2_tasks_olap (tenant_id, inserted_at DESC, latest_worker_id) WHERE latest_worker_id IS NOT NULL;

CREATE TABLE v2_task_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ(3) NOT NULL,

    PRIMARY KEY (external_id)
);

CREATE OR REPLACE FUNCTION v2_tasks_olap_lookup_table_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_task_lookup_table (
        tenant_id,
        external_id,
        task_id,
        inserted_at
    )
    VALUES (
        NEW.tenant_id,
        NEW.external_id,
        NEW.id,
        NEW.inserted_at
    );
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v2_tasks_olap_lookup_table_insert_trigger
AFTER INSERT ON v2_tasks_olap
FOR EACH ROW
EXECUTE PROCEDURE v2_tasks_olap_lookup_table_insert_function();

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
    'RATE_LIMIT_ERROR'
);

CREATE TABLE v2_task_events_olap_tmp (
    tenant_id UUID NOT NULL,
    requeue_after TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    requeue_retries INT NOT NULL DEFAULT 0,
    id bigint GENERATED ALWAYS AS IDENTITY,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ(3) NOT NULL,
    event_type v2_event_type_olap NOT NULL,
    readable_status v2_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    worker_id UUID,

    PRIMARY KEY (tenant_id, requeue_after, task_id, id)
) PARTITION BY HASH(task_id);

CREATE OR REPLACE FUNCTION create_v2_task_events_partitions(
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
    WHERE inhparent = 'v2_task_events_olap_tmp'::regclass;

    IF existing_count > num_partitions THEN
        RAISE EXCEPTION 'Cannot decrease the number of partitions: we already have % partitions which is more than the target %', existing_count, num_partitions;
    END IF;

    FOR i IN 0..(num_partitions - 1) LOOP
        partition_name := format('v2_task_events_olap_tmp_%s', i);
        IF to_regclass(partition_name) IS NULL THEN
            EXECUTE format('CREATE TABLE %I (LIKE v2_task_events_olap_tmp INCLUDING INDEXES)', partition_name);
            EXECUTE format('ALTER TABLE %I SET (
                autovacuum_vacuum_scale_factor = ''0.1'',
                autovacuum_analyze_scale_factor = ''0.05'',
                autovacuum_vacuum_threshold = ''25'',
                autovacuum_analyze_threshold = ''25'',
                autovacuum_vacuum_cost_delay = ''10'',
                autovacuum_vacuum_cost_limit = ''1000''
            )', partition_name);
            EXECUTE format('ALTER TABLE v2_task_events_olap_tmp ATTACH PARTITION %I FOR VALUES WITH (modulus %s, remainder %s)', partition_name, num_partitions, i);
            created_count := created_count + 1;
        END IF;
    END LOOP;
    RETURN created_count;
END;
$$;

CREATE OR REPLACE FUNCTION list_task_events(
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
    inserted_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ(3) NOT NULL,
    event_type v2_event_type_olap NOT NULL,
    workflow_id UUID NOT NULL,
    event_timestamp TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    readable_status v2_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    error_message TEXT,
    output JSONB,
    worker_id UUID,
    additional__event_data TEXT,
    additional__event_message TEXT,

    PRIMARY KEY (tenant_id, task_id, task_inserted_at, id)
);

SELECT * from create_hypertable('v2_task_events_olap', by_range('task_inserted_at',  INTERVAL '1 day'));

SET timescaledb.enable_chunk_skipping = on;

SELECT enable_chunk_skipping('v2_task_events_olap', 'inserted_at');

CREATE INDEX v2_task_events_olap_task_id_idx ON v2_task_events_olap (task_id);

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
      FROM v2_tasks_olap
      GROUP BY tenant_id, workflow_id, bucket
      ORDER BY bucket DESC
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_status_metrics',
  start_offset => NULL,
  end_offset => INTERVAL '5 minutes',
  schedule_interval => INTERVAL '15 seconds');

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
