-- +goose Up
-- +goose StatementBegin
CREATE TYPE "WorkflowPauseQueueBehavior" AS ENUM ('QUEUE', 'DROP');

ALTER TABLE "Workflow"
    ADD COLUMN "pausedWorkflowCronRunQueueBehavior" "WorkflowPauseQueueBehavior",
    ADD COLUMN "pausedWorkflowScheduledRunQueueBehavior" "WorkflowPauseQueueBehavior",
    ADD COLUMN "pausedWorkflowQueueTTL" INTERVAL,
    ADD CONSTRAINT "Workflow_PausedWorkflowCheck" CHECK (
        (
            "isPaused" = FALSE
            AND "pausedWorkflowCronRunQueueBehavior" IS NULL
            AND "pausedWorkflowScheduledRunQueueBehavior" IS NULL
            AND "pausedWorkflowQueueTTL" IS NULL
        )
        OR (
            "isPaused" = TRUE
            AND "pausedWorkflowCronRunQueueBehavior" IS NOT NULL
            AND "pausedWorkflowScheduledRunQueueBehavior" IS NOT NULL
            AND "pausedWorkflowQueueTTL" IS NOT NULL
        )
    )
;

-- v1_paused_workflow_queue_item stores queue items for workflows that are currently paused.
CREATE TABLE v1_paused_workflow_queue_item (
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

    -- important: inserted at first so we can use it to filter for expired queue items
    CONSTRAINT v1_paused_workflow_queue_itemm_pkey PRIMARY KEY (task_inserted_at, task_id, retry_count)
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

ALTER TABLE "Workflow"
    DROP CONSTRAINT "Workflow_PausedWorkflowCheck",
    DROP COLUMN "pausedWorkflowCronRunQueueBehavior",
    DROP COLUMN "pausedWorkflowScheduledRunQueueBehavior",
    DROP COLUMN "pausedWorkflowQueueTTL"
;

DROP TYPE "WorkflowPauseQueueBehavior";
-- +goose StatementEnd
