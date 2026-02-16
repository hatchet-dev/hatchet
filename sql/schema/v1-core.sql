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

CREATE OR REPLACE FUNCTION get_v1_weekly_partitions_before_date(
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
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < targetDate
        AND (substring(inhrelid::regclass::text, format('%s_(\d{8})', targetTableName))::date) < NOW() - INTERVAL '1 week'
    ;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_range_partition(
    targetTableName text,
    targetDate date,
    fillfactor integer DEFAULT 100
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
        format('CREATE TABLE %s (LIKE %s INCLUDING INDEXES INCLUDING CONSTRAINTS)', newTableName, targetTableName);
    EXECUTE
        format('ALTER TABLE %I SET (
            fillfactor = %s,
            autovacuum_vacuum_scale_factor = ''0.1'',
            autovacuum_analyze_scale_factor=''0.05'',
            autovacuum_vacuum_threshold=''25'',
            autovacuum_analyze_threshold=''25'',
            autovacuum_vacuum_cost_delay=''10'',
            autovacuum_vacuum_cost_limit=''1000''
        )', newTableName, fillfactor);
    EXECUTE
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
    RETURN 1;
END;
$$;

CREATE OR REPLACE FUNCTION create_v1_weekly_range_partition(
    targetTableName text,
    targetDate date
) RETURNS integer
    LANGUAGE plpgsql AS
$$
DECLARE
    targetDateStr varchar;
    targetDatePlusOneWeekStr varchar;
    newTableName varchar;
BEGIN
    SELECT to_char(date_trunc('week', targetDate), 'YYYYMMDD') INTO targetDateStr;
    SELECT to_char(date_trunc('week', targetDate) + INTERVAL '1 week', 'YYYYMMDD') INTO targetDatePlusOneWeekStr;
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
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneWeekStr);
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

CREATE TABLE v1_workflow_concurrency (
    -- We need an id used for stable ordering to prevent deadlocks. We must process all concurrency
    -- strategies on a workflow in the same order.
    id bigint GENERATED ALWAYS AS IDENTITY,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    -- If the strategy is NONE and we've removed all concurrency slots, we can set is_active to false
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    strategy v1_concurrency_strategy NOT NULL,
    child_strategy_ids BIGINT[],
    expression TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    max_concurrency INTEGER NOT NULL,
    CONSTRAINT v1_workflow_concurrency_pkey PRIMARY KEY (workflow_id, workflow_version_id, id)
);

CREATE TABLE v1_step_concurrency (
    -- We need an id used for stable ordering to prevent deadlocks. We must process all concurrency
    -- strategies on a step in the same order.
    id bigint GENERATED ALWAYS AS IDENTITY,
    -- The parent_strategy_id exists if concurrency is defined at the workflow level
    parent_strategy_id BIGINT,
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
DECLARE
  wf_concurrency_row v1_workflow_concurrency%ROWTYPE;
  child_ids bigint[];
BEGIN
  IF NEW."concurrencyGroupExpression" IS NOT NULL THEN
    -- Insert into v1_workflow_concurrency and capture the inserted row.
    INSERT INTO v1_workflow_concurrency (
      workflow_id,
      workflow_version_id,
      strategy,
      expression,
      tenant_id,
      max_concurrency
    )
    SELECT
      wf."id",
      wv."id",
      NEW."limitStrategy"::VARCHAR::v1_concurrency_strategy,
      NEW."concurrencyGroupExpression",
      wf."tenantId",
      NEW."maxRuns"
    FROM "WorkflowVersion" wv
    JOIN "Workflow" wf ON wv."workflowId" = wf."id"
    WHERE wv."id" = NEW."workflowVersionId"
    RETURNING * INTO wf_concurrency_row;

    -- Insert into v1_step_concurrency and capture the inserted rows into a variable.
    WITH inserted_steps AS (
      INSERT INTO v1_step_concurrency (
        parent_strategy_id,
        workflow_id,
        workflow_version_id,
        step_id,
        strategy,
        expression,
        tenant_id,
        max_concurrency
      )
      SELECT
        wf_concurrency_row.id,
        s."workflowId",
        s."workflowVersionId",
        s."id",
        NEW."limitStrategy"::VARCHAR::v1_concurrency_strategy,
        NEW."concurrencyGroupExpression",
        s."tenantId",
        NEW."maxRuns"
      FROM (
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
      ) s
      RETURNING *
    )
    SELECT array_remove(array_agg(t.id), NULL)::bigint[] INTO child_ids
    FROM inserted_steps t;

    -- Update the workflow concurrency row using its primary key.
    UPDATE v1_workflow_concurrency
    SET child_strategy_ids = child_ids
    WHERE workflow_id = wf_concurrency_row.workflow_id
      AND workflow_version_id = wf_concurrency_row.workflow_version_id
      AND id = wf_concurrency_row.id;
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
    workflow_version_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
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
    -- step_index is relevant for tracking down the correct SIGNAL_COMPLETED event on a
    -- replay of a child workflow
    step_index BIGINT NOT NULL,
    additional_metadata JSONB,
    dag_id BIGINT,
    dag_inserted_at TIMESTAMPTZ,
    parent_task_external_id UUID,
    parent_task_id BIGINT,
    parent_task_inserted_at TIMESTAMPTZ,
    child_index BIGINT,
    child_key TEXT,
    initial_state v1_task_initial_state NOT NULL DEFAULT 'QUEUED',
    initial_state_reason TEXT,
    concurrency_parent_strategy_ids BIGINT[],
    concurrency_strategy_ids BIGINT[],
    concurrency_keys TEXT[],
    retry_backoff_factor DOUBLE PRECISION,
    retry_max_backoff INTEGER,
    CONSTRAINT v1_task_pkey PRIMARY KEY (id, inserted_at)
) PARTITION BY RANGE(inserted_at);

CREATE TABLE v1_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    task_id BIGINT,
    dag_id BIGINT,
    inserted_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id)
);

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
    inserted_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    event_type v1_task_event_type NOT NULL,
    -- The event key is an optional field that can be used to uniquely identify an event. This is
    -- used for signal events to ensure that we don't create duplicate signals.
    event_key TEXT,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB,
    external_id UUID,
    CONSTRAINT v1_task_event_pkey PRIMARY KEY (task_id, task_inserted_at, id)
) PARTITION BY RANGE(task_inserted_at);

