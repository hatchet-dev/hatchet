-- +goose Up
-- +goose StatementBegin
-- kept apart from "RateLimit" so a key can be paused before its first run and upserts can't clear a pause
CREATE TABLE v1_rate_limit_pause (
    tenant_id UUID NOT NULL,
    key TEXT NOT NULL,
    -- the key's window is frozen from paused_at until paused_until, and shifted by that span once the pause ends
    paused_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_until TIMESTAMPTZ NOT NULL,
    CONSTRAINT v1_rate_limit_pause_pkey PRIMARY KEY (tenant_id, key)
);

-- returns when the latest active pause on any of the task's static or dynamic rate limit keys ends, or NULL if none is paused
CREATE OR REPLACE FUNCTION v1_task_rate_limit_paused_until(
    _tenant_id UUID,
    _task_id BIGINT,
    _task_inserted_at TIMESTAMPTZ,
    _step_id UUID
) RETURNS TIMESTAMPTZ AS $$
    SELECT MAX(p.paused_until)
    FROM v1_rate_limit_pause p
    WHERE
        p.tenant_id = _tenant_id
        AND p.paused_until > NOW()
        AND (
            EXISTS (
                SELECT 1
                FROM "StepRateLimit" srl
                WHERE srl."stepId" = _step_id AND srl."rateLimitKey" = p.key
            )
            OR EXISTS (
                SELECT 1
                FROM v1_task_expression_eval ee
                WHERE
                    ee.task_id = _task_id
                    AND ee.task_inserted_at = _task_inserted_at
                    AND ee.kind = 'DYNAMIC_RATE_LIMIT_KEY'
                    AND ee.value_str = p.key
            )
        );
$$ LANGUAGE sql STABLE;

-- set when the item is held by a paused rate limit key rather than (only) a paused workflow
ALTER TABLE v1_paused_workflow_queue_item ADD COLUMN rate_limit_paused_until TIMESTAMPTZ;

CREATE INDEX v1_paused_workflow_queue_item_rate_limit_idx
    ON v1_paused_workflow_queue_item (tenant_id, rate_limit_paused_until)
    WHERE rate_limit_paused_until IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX v1_paused_workflow_queue_item_rate_limit_idx;
ALTER TABLE v1_paused_workflow_queue_item DROP COLUMN rate_limit_paused_until;
DROP FUNCTION v1_task_rate_limit_paused_until(UUID, BIGINT, TIMESTAMPTZ, UUID);
DROP TABLE v1_rate_limit_pause;
-- +goose StatementEnd
