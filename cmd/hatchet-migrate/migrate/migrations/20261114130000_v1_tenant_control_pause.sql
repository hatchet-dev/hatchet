-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_tenant_control
    ALTER COLUMN ingest_per_second DROP NOT NULL,
    ADD COLUMN paused BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN paused_at TIMESTAMPTZ,
    ADD CONSTRAINT v1_tenant_control_sets_a_control CHECK (paused OR ingest_per_second IS NOT NULL);

CREATE OR REPLACE FUNCTION v1_tenant_control_paused_at_function()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT NEW.paused THEN
        NEW.paused_at := NULL;
    ELSIF TG_OP = 'INSERT' THEN
        NEW.paused_at := NOW();
    ELSIF OLD.paused THEN
        NEW.paused_at := OLD.paused_at;
    ELSE
        NEW.paused_at := NOW();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER v1_tenant_control_paused_at_trigger
BEFORE INSERT OR UPDATE ON v1_tenant_control
FOR EACH ROW
EXECUTE FUNCTION v1_tenant_control_paused_at_function();

-- Ending a pause moves the schedule deadline of each of the tenant's queued items forward by the
-- time the tenant was paused, so that a pause never times out work it held back. It runs in the
-- statement that ends the pause, so no process sees the tenant resumed before its deadlines moved.
-- v1_paused_workflow_queue_item is left alone: an item gets a new deadline when it leaves it.
CREATE OR REPLACE FUNCTION v1_tenant_control_resume_function()
RETURNS TRIGGER AS $$
DECLARE
    paused_for INTERVAL := NOW() - OLD.paused_at;
BEGIN
    -- the row is going with its tenant, which leaves nothing to schedule
    IF NOT EXISTS (SELECT 1 FROM "Tenant" WHERE "id" = OLD.tenant_id) THEN
        RETURN NULL;
    END IF;

    UPDATE v1_queue_item
    SET schedule_timeout_at = schedule_timeout_at + paused_for
    WHERE tenant_id = OLD.tenant_id AND schedule_timeout_at IS NOT NULL;

    -- filled slots too: a task can hold its slot while it waits in v1_rate_limited_queue_items, and
    -- a running task's slot deadline is not looked at
    UPDATE v1_concurrency_slot
    SET schedule_timeout_at = schedule_timeout_at + paused_for
    WHERE tenant_id = OLD.tenant_id;

    UPDATE v1_batched_queue_item
    SET schedule_timeout_at = schedule_timeout_at + paused_for
    WHERE tenant_id = OLD.tenant_id AND schedule_timeout_at IS NOT NULL;

    UPDATE v1_rate_limited_queue_items
    SET schedule_timeout_at = schedule_timeout_at + paused_for
    WHERE tenant_id = OLD.tenant_id AND schedule_timeout_at IS NOT NULL;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER v1_tenant_control_resume_update_trigger
AFTER UPDATE ON v1_tenant_control
FOR EACH ROW
WHEN (OLD.paused AND NOT NEW.paused)
EXECUTE FUNCTION v1_tenant_control_resume_function();

CREATE TRIGGER v1_tenant_control_resume_delete_trigger
AFTER DELETE ON v1_tenant_control
FOR EACH ROW
WHEN (OLD.paused)
EXECUTE FUNCTION v1_tenant_control_resume_function();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- End every pause while the triggers still move the deadlines.
DELETE FROM v1_tenant_control WHERE ingest_per_second IS NULL;
UPDATE v1_tenant_control SET paused = FALSE WHERE paused;

DROP TRIGGER v1_tenant_control_resume_delete_trigger ON v1_tenant_control;
DROP TRIGGER v1_tenant_control_resume_update_trigger ON v1_tenant_control;
DROP TRIGGER v1_tenant_control_paused_at_trigger ON v1_tenant_control;
DROP FUNCTION v1_tenant_control_resume_function();
DROP FUNCTION v1_tenant_control_paused_at_function();

ALTER TABLE v1_tenant_control
    DROP CONSTRAINT v1_tenant_control_sets_a_control,
    DROP COLUMN paused_at,
    DROP COLUMN paused,
    ALTER COLUMN ingest_per_second SET NOT NULL;
-- +goose StatementEnd
