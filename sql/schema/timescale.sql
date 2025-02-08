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

    PRIMARY KEY (tenant_id, id, inserted_at)
);

SELECT * from create_hypertable('v2_tasks_olap', by_range('inserted_at',  INTERVAL '1 day'));

CREATE INDEX v2_tasks_olap_status_workflow_id_idx ON v2_tasks_olap (readable_status, workflow_id);

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
    id bigint GENERATED ALWAYS AS IDENTITY,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ(3) NOT NULL,
    event_type v2_event_type_olap NOT NULL,
    readable_status v2_readable_status_olap NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,

    PRIMARY KEY (tenant_id, task_id, task_inserted_at, id)
);

alter table v2_task_events_olap_tmp set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

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
