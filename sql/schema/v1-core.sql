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

CREATE OR REPLACE FUNCTION create_v1_range_partition(
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
        format('ALTER TABLE %s ATTACH PARTITION %s FOR VALUES FROM (''%s'') TO (''%s'')', targetTableName, newTableName, targetDateStr, targetDatePlusOneDayStr);
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

    CONSTRAINT v1_task_runtime_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

CREATE INDEX v1_task_runtime_tenantId_workerId_idx ON v1_task_runtime (tenant_id ASC, worker_id ASC) WHERE worker_id IS NOT NULL;

CREATE INDEX v1_task_runtime_tenantId_timeoutAt_idx ON v1_task_runtime (tenant_id ASC, timeout_at ASC);

alter table v1_task_runtime set (
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
    signal_task_id bigint,
    signal_task_inserted_at timestamptz,
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
    CONSTRAINT v1_match_pkey PRIMARY KEY (id)
);

CREATE TYPE v1_event_type AS ENUM ('USER', 'INTERNAL');

-- Provides information to the caller about the action to take. This is used to differentiate
-- negative conditions from positive conditions. For example, if a task is waiting for a set of
-- tasks to fail, the success of all tasks would be a CANCEL condition, and the failure of any
-- task would be a QUEUE condition. Different actions are implicitly different groups of conditions.
CREATE TYPE v1_match_condition_action AS ENUM ('CREATE', 'QUEUE', 'CANCEL', 'SKIP');

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
    ON CONFLICT (strategy_id, workflow_version_id, workflow_run_id) DO UPDATE
        -- If there's a conflict, and we're inserting a new concurrency_slot, we'd like to remove the strategy_id
        -- from the completed child strategy ids.
        SET completed_child_strategy_ids = ARRAY(
            SELECT DISTINCT UNNEST(ARRAY_REMOVE(v1_workflow_concurrency_slot.completed_child_strategy_ids, cs.strategy_id))
            FROM new_table cs
            WHERE EXCLUDED.strategy_id = cs.parent_strategy_id
        );

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

CREATE OR REPLACE FUNCTION after_v1_concurrency_slot_delete_function()
RETURNS trigger AS $$
BEGIN
    -- When v1_concurrency_slot is DELETED, we add it to the completed_child_strategy_ids on the parent, but only
    -- when it's NOT a backoff retry. Backoff retries will continue to consume a workflow concurrency slot.
    WITH parent_slot AS (
        SELECT
            cs.workflow_id,
            cs.workflow_version_id,
            cs.workflow_run_id,
            cs.strategy_id,
            cs.parent_strategy_id
        FROM
            deleted_rows cs
        JOIN
            v1_task t ON t.id = cs.task_id AND t.inserted_at = cs.task_inserted_at
        LEFT JOIN
            v1_retry_queue_item rqi ON rqi.task_id = t.id AND rqi.task_inserted_at = t.inserted_at
        WHERE
            cs.parent_strategy_id IS NOT NULL
            AND rqi.task_id IS NULL
    ), locked_parent_slots AS (
        SELECT
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id,
            cs.strategy_id AS child_strategy_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            parent_slot cs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
        ORDER BY
            wcs.strategy_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id
        FOR UPDATE
    )
    UPDATE v1_workflow_concurrency_slot wcs
    SET completed_child_strategy_ids = ARRAY(
        SELECT DISTINCT UNNEST(ARRAY_APPEND(wcs.completed_child_strategy_ids, cs.child_strategy_id))
    )
    FROM locked_parent_slots cs
    WHERE
        wcs.strategy_id = cs.strategy_id
        AND wcs.workflow_version_id = cs.workflow_version_id
        AND wcs.workflow_run_id = cs.workflow_run_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_concurrency_slot_delete
AFTER DELETE ON v1_concurrency_slot
REFERENCING OLD TABLE AS deleted_rows
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_concurrency_slot_delete_function();

-- After we update the v1_workflow_concurrency_slot, we'd like to check whether all child_strategy_ids are
-- in the completed_child_strategy_ids. If so, we should delete the v1_workflow_concurrency_slot.
CREATE OR REPLACE FUNCTION after_v1_workflow_concurrency_slot_update_function()
RETURNS trigger AS $$
BEGIN
    -- place a lock on new_table
    WITH slots_to_delete AS (
        SELECT
            wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
        FROM
            new_table wcs
        WHERE
            CARDINALITY(wcs.child_strategy_ids) = CARDINALITY(wcs.completed_child_strategy_ids)
        ORDER BY
            wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
        FOR UPDATE
    )
    DELETE FROM
        v1_workflow_concurrency_slot wcs
    WHERE
        (strategy_id, workflow_version_id, workflow_run_id) IN (
            SELECT
                strategy_id, workflow_version_id, workflow_run_id
            FROM
                slots_to_delete
        );

    RETURN NULL;
END;

$$ LANGUAGE plpgsql;

CREATE TRIGGER after_v1_workflow_concurrency_slot_update
AFTER UPDATE ON v1_workflow_concurrency_slot
REFERENCING NEW TABLE AS new_table
FOR EACH STATEMENT
EXECUTE FUNCTION after_v1_workflow_concurrency_slot_update_function();

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
    FOR rec IN SELECT * FROM new_table WHERE initial_state = 'QUEUED' AND concurrency_strategy_ids[1] IS NOT NULL AND concurrency_keys[1] IS NULL LOOP
        RAISE WARNING 'New table row: %', row_to_json(rec);
    END LOOP;

    -- When a task is inserted in a non-queued state, we should add all relevant completed_child_strategy_ids to the parent
    -- concurrency slots.
    WITH parent_slots AS (
        SELECT
            nt.workflow_id,
            nt.workflow_version_id,
            nt.workflow_run_id,
            UNNEST(nt.concurrency_strategy_ids) AS strategy_id,
            UNNEST(nt.concurrency_parent_strategy_ids) AS parent_strategy_id
        FROM
            new_table nt
        WHERE
            cardinality(nt.concurrency_parent_strategy_ids) > 0
            AND nt.initial_state != 'QUEUED'
    ), locked_parent_slots AS (
        SELECT
            wcs.workflow_id,
            wcs.workflow_version_id,
            wcs.workflow_run_id,
            wcs.strategy_id,
            cs.strategy_id AS child_strategy_id
        FROM
            v1_workflow_concurrency_slot wcs
        JOIN
            parent_slots cs ON (wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id) = (cs.parent_strategy_id, cs.workflow_version_id, cs.workflow_run_id)
        ORDER BY
            wcs.strategy_id, wcs.workflow_version_id, wcs.workflow_run_id
        FOR UPDATE
    )
    UPDATE
        v1_workflow_concurrency_slot wcs
    SET
        -- get unique completed_child_strategy_ids after append with cs.strategy_id
        completed_child_strategy_ids = ARRAY(
            SELECT
                DISTINCT UNNEST(ARRAY_APPEND(wcs.completed_child_strategy_ids, cs.child_strategy_id))
        )
    FROM
        locked_parent_slots cs
    WHERE
        wcs.strategy_id = cs.strategy_id
        AND wcs.workflow_version_id = cs.workflow_version_id
        AND wcs.workflow_run_id = cs.workflow_run_id;

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
            AND nt.concurrency_strategy_ids[1] IS NOT NULL
            AND (nt.retry_backoff_factor IS NULL OR ot.app_retry_count IS NOT DISTINCT FROM nt.app_retry_count OR nt.app_retry_count = 0)
            AND ot.retry_count IS DISTINCT FROM nt.retry_count
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
    -- If this is a PARENT_OVERRIDE condition, this will be set to the parent readable_id
    parent_readable_id TEXT,
    PRIMARY KEY (step_id, id)
);

CREATE TABLE v1_durable_sleep (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    sleep_until TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, sleep_until, id)
);
