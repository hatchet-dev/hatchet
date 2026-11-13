-- +goose Up

ALTER TABLE v1_task_runtime ADD COLUMN IF NOT EXISTS evicted_at TIMESTAMPTZ DEFAULT NULL;

CREATE INDEX IF NOT EXISTS v1_task_runtime_tenant_worker_not_evicted_idx
    ON v1_task_runtime (tenant_id, worker_id) WHERE evicted_at IS NULL;

ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'DURABLE_EVICTED';
ALTER TYPE v1_event_type_olap ADD VALUE IF NOT EXISTS 'DURABLE_RESTORING';


-- +goose Down
ALTER TABLE v1_task_runtime DROP COLUMN IF EXISTS evicted_at;
-- NOTE: Postgres does not support removing enum values.
-- The 'DURABLE_EVICTED' and 'DURABLE_RESTORING' values in v1_event_type_olap cannot be reverted.
