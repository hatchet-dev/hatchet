-- +goose Up
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

-- +goose Down
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
          AND s."batchSize" IS NOT NULL
          AND s."batchSize" >= 1
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
