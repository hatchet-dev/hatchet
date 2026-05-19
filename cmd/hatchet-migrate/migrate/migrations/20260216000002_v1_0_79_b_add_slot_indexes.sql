-- +goose Up
-- +goose NO TRANSACTION

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_task_runtime_slot_tenant_worker_type_idx
    ON v1_task_runtime_slot (tenant_id ASC, worker_id ASC, slot_type ASC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS v1_step_slot_request_step_idx
    ON v1_step_slot_request (step_id ASC);

-- +goose Down
DROP INDEX IF EXISTS v1_task_runtime_slot_tenant_worker_type_idx;
DROP INDEX IF EXISTS v1_step_slot_request_step_idx;
