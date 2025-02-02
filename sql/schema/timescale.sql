CREATE TYPE v2_sticky_strategy_olap AS ENUM ('NONE', 'SOFT', 'HARD');

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

    PRIMARY KEY (tenant_id, id, inserted_at)
);

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

CREATE TYPE v2_readable_status_olap AS ENUM (
    'QUEUED',
    'RUNNING',
    'COMPLETED',
    'CANCELLED',
    'FAILED'
);

CREATE TABLE v2_task_events_olap (
    tenant_id UUID NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ(3) NOT NULL,
    event_type v2_event_type_olap NOT NULL,
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

CREATE INDEX v2_task_events_olap_task_id_idx ON v2_task_events_olap (task_id);

SELECT * from create_hypertable('v2_task_events_olap', by_range('task_inserted_at',  INTERVAL '1 day'));

SELECT enable_chunk_skipping('v2_task_events_olap', 'inserted_at');

CREATE MATERIALIZED VIEW v2_cagg_task_status
WITH (timescaledb.continuous, timescaledb.materialized_only = true, timescaledb.create_group_indexes = false) AS
SELECT
  tenant_id,
  task_id,
  task_inserted_at,
  time_bucket('5 minutes', task_inserted_at) AS bucket,
  (array_agg(readable_status ORDER BY retry_count DESC, readable_status DESC))[1] AS status,
  max(retry_count) AS max_retry_count
FROM v2_task_events_olap
GROUP BY tenant_id, task_id, task_inserted_at, bucket
ORDER BY bucket DESC, task_inserted_at DESC
WITH NO DATA;

CREATE INDEX v2_cagg_task_status_bucket_tenant_id_status_idx ON v2_cagg_task_status (bucket, tenant_id, status);

SELECT add_continuous_aggregate_policy('v2_cagg_task_status',
  start_offset => NULL,
  end_offset => NULL,
  schedule_interval => INTERVAL '15 seconds');

CREATE  MATERIALIZED VIEW v2_cagg_status_metrics
   WITH (timescaledb.continuous, timescaledb.materialized_only = true)
   AS
      SELECT
        time_bucket('5 minutes', bucket) AS bucket_2,
        tenant_id,
        COUNT(*) FILTER (WHERE status = 'QUEUED') AS queued_count,
        COUNT(*) FILTER (WHERE status = 'RUNNING') AS running_count,
        COUNT(*) FILTER (WHERE status = 'COMPLETED') AS completed_count,
        COUNT(*) FILTER (WHERE status = 'CANCELLED') AS cancelled_count,
        COUNT(*) FILTER (WHERE status = 'FAILED') AS failed_count
      FROM v2_cagg_task_status
      GROUP BY bucket_2, tenant_id
      ORDER BY bucket_2
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_status_metrics',
  start_offset => NULL,
  end_offset => NULL,
  schedule_interval => INTERVAL '15 seconds');

CREATE MATERIALIZED VIEW v2_cagg_task_events_minute
WITH (timescaledb.continuous, timescaledb.materialized_only = true) AS
SELECT
    time_bucket('1 minute', task_inserted_at) AS bucket,
    tenant_id,
    COUNT(*) FILTER (WHERE readable_status = 'QUEUED') AS queued_count,
    COUNT(*) FILTER (WHERE readable_status = 'RUNNING') AS running_count,
    COUNT(*) FILTER (WHERE readable_status = 'COMPLETED') AS completed_count,
    COUNT(*) FILTER (WHERE readable_status = 'CANCELLED') AS cancelled_count,
    COUNT(*) FILTER (WHERE readable_status = 'FAILED') AS failed_count
FROM v2_task_events_olap
GROUP BY bucket, tenant_id
ORDER BY bucket
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_task_events_minute',
  start_offset => NULL,
  end_offset => NULL, 
  schedule_interval => INTERVAL '15 seconds');