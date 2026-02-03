-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'v1_worker_slot_group'
    ) THEN
        CREATE TYPE v1_worker_slot_group AS ENUM ('SLOTS', 'DURABLE_SLOTS');
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE "Worker"
    ADD COLUMN IF NOT EXISTS "durableMaxRuns" INTEGER NOT NULL DEFAULT 0;

ALTER TABLE "Step"
    ADD COLUMN IF NOT EXISTS "isDurable" BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE v1_task_runtime
    ADD COLUMN IF NOT EXISTS slot_group v1_worker_slot_group NOT NULL DEFAULT 'SLOTS';

-- TODO: concurrently create the index
-- -- +goose NO TRANSACTION
CREATE INDEX IF NOT EXISTS v1_task_runtime_tenantId_workerId_slotGroup_idx
    ON v1_task_runtime (tenant_id ASC, worker_id ASC, slot_group ASC)
    WHERE worker_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS v1_task_runtime_tenantId_workerId_slotGroup_idx;

ALTER TABLE v1_task_runtime
    DROP COLUMN IF EXISTS slot_group;

ALTER TABLE "Step"
    DROP COLUMN IF EXISTS "isDurable";

ALTER TABLE "Worker"
    DROP COLUMN IF EXISTS "durableMaxRuns";

DROP TYPE IF EXISTS v1_worker_slot_group;
