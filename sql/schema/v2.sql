
-- CreateTable
CREATE TABLE v2_queue (
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    last_active TIMESTAMP(3),

    CONSTRAINT v2_queue_pkey PRIMARY KEY (tenant_id, name)
);

-- CreateTable
CREATE TABLE v2_task (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    schedule_timeout TEXT NOT NULL,
    step_timeout TEXT,
    priority INTEGER DEFAULT 1,
    sticky "StickyStrategy",
    desired_worker_id UUID,
    external_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    input JSONB NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    internal_retry_count INTEGER NOT NULL DEFAULT 0,
    app_retry_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT v2_task_pkey PRIMARY KEY (id)
);

alter table v2_task set (
    autovacuum_vacuum_scale_factor = '0.1', 
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE TYPE v2_task_event_type AS ENUM (
    'COMPLETED',
    'FAILED',
    'CANCELLED'
);

-- CreateTable
CREATE TABLE v2_task_event (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    event_type v2_task_event_type NOT NULL,
    created_at TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    data JSONB,
    CONSTRAINT v2_task_event_pkey PRIMARY KEY (id)
);

-- CreateTable
CREATE TABLE v2_queue_item (
    id bigint GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id bigint NOT NULL,
    action_id TEXT NOT NULL,
    step_id UUID NOT NULL,
    schedule_timeout_at TIMESTAMP(3),
    step_timeout TEXT,
    priority INTEGER NOT NULL DEFAULT 1,
    sticky "StickyStrategy",
    desired_worker_id UUID,
    -- TODO: REMOVE is_queued
    is_queued BOOLEAN NOT NULL,
    retry_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT v2_queue_item_pkey PRIMARY KEY (id)
);

alter table v2_queue_item set (
    autovacuum_vacuum_scale_factor = '0.1', 
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);

CREATE INDEX v2_queue_item_isQueued_priority_tenantId_queue_id_idx ON v2_queue_item (
    is_queued ASC,
    tenant_id ASC,
    queue ASC,
    priority DESC,
    id ASC
);

CREATE OR REPLACE FUNCTION v2_task_to_v2_queue_item_insert_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_queue_item (
        tenant_id,
        queue,
        task_id,
        action_id,
        step_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        is_queued,
        retry_count
    )
    VALUES (
        NEW.tenant_id,
        NEW.queue,
        NEW.id,
        NEW.action_id,
        NEW.step_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(NEW.schedule_timeout),
        NEW.step_timeout,
        COALESCE(NEW.priority, 1),
        NEW.sticky,
        NEW.desired_worker_id,
        TRUE,
        NEW.retry_count
    );
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v2_task_to_v2_queue_item_insert_trigger
AFTER INSERT ON v2_task
FOR EACH ROW
EXECUTE PROCEDURE v2_task_to_v2_queue_item_insert_function();

CREATE OR REPLACE FUNCTION v2_task_to_v2_queue_item_update_retry_count_function()
RETURNS TRIGGER AS
$$
BEGIN
    INSERT INTO v2_queue_item (
        tenant_id,
        queue,
        task_id,
        action_id,
        step_id,
        schedule_timeout_at,
        step_timeout,
        priority,
        sticky,
        desired_worker_id,
        is_queued,
        retry_count
    )
    VALUES (
        NEW.tenant_id,
        NEW.queue,
        NEW.id,
        NEW.action_id,
        NEW.step_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(NEW.schedule_timeout),
        NEW.step_timeout,
        -- retries are always given priority=4
        4,
        NEW.sticky,
        NEW.desired_worker_id,
        TRUE,
        NEW.retry_count
    );
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER v2_task_to_v2_queue_item_update_retry_count_trigger
AFTER UPDATE OF retry_count ON v2_task
FOR EACH ROW
WHEN (OLD.retry_count IS DISTINCT FROM NEW.retry_count)
EXECUTE PROCEDURE v2_task_to_v2_queue_item_update_retry_count_function();

-- CreateTable
CREATE TABLE v2_task_runtime (
    task_id bigint NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    timeout_at TIMESTAMP(3) NOT NULL,

    CONSTRAINT v2_task_runtime_pkey PRIMARY KEY (task_id, retry_count)
);

CREATE INDEX v2_task_runtime_tenantId_workerId_idx ON v2_task_runtime (tenant_id ASC, worker_id ASC);

CREATE INDEX v2_task_runtime_tenantId_timeoutAt_idx ON v2_task_runtime (tenant_id ASC, timeout_at ASC);

alter table v2_task_runtime set (
    autovacuum_vacuum_scale_factor = '0.1', 
    autovacuum_analyze_scale_factor='0.05',
    autovacuum_vacuum_threshold='25',
    autovacuum_analyze_threshold='25',
    autovacuum_vacuum_cost_delay='10',
    autovacuum_vacuum_cost_limit='1000'
);