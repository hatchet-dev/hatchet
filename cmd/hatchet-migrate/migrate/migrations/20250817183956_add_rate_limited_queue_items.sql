-- +goose Up
-- +goose StatementBegin
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE v1_rate_limited_queue_items;
-- +goose StatementEnd