-- Create unique index on (tenant_id, task_id, event_key) when event_key is not null
CREATE UNIQUE INDEX v1_task_event_event_key_unique_idx ON v1_task_event (
    tenant_id ASC,
    task_id ASC,
    task_inserted_at ASC,
    event_type ASC,
    event_key ASC
) WHERE event_key IS NOT NULL;

-- CreateTable
CREATE TABLE v1_task_expression_eval (
    key TEXT NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    value_str TEXT,
    value_int INTEGER,
    kind "StepExpressionKind" NOT NULL,

    CONSTRAINT v1_task_expression_eval_pkey PRIMARY KEY (task_id, task_inserted_at, kind, key)
);

-- CreateTable
-- NOTE: changes to v1_queue_item should be reflected in v1_rate_limited_queue_items
CREATE TABLE v1_queue_item (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    external_id UUID NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
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
    task_inserted_at ASC,
    retry_count ASC
);

-- CreateTable
CREATE TABLE v1_task_runtime (
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID,
    tenant_id UUID NOT NULL,
    timeout_at TIMESTAMP(3) NOT NULL,
    evicted_at TIMESTAMPTZ DEFAULT NULL,

    CONSTRAINT v1_task_runtime_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

CREATE INDEX v1_task_runtime_tenantId_workerId_idx ON v1_task_runtime (tenant_id ASC, worker_id ASC) WHERE worker_id IS NOT NULL;

CREATE INDEX v1_task_runtime_tenantId_timeoutAt_idx ON v1_task_runtime (tenant_id ASC, timeout_at ASC);

CREATE INDEX v1_task_runtime_tenant_worker_not_evicted_idx ON v1_task_runtime (tenant_id, worker_id) WHERE evicted_at IS NULL;

alter table v1_task_runtime set (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

-- v1_worker_slot_config stores per-worker config for arbitrary slot types.
CREATE TABLE v1_worker_slot_config (
    tenant_id UUID NOT NULL,
    worker_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    max_units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, worker_id, slot_type)
);

-- v1_step_slot_request stores per-step slot requests.
CREATE TABLE v1_step_slot_request (
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, step_id, slot_type)
);

CREATE INDEX v1_step_slot_request_step_idx
    ON v1_step_slot_request (step_id ASC);

-- v1_task_runtime_slot stores runtime slot consumption per task.
CREATE TABLE v1_task_runtime_slot (
    tenant_id UUID NOT NULL,
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, task_inserted_at, retry_count, slot_type)
);

CREATE INDEX v1_task_runtime_slot_tenant_worker_type_idx
    ON v1_task_runtime_slot (tenant_id ASC, worker_id ASC, slot_type ASC);

-- v1_rate_limited_queue_items represents a queue item that has been rate limited and removed from the v1_queue_item table.
CREATE TABLE v1_rate_limited_queue_items (
    requeue_after TIMESTAMPTZ NOT NULL,
    -- everything below this is the same as v1_queue_item
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id bigint NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    external_id UUID NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
    schedule_timeout_at TIMESTAMP(3),
    step_timeout TEXT,
    priority INTEGER NOT NULL DEFAULT 1,
    sticky v1_sticky_strategy NOT NULL,
    desired_worker_id UUID,
    retry_count INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT v1_rate_limited_queue_items_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

CREATE INDEX v1_rate_limited_queue_items_tenant_requeue_after_idx ON v1_rate_limited_queue_items (
    tenant_id ASC,
    queue ASC,
    requeue_after ASC
);

alter table v1_rate_limited_queue_items set (
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
    -- existing_data is data that from previous match conditions that we'd like to propagate when the
    -- new match condition is met. this is used when this is a match created from a previous match, for
    -- example when we've satisfied trigger conditions and would like to register durable sleep, user events
    -- before triggering the DAG.
    existing_data JSONB,
    signal_task_id bigint,
    signal_task_inserted_at timestamptz,
    signal_task_external_id UUID,
    signal_external_id UUID,
    signal_key TEXT,
    -- references the parent DAG for the task, which we can use to get input + additional metadata
    trigger_dag_id bigint,
    trigger_dag_inserted_at timestamptz,
    -- references the step id to instantiate the task
    trigger_step_id UUID,
    trigger_step_index BIGINT,
    -- references the external id for the new task
    trigger_external_id UUID,
    trigger_workflow_run_id UUID,
    trigger_parent_task_external_id UUID,
    trigger_parent_task_id BIGINT,
    trigger_parent_task_inserted_at timestamptz,
    trigger_child_index BIGINT,
    trigger_child_key TEXT,
    -- references the existing task id, which may be set when we're replaying a task
    trigger_existing_task_id bigint,
    trigger_existing_task_inserted_at timestamptz,
    trigger_priority integer,
    durable_event_log_entry_node_id bigint,
    CONSTRAINT v1_match_pkey PRIMARY KEY (id)
);

CREATE TYPE v1_event_type AS ENUM ('USER', 'INTERNAL');

-- Provides information to the caller about the action to take. This is used to differentiate
-- negative conditions from positive conditions. For example, if a task is waiting for a set of
-- tasks to fail, the success of all tasks would be a CANCEL condition, and the failure of any
-- task would be a QUEUE condition. Different actions are implicitly different groups of conditions.
CREATE TYPE v1_match_condition_action AS ENUM ('CREATE', 'QUEUE', 'CANCEL', 'SKIP', 'CREATE_MATCH');

CREATE TABLE v1_match_condition (
    v1_match_id bigint NOT NULL,
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    registered_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    event_type v1_event_type NOT NULL,
    -- for INTERNAL events, this will correspond to a v1_task_event_type value
    event_key TEXT NOT NULL,
    event_resource_hint TEXT,
    -- readable_data_key is used as the key when constructing the aggregated data for the v1_match
    readable_data_key TEXT NOT NULL,
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,
    action v1_match_condition_action NOT NULL DEFAULT 'QUEUE',
    or_group_id UUID NOT NULL,
    expression TEXT,
    data JSONB,
    CONSTRAINT v1_match_condition_pkey PRIMARY KEY (v1_match_id, id)
);

CREATE TABLE v1_filter (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    scope TEXT NOT NULL,
    expression TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    payload_hash TEXT GENERATED ALWAYS AS (MD5(payload::TEXT)) STORED,
    is_declarative BOOLEAN NOT NULL DEFAULT FALSE,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, id)
);

