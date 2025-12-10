-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LeaseKind" ADD VALUE IF NOT EXISTS 'BATCH';

CREATE TABLE IF NOT EXISTS v1_batched_queue_item (
    id BIGINT GENERATED ALWAYS AS IDENTITY,
    tenant_id UUID NOT NULL,
    queue TEXT NOT NULL,
    task_id BIGINT NOT NULL,
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
    batch_key TEXT NOT NULL,
    inserted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_batched_queue_item_pkey PRIMARY KEY (id),
    CONSTRAINT v1_batched_queue_item_task_key UNIQUE (task_id, task_inserted_at, retry_count)
);

ALTER TABLE v1_batched_queue_item SET (
    autovacuum_vacuum_scale_factor = '0.1',
    autovacuum_analyze_scale_factor = '0.05',
    autovacuum_vacuum_threshold = '25',
    autovacuum_analyze_threshold = '25',
    autovacuum_vacuum_cost_delay = '10',
    autovacuum_vacuum_cost_limit = '1000'
);

CREATE INDEX IF NOT EXISTS v1_batched_queue_item_lookup_idx
    ON v1_batched_queue_item (tenant_id ASC, step_id ASC, batch_key ASC, inserted_at ASC);

CREATE INDEX IF NOT EXISTS v1_batched_queue_item_queue_idx
    ON v1_batched_queue_item (tenant_id ASC, queue ASC, priority DESC, inserted_at ASC);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION v1_queue_item_redirect_to_batch_fn()
RETURNS TRIGGER AS $$
DECLARE
    trimmed_key TEXT;
    has_batch_config BOOLEAN;
BEGIN
    trimmed_key := NULLIF(BTRIM(NEW.batch_key), '');

    IF trimmed_key IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM "Step" s
        WHERE s."id" = NEW.step_id
          AND s."batch_size" IS NOT NULL
          AND s."batch_size" >= 1
    ) INTO has_batch_config;

    IF NOT has_batch_config THEN
        RETURN NEW;
    END IF;

    INSERT INTO v1_batched_queue_item (
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
        retry_count,
        batch_key,
        inserted_at
    )
    VALUES (
        NEW.tenant_id,
        NEW.queue,
        NEW.task_id,
        NEW.task_inserted_at,
        NEW.external_id,
        NEW.action_id,
        NEW.step_id,
        NEW.workflow_id,
        NEW.workflow_run_id,
        NEW.schedule_timeout_at,
        NEW.step_timeout,
        NEW.priority,
        NEW.sticky,
        NEW.desired_worker_id,
        NEW.retry_count,
        trimmed_key,
        CURRENT_TIMESTAMP
    )
    ON CONFLICT (task_id, task_inserted_at, retry_count) DO NOTHING;

    DELETE FROM v1_queue_item WHERE id = NEW.id;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_queue_item_redirect_to_batch ON v1_queue_item;

CREATE TRIGGER v1_queue_item_redirect_to_batch
AFTER INSERT ON v1_queue_item
FOR EACH ROW
EXECUTE PROCEDURE v1_queue_item_redirect_to_batch_fn();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS v1_queue_item_redirect_to_batch ON v1_queue_item;
DROP FUNCTION IF EXISTS v1_queue_item_redirect_to_batch_fn();

DROP INDEX IF EXISTS v1_batched_queue_item_queue_idx;
DROP INDEX IF EXISTS v1_batched_queue_item_lookup_idx;
DROP TABLE IF EXISTS v1_batched_queue_item;

SELECT 'no-op: cannot remove value from "LeaseKind"' AS notice;
-- +goose StatementEnd
