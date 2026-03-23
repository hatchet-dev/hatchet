-- +goose Up
-- +goose StatementBegin

-- v1_paused_workflow_queue_items stores queue items for workflows that are currently paused.
CREATE TABLE IF NOT EXISTS v1_paused_workflow_queue_items (
    paused_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    desired_worker_label JSONB,
    CONSTRAINT v1_paused_workflow_queue_items_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

CREATE INDEX IF NOT EXISTS v1_paused_workflow_queue_items_workflow_idx
    ON v1_paused_workflow_queue_items (workflow_id, tenant_id);

ALTER TABLE v1_paused_workflow_queue_items SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'WORKFLOW_PAUSED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'WORKFLOW_UNPAUSED';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS v1_paused_workflow_queue_items_workflow_idx;
DROP TABLE IF EXISTS v1_paused_workflow_queue_items;

ALTER TYPE v1_event_type_olap DROP VALUE IF EXISTS 'WORKFLOW_PAUSED';
ALTER TYPE v1_event_type_olap DROP VALUE IF EXISTS 'WORKFLOW_UNPAUSED';
-- +goose StatementEnd