CREATE UNIQUE INDEX v1_filter_unique_tenant_workflow_id_scope_expression_payload ON v1_filter (
    tenant_id ASC,
    workflow_id ASC,
    scope ASC,
    expression ASC,
    payload_hash
);

CREATE TYPE v1_incoming_webhook_auth_type AS ENUM ('BASIC', 'API_KEY', 'HMAC');
CREATE TYPE v1_incoming_webhook_hmac_algorithm AS ENUM ('SHA1', 'SHA256', 'SHA512', 'MD5');
CREATE TYPE v1_incoming_webhook_hmac_encoding AS ENUM ('HEX', 'BASE64', 'BASE64URL');

-- Can add more sources in the future
CREATE TYPE v1_incoming_webhook_source_name AS ENUM ('GENERIC', 'GITHUB', 'STRIPE', 'SLACK', 'LINEAR', 'SVIX');

CREATE TABLE v1_incoming_webhook (
    tenant_id UUID NOT NULL,

    -- names are tenant-unique
    name TEXT NOT NULL,

    source_name v1_incoming_webhook_source_name NOT NULL,

    -- CEL expression that creates an event key
    -- from the payload of the webhook
    event_key_expression TEXT NOT NULL,
    scope_expression TEXT,
    static_payload JSONB,

    auth_method v1_incoming_webhook_auth_type NOT NULL,

    auth__basic__username TEXT,
    auth__basic__password BYTEA,

    auth__api_key__header_name TEXT,
    auth__api_key__key BYTEA,

    auth__hmac__algorithm v1_incoming_webhook_hmac_algorithm,
    auth__hmac__encoding v1_incoming_webhook_hmac_encoding,
    auth__hmac__signature_header_name TEXT,
    auth__hmac__webhook_signing_secret BYTEA,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, name),
    CHECK (
        (
            auth_method = 'BASIC'
            AND (
                auth__basic__username IS NOT NULL
                AND auth__basic__password IS NOT NULL
            )
        )
        OR
        (
            auth_method = 'API_KEY'
            AND (
                auth__api_key__header_name IS NOT NULL
                AND auth__api_key__key IS NOT NULL
            )
        )
        OR
        (
            auth_method = 'HMAC'
            AND (
                auth__hmac__algorithm IS NOT NULL
                AND auth__hmac__encoding IS NOT NULL
                AND auth__hmac__signature_header_name IS NOT NULL
                AND auth__hmac__webhook_signing_secret IS NOT NULL
            )
        )
    ),
    CHECK (LENGTH(event_key_expression) > 0),
    -- Optional: prevent empty string but allow NULL
    CHECK (scope_expression IS NULL OR LENGTH(scope_expression) > 0),
    CHECK (LENGTH(name) > 0)
);


CREATE INDEX v1_match_condition_filter_idx ON v1_match_condition (
    tenant_id ASC,
    event_type ASC,
    event_key ASC,
    is_satisfied ASC,
    event_resource_hint ASC
);

