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

CREATE TABLE v2_tasks_olap_copy (
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

SELECT * from create_hypertable('v2_tasks_olap_copy', by_range('inserted_at',  INTERVAL '1 day'));

CREATE INDEX v2_tasks_olap_copy_status_workflow_id_idx ON v2_tasks_olap_copy (readable_status, workflow_id);

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

-- Create a copy of v2_task_events_olap to use as a hypertable, because we can't use the trigger on the original
-- table
CREATE TABLE v2_task_events_olap_copy (
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

SELECT * from create_hypertable('v2_task_events_olap_copy', by_range('task_inserted_at',  INTERVAL '1 day'));

SET timescaledb.enable_chunk_skipping = on;

SELECT enable_chunk_skipping('v2_task_events_olap_copy', 'inserted_at');

CREATE INDEX v2_task_events_olap_copy_task_id_idx ON v2_task_events_olap_copy (task_id);

CREATE OR REPLACE FUNCTION v2_tasks_copy()
RETURNS trigger AS $$
DECLARE
  r RECORD; -- Here
BEGIN
  -- This copies from v2_tasks_olap -> v2_tasks_olap_copy, figuring out the correct status and retry
  -- count to set in the process

  -- Get an advisory lock on the relevant tasks
  PERFORM 
    pg_advisory_xact_lock(id) 
  FROM 
    (
        SELECT
          DISTINCT id
        FROM
          new_tasks
        ORDER BY 
          id
    ) AS subquery;
  
  WITH relevant_events AS (
    SELECT
      e.tenant_id,
      e.task_id,
      e.task_inserted_at,
      e.retry_count, 
      e.readable_status
    FROM
      v2_task_events_olap_copy e
    JOIN
      new_tasks nt ON 
        nt.tenant_id = e.tenant_id
        AND nt.id = e.task_id
        AND nt.inserted_at = e.task_inserted_at
  ), max_retry_counts AS (
    SELECT 
      tenant_id,
      task_id,
      task_inserted_at,
      MAX(retry_count) AS max_retry_count
    FROM 
      relevant_events
    GROUP BY 
      tenant_id, task_id, task_inserted_at
  ), statuses AS (
    SELECT
      e.tenant_id,
      e.task_id,
      e.task_inserted_at,
      e.retry_count,
      MAX(e.readable_status) AS max_readable_status
    FROM
      relevant_events e
    JOIN
      max_retry_counts mrc ON 
        e.tenant_id = mrc.tenant_id 
        AND e.task_id = mrc.task_id 
        AND e.task_inserted_at = mrc.task_inserted_at
        AND e.retry_count = mrc.max_retry_count
    GROUP BY
      e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count
  )
  INSERT INTO v2_tasks_olap_copy (
    tenant_id,
    id,
    inserted_at,
    external_id,
    queue,
    action_id,
    step_id,
    workflow_id,
    schedule_timeout,
    step_timeout,
    priority,
    sticky,
    desired_worker_id,
    display_name,
    input,
    additional_metadata,
    readable_status,
    latest_retry_count
  )
  SELECT
    nt.tenant_id,
    nt.id,
    nt.inserted_at,
    nt.external_id,
    nt.queue,
    nt.action_id,
    nt.step_id,
    nt.workflow_id,
    nt.schedule_timeout,
    nt.step_timeout,
    nt.priority,
    nt.sticky,
    nt.desired_worker_id,
    nt.display_name,
    nt.input,
    nt.additional_metadata,
    COALESCE(s.max_readable_status, 'QUEUED') AS readable_status,
    COALESCE(s.retry_count, 0) AS latest_retry_count
  FROM
    new_tasks nt
  LEFT JOIN
    statuses s ON 
      nt.tenant_id = s.tenant_id
      AND nt.id = s.task_id
      AND nt.inserted_at = s.task_inserted_at;

  -- Finally, delete the events from the original table
  DELETE FROM 
    v2_tasks_olap
  WHERE 
    (tenant_id, id, inserted_at) IN (SELECT tenant_id, id, inserted_at FROM new_tasks);

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_insert_tasks
AFTER INSERT ON v2_tasks_olap
REFERENCING NEW TABLE AS new_tasks
FOR EACH STATEMENT
EXECUTE FUNCTION v2_tasks_copy();

CREATE OR REPLACE FUNCTION update_task_readable_status_after_copy()
RETURNS trigger AS $$
DECLARE
  r RECORD; -- Here
BEGIN
  -- Debug: Print rows from locked_tasks to console
  FOR r IN (SELECT task_id, task_inserted_at, retry_count, readable_status, txid_current() as txid FROM new_events) LOOP
    RAISE WARNING 'new_events row: txid=% task_id=%, task_inserted_at=%, retry_count=%, readable_status=%',
      r.txid, r.task_id, r.task_inserted_at, r.retry_count, r.readable_status;
  END LOOP;

  -- Get an advisory lock on the relevant tasks
  PERFORM 
    pg_advisory_xact_lock(task_id) 
  FROM 
    (
        SELECT
          DISTINCT task_id
        FROM
          new_events
        ORDER BY 
          task_id
    ) AS subquery;

  -- Begin with updating the task statuses
  WITH max_retry_counts AS (
    SELECT 
      tenant_id,
      task_id,
      task_inserted_at,
      MAX(retry_count) AS max_retry_count
    FROM 
      new_events
    GROUP BY 
      tenant_id, task_id, task_inserted_at
  ), updatable_events AS (
    SELECT
      e.tenant_id,
      e.task_id,
      e.task_inserted_at,
      e.retry_count,
      MAX(e.readable_status) AS max_readable_status
    FROM
      new_events e
    JOIN
      max_retry_counts mrc ON 
        e.tenant_id = mrc.tenant_id 
        AND e.task_id = mrc.task_id 
        AND e.task_inserted_at = mrc.task_inserted_at
        AND e.retry_count = mrc.max_retry_count
    GROUP BY
      e.tenant_id, e.task_id, e.task_inserted_at, e.retry_count
  )
  UPDATE 
    v2_tasks_olap_copy t
  SET 
    readable_status = e.max_readable_status,
    latest_retry_count = e.retry_count
  FROM 
    updatable_events e
  WHERE 
    (t.tenant_id, t.id, t.inserted_at) = (e.tenant_id, e.task_id, e.task_inserted_at)
    AND e.retry_count >= t.latest_retry_count
    AND e.max_readable_status > t.readable_status;

  -- Next, copy all the events to the hypertable
  INSERT INTO v2_task_events_olap_copy (
      tenant_id, 
      task_id, 
      task_inserted_at, 
      event_type, 
      workflow_id, 
      event_timestamp, 
      readable_status, 
      retry_count, 
      error_message, 
      output, 
      worker_id, 
      additional__event_data, 
      additional__event_message
  )
  SELECT 
      tenant_id, 
      task_id, 
      task_inserted_at, 
      event_type, 
      workflow_id, 
      event_timestamp, 
      readable_status, 
      retry_count, 
      error_message, 
      output, 
      worker_id, 
      additional__event_data, 
      additional__event_message 
  FROM new_events;

  -- Finally, delete the events from the original table
  DELETE FROM v2_task_events_olap WHERE id IN (SELECT id FROM new_events);

  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_task_readable_status_after_copy
AFTER INSERT ON v2_task_events_olap
REFERENCING NEW TABLE AS new_events
FOR EACH STATEMENT
EXECUTE FUNCTION update_task_readable_status_after_copy();

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
      FROM v2_tasks_olap_copy
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
FROM v2_task_events_olap_copy
GROUP BY bucket, tenant_id, workflow_id
ORDER BY bucket
WITH NO DATA;

SELECT add_continuous_aggregate_policy('v2_cagg_task_events_minute',
  start_offset => NULL,
  end_offset => INTERVAL '1 minute',
  schedule_interval => INTERVAL '15 seconds');