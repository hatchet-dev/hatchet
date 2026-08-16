-- +goose Up
-- +goose StatementBegin
CREATE TYPE paused_workflow_queue_strategy AS ENUM ('QUEUE', 'DROP');

CREATE TABLE v1_paused_workflow_config (
    workflow_id UUID NOT NULL,
    is_paused BOOLEAN NOT NULL DEFAULT FALSE,
    cron_run_queue_strategy paused_workflow_queue_strategy NOT NULL DEFAULT 'QUEUE',
    scheduled_run_queue_strategy paused_workflow_queue_strategy NOT NULL DEFAULT 'QUEUE',

    CONSTRAINT v1_paused_workflow_config_pkey PRIMARY KEY (workflow_id)
);

-- v1_paused_workflow_queue_item stores queue items for workflows that are currently paused.
CREATE TABLE v1_paused_workflow_queue_item (
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
    batch_key TEXT,
    CONSTRAINT v1_paused_workflow_queue_itemm_pkey PRIMARY KEY (task_id, task_inserted_at, retry_count)
);

CREATE INDEX v1_paused_workflow_queue_item_workflow_idx
    ON v1_paused_workflow_queue_item (workflow_id, tenant_id);

ALTER TABLE v1_paused_workflow_queue_item SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_paused_workflow_queue_item;
DROP TABLE v1_paused_workflow_config;
DROP TYPE paused_workflow_queue_strategy;
-- +goose StatementEnd