CREATE TABLE v1_dag (
    id bigint GENERATED ALWAYS AS IDENTITY,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    parent_task_external_id UUID,
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
CREATE TABLE v1_workflow_concurrency_slot (
    sort_id BIGINT NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
    strategy_id BIGINT NOT NULL,
    -- DEPRECATED: this is no longer used to track progress of child strategy ids.
    completed_child_strategy_ids BIGINT[],
    child_strategy_ids BIGINT[],
    priority INTEGER NOT NULL DEFAULT 1,
    key TEXT NOT NULL,
    is_filled BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT v1_workflow_concurrency_slot_pkey PRIMARY KEY (strategy_id, workflow_version_id, workflow_run_id)
);

CREATE INDEX v1_workflow_concurrency_slot_query_idx ON v1_workflow_concurrency_slot (tenant_id, strategy_id ASC, key ASC, priority DESC, sort_id ASC);

-- CreateTable
CREATE TABLE v1_concurrency_slot (
    sort_id BIGINT GENERATED ALWAYS AS IDENTITY,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    task_retry_count INTEGER NOT NULL,
    external_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    workflow_version_id UUID NOT NULL,
    workflow_run_id UUID NOT NULL,
    strategy_id BIGINT NOT NULL,
    parent_strategy_id BIGINT,
    priority INTEGER NOT NULL DEFAULT 1,
    key TEXT NOT NULL,
    is_filled BOOLEAN NOT NULL DEFAULT FALSE,
    next_parent_strategy_ids BIGINT[],
    next_strategy_ids BIGINT[],
    next_keys TEXT[],
    queue_to_notify TEXT NOT NULL,
    schedule_timeout_at TIMESTAMP(3) NOT NULL,
    CONSTRAINT v1_concurrency_slot_pkey PRIMARY KEY (task_id, task_inserted_at, task_retry_count, strategy_id)
);

CREATE INDEX v1_concurrency_slot_query_idx ON v1_concurrency_slot (tenant_id, strategy_id ASC, key ASC, sort_id ASC);

-- When concurrency slot is CREATED, we should check whether the parent concurrency slot exists; if not, we should create
-- the parent concurrency slot as well.
CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_insert_function()
RETURNS trigger AS $$
BEGIN
    WITH parent_slot AS (
        SELECT
            *
        FROM
            new_table cs
        WHERE
            cs.parent_strategy_id IS NOT NULL
    ), parent_to_child_strategy_ids AS (
        SELECT
            wc.id AS parent_strategy_id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id,
            MAX(ps.sort_id) AS sort_id,
            MAX(ps.priority) AS priority,
            MAX(ps.key) AS key,
            ARRAY_AGG(DISTINCT wc.child_strategy_ids) AS child_strategy_ids
        FROM
            parent_slot ps
        JOIN v1_workflow_concurrency wc ON wc.workflow_id = ps.workflow_id AND wc.workflow_version_id = ps.workflow_version_id AND wc.id = ps.parent_strategy_id
        GROUP BY
            wc.id,
            wc.tenant_id,
            ps.workflow_id,
            ps.workflow_version_id,
            ps.workflow_run_id
    )
    INSERT INTO v1_workflow_concurrency_slot (
        sort_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        strategy_id,
        child_strategy_ids,
        priority,
        key
    )
    SELECT
        pcs.sort_id,
        pcs.tenant_id,
        pcs.workflow_id,
        pcs.workflow_version_id,
        pcs.workflow_run_id,
        pcs.parent_strategy_id,
        pcs.child_strategy_ids,
        pcs.priority,
        pcs.key
    FROM
        parent_to_child_strategy_ids pcs
    ON CONFLICT (strategy_id, workflow_version_id, workflow_run_id) DO NOTHING;

    -- If the v1_step_concurrency strategy is not active, we set it to active.
    WITH inactive_strategies AS (
        SELECT
            strategy.*
        FROM
            new_table cs
        JOIN
            v1_step_concurrency strategy ON strategy.workflow_id = cs.workflow_id AND strategy.workflow_version_id = cs.workflow_version_id AND strategy.id = cs.strategy_id
        WHERE
            strategy.is_active = FALSE
        ORDER BY
            strategy.id
        FOR UPDATE
    )
    UPDATE v1_step_concurrency strategy
    SET is_active = TRUE
    FROM inactive_strategies
    WHERE
        strategy.workflow_id = inactive_strategies.workflow_id AND
        strategy.workflow_version_id = inactive_strategies.workflow_version_id AND
        strategy.step_id = inactive_strategies.step_id AND
        strategy.id = inactive_strategies.id;

    RETURN NULL;
END;

$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_insert
AFTER INSERT ON v1_concurrency_slot
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_insert_function();

CREATE OR REPLACE FUNCTION cleanup_workflow_concurrency_slots(
    p_strategy_id BIGINT,
    p_workflow_version_id UUID,
    p_workflow_run_id UUID
) RETURNS VOID AS $$
DECLARE
    v_sort_id BIGINT;
BEGIN
    -- Get the sort_id for the specific workflow concurrency slot
    SELECT sort_id INTO v_sort_id
    FROM v1_workflow_concurrency_slot
    WHERE strategy_id = p_strategy_id
      AND workflow_version_id = p_workflow_version_id
      AND workflow_run_id = p_workflow_run_id;

    -- Acquire an advisory lock for the strategy ID
    -- There is a small chance of collisions but it's extremely unlikely
    PERFORM pg_advisory_xact_lock(1000000 * p_strategy_id + v_sort_id);

    WITH relevant_tasks_for_dags AS (
        SELECT
            t.id,
            t.inserted_at,
            t.retry_count
        FROM
            v1_task t
        JOIN
            v1_dag_to_task dt ON t.id = dt.task_id AND t.inserted_at = dt.task_inserted_at
        JOIN
            v1_lookup_table lt ON dt.dag_id = lt.dag_id AND dt.dag_inserted_at = lt.inserted_at
        WHERE
            lt.external_id = p_workflow_run_id
            AND lt.dag_id IS NOT NULL
    ), final_concurrency_slots_for_dags AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
            AND NOT EXISTS (
                -- Check if any task in this DAG has a v1_concurrency_slot
                SELECT 1
                FROM relevant_tasks_for_dags rt
                WHERE EXISTS (
                    SELECT 1
                    FROM v1_concurrency_slot cs2
                    WHERE cs2.task_id = rt.id
                        AND cs2.task_inserted_at = rt.inserted_at
                        AND cs2.task_retry_count = rt.retry_count
                )
            )
            AND CARDINALITY(wcs.child_strategy_ids) <= (
                SELECT COUNT(*)
                FROM relevant_tasks_for_dags rt
            )
        GROUP BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
    ), final_concurrency_slots_for_tasks AS (
        -- If the workflow run id corresponds to a single task, we can safely delete the workflow concurrency slot
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            v1_lookup_table lt ON wcs.workflow_run_id = lt.external_id AND lt.task_id IS NOT NULL
        WHERE
            wcs.strategy_id = p_strategy_id
            AND wcs.workflow_version_id = p_workflow_version_id
            AND wcs.workflow_run_id = p_workflow_run_id
    ), all_parent_slots_to_delete AS (
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_dags
        UNION ALL
        SELECT
            strategy_id,
            workflow_version_id,
            workflow_run_id
        FROM
            final_concurrency_slots_for_tasks
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            all_parent_slots_to_delete ps ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (ps.strategy_id, ps.workflow_version_id, ps.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    DELETE FROM
        v1_workflow_concurrency_slot wcs
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                strategy_id,
                workflow_version_id,
                workflow_run_id
            FROM
                locked_parent_slots
        );

END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
DECLARE
    rec RECORD;
BEGIN
    FOR rec IN SELECT * FROM deleted_rows ORDER BY parent_strategy_id, workflow_version_id, workflow_run_id LOOP
        IF rec.parent_strategy_id IS NOT NULL THEN
            PERFORM cleanup_workflow_concurrency_slots(
                rec.parent_strategy_id,
                rec.workflow_version_id,
                rec.workflow_run_id
            );
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_delete
AFTER DELETE ON v1_concurrency_slot
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_delete_function();

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
RETURNS TRIGGER AS $$
DECLARE
    rec RECORD;
BEGIN
    -- Only insert if there's a single task with initial_state = 'QUEUED' and concurrency_strategy_ids is not null
    IF (SELECT COUNT(*) FROM new_table WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL) > 0 THEN
        WITH new_slot_rows AS (
            SELECT
                id,
                inserted_at,
                retry_count,
                tenant_id,
                priority,
                concurrency_parent_strategy_ids[1] AS parent_strategy_id,
                CASE
                    WHEN array_length(concurrency_parent_strategy_ids, 1) > 1 THEN concurrency_parent_strategy_ids[2:array_length(concurrency_parent_strategy_ids, 1)]
                    ELSE '{}'::bigint[]
                END AS next_parent_strategy_ids,
                concurrency_strategy_ids[1] AS strategy_id,
                external_id,
                workflow_run_id,
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
                workflow_version_id,
                queue,
                CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout) AS schedule_timeout_at
            FROM new_table
            WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL
        )
        INSERT INTO v1_concurrency_slot (
            task_id,
            task_inserted_at,
            task_retry_count,
            external_id,
            tenant_id,
            workflow_id,
            workflow_version_id,
            workflow_run_id,
            parent_strategy_id,
            next_parent_strategy_ids,
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
            external_id,
            tenant_id,
            workflow_id,
            workflow_version_id,
            workflow_run_id,
            parent_strategy_id,
            next_parent_strategy_ids,
            strategy_id,
            next_strategy_ids,
            COALESCE(priority, 1),
            key,
            next_keys,
            queue,
            schedule_timeout_at
        FROM new_slot_rows;
    END IF;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
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
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        COALESCE(priority, 1),
        sticky,
        desired_worker_id,
        retry_count
    FROM new_table
    WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NULL;

    -- Only insert into v1_dag and v1_dag_to_task if dag_id and dag_inserted_at are not null
    IF (SELECT COUNT(*) FROM new_table WHERE dag_id IS NOT NULL AND dag_inserted_at IS NOT NULL) > 0 THEN
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
    END IF;

    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        task_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

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
    WITH new_retry_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            -- Convert the retry_after based on min(retry_backoff_factor ^ retry_count, retry_max_backoff)
            NOW() + (LEAST(nt.retry_max_backoff, POWER(nt.retry_backoff_factor, nt.app_retry_count)) * interval '1 second') AS retry_after
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            AND nt.retry_backoff_factor IS NOT NULL
            AND ot.app_retry_count IS DISTINCT FROM nt.app_retry_count
            AND nt.app_retry_count != 0
    )
    INSERT INTO v1_retry_queue_item (
        task_id,
        task_inserted_at,
        task_retry_count,
        retry_after,
        tenant_id
    )
    SELECT
        id,
        inserted_at,
        retry_count,
        retry_after,
        tenant_id
    FROM new_retry_rows;

    WITH new_slot_rows AS (
        SELECT
            nt.id,
            nt.inserted_at,
            nt.retry_count,
            nt.tenant_id,
            nt.workflow_run_id,
            nt.external_id,
            nt.concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(nt.concurrency_parent_strategy_ids, 1) > 1 THEN nt.concurrency_parent_strategy_ids[2:array_length(nt.concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
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
            nt.workflow_version_id,
            nt.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(nt.schedule_timeout) AS schedule_timeout_at
        FROM new_table nt
        JOIN old_table ot ON ot.id = nt.id
        WHERE nt.initial_state = 'QUEUED'
            -- Concurrency strategy id should never be null
            AND nt.concurrency_strategy_ids[1] IS NOT NULL
            AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
            AND ot.retry_count IS DISTINCT FROM nt.retry_count
    ), updated_slot AS (
        UPDATE
            v1_concurrency_slot cs
        SET
            task_retry_count = nt.retry_count,
            schedule_timeout_at = nt.schedule_timeout_at,
            is_filled = FALSE,
            priority = 4
        FROM
            new_slot_rows nt
        WHERE
            cs.task_id = nt.id
            AND cs.task_inserted_at = nt.inserted_at
            AND cs.strategy_id = nt.strategy_id
        RETURNING cs.*
    ), slots_to_insert AS (
        -- select the rows that were not updated
        SELECT
            nt.*
        FROM
            new_slot_rows nt
        LEFT JOIN
            updated_slot cs ON cs.task_id = nt.id AND cs.task_inserted_at = nt.inserted_at AND cs.strategy_id = nt.strategy_id
        WHERE
            cs.task_id IS NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
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
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        4,
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM slots_to_insert;

    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
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
        nt.inserted_at,
        nt.external_id,
        nt.action_id,
        nt.step_id,
        nt.workflow_id,
        nt.workflow_run_id,
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
        AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
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

CREATE OR REPLACE FUNCTION v1_retry_queue_item_delete_function()
RETURNS TRIGGER AS
$$
BEGIN
    WITH new_slot_rows AS (
        SELECT
            t.id,
            t.inserted_at,
            t.retry_count,
            t.tenant_id,
            t.workflow_run_id,
            t.external_id,
            t.concurrency_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(t.concurrency_parent_strategy_ids, 1) > 1 THEN t.concurrency_parent_strategy_ids[2:array_length(t.concurrency_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
            t.concurrency_strategy_ids[1] AS strategy_id,
            CASE
                WHEN array_length(t.concurrency_strategy_ids, 1) > 1 THEN t.concurrency_strategy_ids[2:array_length(t.concurrency_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_strategy_ids,
            t.concurrency_keys[1] AS key,
            CASE
                WHEN array_length(t.concurrency_keys, 1) > 1 THEN t.concurrency_keys[2:array_length(t.concurrency_keys, 1)]
                ELSE '{}'::text[]
            END AS next_keys,
            t.workflow_id,
            t.workflow_version_id,
            t.queue,
            CURRENT_TIMESTAMP + convert_duration_to_interval(t.schedule_timeout) AS schedule_timeout_at
        FROM deleted_rows dr
        JOIN
            v1_task t ON t.id = dr.task_id AND t.inserted_at = dr.task_inserted_at
        WHERE
            dr.retry_after <= NOW()
            AND t.initial_state = 'QUEUED'
            -- Check to see if the task has a concurrency strategy
            AND t.concurrency_strategy_ids[1] IS NOT NULL
    )
    INSERT INTO v1_concurrency_slot (
        task_id,
        task_inserted_at,
        task_retry_count,
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
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
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
        strategy_id,
        next_strategy_ids,
        4,
        key,
        next_keys,
        queue,
        schedule_timeout_at
    FROM new_slot_rows;

    WITH tasks AS (
        SELECT
            t.*
        FROM
            deleted_rows dr
        JOIN v1_task t ON t.id = dr.task_id AND t.inserted_at = dr.task_inserted_at
        WHERE
            dr.retry_after <= NOW()
            AND t.initial_state = 'QUEUED'
            AND t.concurrency_strategy_ids[1] IS NULL
    )
    INSERT INTO v1_queue_item (
        tenant_id,
        queue,
        task_id,
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
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
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(schedule_timeout),
        step_timeout,
        4,
        sticky,
        desired_worker_id,
        retry_count
    FROM tasks;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_retry_queue_item_delete_trigger
AFTER DELETE ON v1_retry_queue_item
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_retry_queue_item_delete_function();

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
            t.workflow_run_id,
            t.external_id,
            nt.next_parent_strategy_ids[1] AS parent_strategy_id,
            CASE
                WHEN array_length(nt.next_parent_strategy_ids, 1) > 1 THEN nt.next_parent_strategy_ids[2:array_length(nt.next_parent_strategy_ids, 1)]
                ELSE '{}'::bigint[]
            END AS next_parent_strategy_ids,
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
            t.workflow_version_id,
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
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
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
        external_id,
        tenant_id,
        workflow_id,
        workflow_version_id,
        workflow_run_id,
        parent_strategy_id,
        next_parent_strategy_ids,
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
        task_inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
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
        inserted_at,
        external_id,
        action_id,
        step_id,
        workflow_id,
        workflow_run_id,
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

CREATE OR REPLACE FUNCTION v1_dag_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_lookup_table (
        external_id,
        tenant_id,
        dag_id,
        inserted_at
    )
    SELECT
        external_id,
        tenant_id,
        id,
        inserted_at
    FROM new_table
    ON CONFLICT (external_id) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_dag_insert_trigger
AFTER INSERT ON v1_dag
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE PROCEDURE v1_dag_insert_function();

CREATE TYPE v1_log_line_level AS ENUM ('DEBUG', 'INFO', 'WARN', 'ERROR');

CREATE TABLE v1_log_line (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tenant_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    message TEXT NOT NULL,
    level v1_log_line_level NOT NULL DEFAULT 'INFO',
    metadata JSONB,
    retry_count INTEGER NOT NULL DEFAULT 0,

    PRIMARY KEY (task_id, task_inserted_at, id)
) PARTITION BY RANGE(task_inserted_at);

CREATE TYPE v1_step_match_condition_kind AS ENUM ('PARENT_OVERRIDE', 'USER_EVENT', 'SLEEP');

CREATE TABLE v1_step_match_condition (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    readable_data_key TEXT NOT NULL,
    action v1_match_condition_action NOT NULL DEFAULT 'CREATE',
    or_group_id UUID NOT NULL,
    expression TEXT,
    kind v1_step_match_condition_kind NOT NULL,
    -- If this is a SLEEP condition, this will be set to the sleep duration
    sleep_duration TEXT,
    -- If this is a USER_EVENT condition, this will be set to the user event key
    event_key TEXT,
    -- If this is a PARENT_OVERRIDE condition, this will be set to the parent readable_id
    parent_readable_id TEXT,
    PRIMARY KEY (step_id, id)
);

CREATE TABLE v1_durable_sleep (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    sleep_until TIMESTAMPTZ NOT NULL,
    sleep_duration TEXT NOT NULL,
    PRIMARY KEY (tenant_id, sleep_until, id)
);

CREATE TYPE v1_payload_type AS ENUM ('TASK_INPUT', 'DAG_INPUT', 'TASK_OUTPUT', 'TASK_EVENT_DATA', 'USER_EVENT_INPUT', 'DURABLE_EVENT_LOG_ENTRY_DATA', 'DURABLE_EVENT_LOG_ENTRY_RESULT_DATA');

-- IMPORTANT: Keep these values in sync with `v1_payload_type_olap` in the OLAP db
CREATE TYPE v1_payload_location AS ENUM ('INLINE', 'EXTERNAL');

CREATE TABLE v1_payload (
    tenant_id UUID NOT NULL,
    id BIGINT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL,
    external_id UUID,
    type v1_payload_type NOT NULL,
    location v1_payload_location NOT NULL,
    external_location_key TEXT,
    inline_content JSONB,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, inserted_at, id, type),
    CHECK (
        location = 'INLINE'
        OR
        (location = 'EXTERNAL' AND inline_content IS NULL AND external_location_key IS NOT NULL)
    )
) PARTITION BY RANGE(inserted_at);

CREATE TABLE v1_payload_cutover_job_offset (
    key DATE PRIMARY KEY,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    lease_process_id UUID NOT NULL DEFAULT gen_random_uuid(),
    lease_expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    last_tenant_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'::UUID,
    last_inserted_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
    last_id BIGINT NOT NULL DEFAULT 0,
    last_type v1_payload_type NOT NULL DEFAULT 'TASK_INPUT'
);

CREATE OR REPLACE FUNCTION copy_v1_payload_partition_structure(
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO target_table_name;
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
                INSERT INTO %I (tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content, updated_at)
                VALUES (NEW.tenant_id, NEW.id, NEW.inserted_at, NEW.external_id, NEW.type, NEW.location, NEW.external_location_key, NEW.inline_content, NEW.updated_at)
                ON CONFLICT (tenant_id, id, inserted_at, type) DO UPDATE
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
                    AND id = NEW.id
                    AND inserted_at = NEW.inserted_at
                    AND type = NEW.type;
                RETURN NEW;
            ELSIF TG_OP = ''DELETE'' THEN
                DELETE FROM %I
                WHERE
                    tenant_id = OLD.tenant_id
                    AND id = OLD.id
                    AND inserted_at = OLD.inserted_at
                    AND type = OLD.type;
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

CREATE OR REPLACE FUNCTION list_paginated_payloads_for_offload(
    partition_date date,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type,
    next_tenant_id uuid,
    next_inserted_at timestamptz,
    next_id bigint,
    next_type v1_payload_type,
    batch_size integer
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
    external_location_key TEXT,
    inline_content JSONB,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS MATERIALIZED (
            SELECT tenant_id, id, inserted_at, external_id, type, location,
                external_location_key, inline_content, updated_at
            FROM %I
            WHERE
                (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            ORDER BY tenant_id, inserted_at, id, type

            -- Multiplying by two here to handle an edge case. There is a small chance we miss a row
            -- when a different row is inserted before it, in between us creating the chunks and selecting
            -- them. By multiplying by two to create a "candidate" set, we significantly reduce the chance of us missing
            -- rows in this way, since if a row is inserted before one of our last rows, we will still have
            -- the next row after it in the candidate set.
            LIMIT $9 * 2
        )

        SELECT tenant_id, id, inserted_at, external_id, type, location,
               external_location_key, inline_content, updated_at
        FROM candidates
        WHERE
            (tenant_id, inserted_at, id, type) >= ($1, $2, $3, $4)
            AND (tenant_id, inserted_at, id, type) <= ($5, $6, $7, $8)
        ORDER BY tenant_id, inserted_at, id, type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, next_tenant_id, next_inserted_at, next_id, next_type, batch_size;
END;
$$;

CREATE OR REPLACE FUNCTION compute_payload_batch_size(
    partition_date DATE,
    last_tenant_id UUID,
    last_inserted_at TIMESTAMPTZ,
    last_id BIGINT,
    last_type v1_payload_type,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH candidates AS (
            SELECT *
            FROM %I
            WHERE (tenant_id, inserted_at, id, type) >= ($1::UUID, $2::TIMESTAMPTZ, $3::BIGINT, $4::v1_payload_type)
            ORDER BY tenant_id, inserted_at, id, type
            LIMIT $5::INTEGER
        )

        SELECT COALESCE(SUM(pg_column_size(inline_content)), 0) AS total_size_bytes
        FROM candidates
    ', source_partition_name);

    EXECUTE query INTO result_size USING last_tenant_id, last_inserted_at, last_id, last_type, batch_size;

    RETURN result_size;
END;
$$;

CREATE OR REPLACE FUNCTION create_payload_offload_range_chunks(
    partition_date date,
    window_size int,
    chunk_size int,
    last_tenant_id uuid,
    last_inserted_at timestamptz,
    last_id bigint,
    last_type v1_payload_type
) RETURNS TABLE (
    lower_tenant_id UUID,
    lower_id BIGINT,
    lower_inserted_at TIMESTAMPTZ,
    lower_type v1_payload_type,
    upper_tenant_id UUID,
    upper_id BIGINT,
    upper_inserted_at TIMESTAMPTZ,
    upper_type v1_payload_type
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        WITH paginated AS (
            SELECT tenant_id, id, inserted_at, type, ROW_NUMBER() OVER (ORDER BY tenant_id, inserted_at, id, type) AS rn
            FROM %I
            WHERE (tenant_id, inserted_at, id, type) > ($1, $2, $3, $4)
            ORDER BY tenant_id, inserted_at, id, type
            LIMIT $5::INTEGER
        ), lower_bounds AS (
            SELECT rn::INTEGER / $6::INTEGER AS batch_ix, tenant_id::UUID, id::BIGINT, inserted_at::TIMESTAMPTZ, type::v1_payload_type
            FROM paginated
            WHERE MOD(rn, $6::INTEGER) = 1
        ), upper_bounds AS (
            SELECT
                -- Using `CEIL` and subtracting 1 here to make the `batch_ix` zero indexed like the `lower_bounds` one is.
                -- We need the `CEIL` to handle the case where the number of rows in the window is not evenly divisible by the batch size,
                -- because without CEIL if e.g. there were 5 rows in the window and a batch size of two and we did integer division, we would end
                -- up with batches of index 0, 1, and 1 after dividing and subtracting. With float division and `CEIL`, we get 0, 1, and 2 as expected.
                -- Then we need to subtract one because we compute the batch index by using integer division on the lower bounds, which are all zero indexed.
                CEIL(rn::FLOAT / $6::FLOAT) - 1 AS batch_ix,
                tenant_id::UUID,
                id::BIGINT,
                inserted_at::TIMESTAMPTZ,
                type::v1_payload_type
            FROM paginated
            -- We want to include either the last row of each batch, or the last row of the entire paginated set, which may not line up with a batch end.
            WHERE MOD(rn, $6::INTEGER) = 0 OR rn = (SELECT MAX(rn) FROM paginated)
        )

        SELECT
            lb.tenant_id AS lower_tenant_id,
            lb.id AS lower_id,
            lb.inserted_at AS lower_inserted_at,
            lb.type AS lower_type,
            ub.tenant_id AS upper_tenant_id,
            ub.id AS upper_id,
            ub.inserted_at AS upper_inserted_at,
            ub.type AS upper_type
        FROM lower_bounds lb
        JOIN upper_bounds ub ON lb.batch_ix = ub.batch_ix
        ORDER BY lb.tenant_id, lb.inserted_at, lb.id, lb.type
    ', source_partition_name);

    RETURN QUERY EXECUTE query USING last_tenant_id, last_inserted_at, last_id, last_type, window_size, chunk_size;
END;
$$;

CREATE OR REPLACE FUNCTION diff_payload_source_and_target_partitions(
    partition_date date
) RETURNS TABLE (
    tenant_id UUID,
    id BIGINT,
    inserted_at TIMESTAMPTZ,
    external_id UUID,
    type v1_payload_type,
    location v1_payload_location,
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO temp_partition_name;

    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE EXCEPTION 'Partition % does not exist', source_partition_name;
    END IF;

    query := format('
        SELECT tenant_id, id, inserted_at, external_id, type, location, external_location_key, inline_content, updated_at
        FROM %I source
        WHERE NOT EXISTS (
            SELECT 1
            FROM %I AS target
            WHERE
                source.tenant_id = target.tenant_id
                AND source.inserted_at = target.inserted_at
                AND source.id = target.id
                AND source.type = target.type
        )
    ', source_partition_name, temp_partition_name);

    RETURN QUERY EXECUTE query;
END;
$$;

CREATE OR REPLACE FUNCTION swap_v1_payload_partition_with_temp(
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
    SELECT format('v1_payload_%s', partition_date_str) INTO source_partition_name;
    SELECT format('v1_payload_offload_tmp_%s', partition_date_str) INTO temp_table_name;
    SELECT format('v1_payload_offload_tmp_%s_pkey', partition_date_str) INTO old_pk_name;
    SELECT format('v1_payload_%s_pkey', partition_date_str) INTO new_pk_name;
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

    LOCK TABLE v1_payload IN ACCESS EXCLUSIVE MODE;

    RAISE NOTICE 'Dropping trigger from partition %', source_partition_name;
    EXECUTE format('DROP TRIGGER IF EXISTS %I ON %I', trigger_name, source_partition_name);

    RAISE NOTICE 'Dropping trigger function %', trigger_function_name;
    EXECUTE format('DROP FUNCTION IF EXISTS %I()', trigger_function_name);

    IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = source_partition_name) THEN
        RAISE NOTICE 'Dropping old partition %', source_partition_name;
        EXECUTE format('ALTER TABLE v1_payload DETACH PARTITION %I', source_partition_name);
        EXECUTE format('DROP TABLE %I CASCADE', source_partition_name);
    END IF;

    RAISE NOTICE 'Renaming primary key % to %', old_pk_name, new_pk_name;
    EXECUTE format('ALTER INDEX %I RENAME TO %I', old_pk_name, new_pk_name);

    RAISE NOTICE 'Renaming temp table % to %', temp_table_name, source_partition_name;
    EXECUTE format('ALTER TABLE %I RENAME TO %I', temp_table_name, source_partition_name);

    RAISE NOTICE 'Attaching new partition % to v1_payload', source_partition_name;
    EXECUTE format(
        'ALTER TABLE v1_payload ATTACH PARTITION %I FOR VALUES FROM (%L) TO (%L)',
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

CREATE TABLE v1_idempotency_key (
    tenant_id UUID NOT NULL,

    key TEXT NOT NULL,

    expires_at TIMESTAMPTZ NOT NULL,
    claimed_by_external_id UUID,

    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (tenant_id, expires_at, key)
);

CREATE UNIQUE INDEX v1_idempotency_key_unique_tenant_key ON v1_idempotency_key (tenant_id, key);

-- v1_operation_interval_settings represents the interval settings for a specific tenant. "Operation" means
-- any sort of tenant-specific polling-based operation on the engine, like timeouts, reassigns, etc.
CREATE TABLE v1_operation_interval_settings (
    tenant_id UUID NOT NULL,
    operation_id TEXT NOT NULL,
    -- The interval represents a Go time.Duration, hence the nanoseconds
    interval_nanoseconds BIGINT NOT NULL,
    PRIMARY KEY (tenant_id, operation_id)
);

-- Events tables
CREATE TABLE v1_event (
    id bigint GENERATED ALWAYS AS IDENTITY,
    seen_at TIMESTAMPTZ NOT NULL,
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL DEFAULT gen_random_uuid(),
    key TEXT NOT NULL,
    additional_metadata JSONB,
    scope TEXT,
    triggering_webhook_name TEXT,

    PRIMARY KEY (tenant_id, seen_at, id)
) PARTITION BY RANGE(seen_at);

CREATE INDEX v1_event_key_idx ON v1_event (tenant_id, key);

CREATE TABLE v1_event_lookup_table (
    tenant_id UUID NOT NULL,
    external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,

    PRIMARY KEY (external_id, event_seen_at)
) PARTITION BY RANGE(event_seen_at);

CREATE OR REPLACE FUNCTION v1_event_lookup_table_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v1_event_lookup_table (
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
    ON CONFLICT (external_id, event_seen_at) DO NOTHING;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v1_event_lookup_table_insert_trigger
AFTER INSERT ON v1_event
REFERENCING NEW TABLE AS new_rows
FOR EACH STATEMENT
EXECUTE FUNCTION v1_event_lookup_table_insert_function();

CREATE TABLE v1_event_to_run (
    run_external_id UUID NOT NULL,
    event_id BIGINT NOT NULL,
    event_seen_at TIMESTAMPTZ NOT NULL,
    filter_id UUID,

    PRIMARY KEY (event_id, event_seen_at, run_external_id)
) PARTITION BY RANGE(event_seen_at);

-- v1_durable_event_log represents the log file for the durable event history
-- of a durable task. This table stores metadata like sequence values for entries.
--
-- Important: writers to v1_durable_event_log_entry should lock this row to increment the sequence value.
CREATE TABLE v1_durable_event_log_file (
    tenant_id UUID NOT NULL,

    -- The id and inserted_at of the durable task which created this entry
    durable_task_id BIGINT NOT NULL,
    durable_task_inserted_at TIMESTAMPTZ NOT NULL,

    latest_invocation_count BIGINT NOT NULL,

    latest_inserted_at TIMESTAMPTZ NOT NULL,
    -- A monotonically increasing node id for this durable event log scoped to the durable task.
    -- Starts at 0 and increments by 1 for each new entry.
    latest_node_id BIGINT NOT NULL,
    -- The latest branch id. Branches represent different execution paths on a replay.
    latest_branch_id BIGINT NOT NULL,
    -- The parent node id which should be linked to the first node in a new branch to its parent node.
    latest_branch_first_parent_node_id BIGINT NOT NULL,
    CONSTRAINT v1_durable_event_log_file_pkey PRIMARY KEY (durable_task_id, durable_task_inserted_at)
) PARTITION BY RANGE(durable_task_inserted_at);

CREATE TYPE v1_durable_event_log_kind AS ENUM (
    'RUN',
    'WAIT_FOR',
    'MEMO'
);

CREATE TABLE v1_durable_event_log_entry (
    tenant_id UUID NOT NULL,

    -- need an external id for consistency with the payload store logic (unfortunately)
    external_id UUID NOT NULL,
    -- The id and inserted_at of the durable task which created this entry
    -- The inserted_at time of this event from a DB clock perspective.
    -- Important: for consistency, this should always be auto-generated via the CURRENT_TIMESTAMP!
    inserted_at TIMESTAMPTZ NOT NULL,
    id BIGINT NOT NULL GENERATED ALWAYS AS IDENTITY,

    durable_task_id BIGINT NOT NULL,
    durable_task_inserted_at TIMESTAMPTZ NOT NULL,

    kind v1_durable_event_log_kind NOT NULL,
    -- The node number in the durable event log. This represents a monotonically increasing
    -- sequence value generated from v1_durable_event_log_file.latest_node_id
    node_id BIGINT NOT NULL,
    -- The parent node id for this event, if any. This can be null.
    parent_node_id BIGINT,
    -- The branch id when this event was first seen. A durable event log can be a part of many branches.
    branch_id BIGINT NOT NULL,
    -- Todo: Associated data for this event should be stored in the v1_payload table!
    -- data JSONB,
    -- The hash of the data stored in the v1_payload table to check non-determinism violations.
    -- This can be null for event types that don't have associated data.
    -- TODO: we can add CHECK CONSTRAINT for event types that require data_hash to be non-null.
    data_hash BYTEA,
    -- Can discuss: adds some flexibility for future hash algorithms
    data_hash_alg TEXT,
    -- Access patterns:
    -- Definite: we'll query directly for the node_id when a durable task is replaying its log
    -- Possible: we may want to query a range of node_ids for a durable task
    -- Possible: we may want to query a range of inserted_ats for a durable task

    -- Whether this callback has been seen by the engine or not. Note that is_satisfied _may_ change multiple
    -- times through the lifecycle of a callback, and readers should not assume that once it's true it will always be true.
    is_satisfied BOOLEAN NOT NULL DEFAULT FALSE,

    CONSTRAINT v1_durable_event_log_entry_pkey PRIMARY KEY (durable_task_id, durable_task_inserted_at, node_id)
) PARTITION BY RANGE(durable_task_inserted_at);
