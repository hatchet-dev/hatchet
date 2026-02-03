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

ALTER TABLE "Worker"
    ADD COLUMN IF NOT EXISTS "durableMaxRuns" INTEGER NOT NULL DEFAULT 0;

ALTER TABLE "Step"
    ADD COLUMN IF NOT EXISTS "isDurable" BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE v1_task_runtime
    ADD COLUMN IF NOT EXISTS slot_group v1_worker_slot_group NOT NULL DEFAULT 'SLOTS';

CREATE TABLE IF NOT EXISTS v1_worker_slot_config (
    tenant_id UUID NOT NULL,
    worker_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    max_units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, worker_id, slot_type)
);

CREATE TABLE IF NOT EXISTS v1_step_slot_request (
    tenant_id UUID NOT NULL,
    step_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, step_id, slot_type)
);

CREATE TABLE IF NOT EXISTS v1_task_runtime_slot (
    tenant_id UUID NOT NULL,
    task_id BIGINT NOT NULL,
    task_inserted_at TIMESTAMPTZ NOT NULL,
    retry_count INTEGER NOT NULL,
    worker_id UUID NOT NULL,
    slot_type TEXT NOT NULL,
    units INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, task_inserted_at, retry_count, slot_type)
);
-- +goose StatementEnd

-- TODO: concurrently create the index
-- -- +goose NO TRANSACTION
CREATE INDEX IF NOT EXISTS v1_task_runtime_tenantId_workerId_slotGroup_idx
    ON v1_task_runtime (tenant_id ASC, worker_id ASC, slot_group ASC)
    WHERE worker_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS v1_task_runtime_slot_tenant_worker_type_idx
    ON v1_task_runtime_slot (tenant_id ASC, worker_id ASC, slot_type ASC);

CREATE INDEX IF NOT EXISTS v1_step_slot_request_step_idx
    ON v1_step_slot_request (step_id ASC);

-- +goose StatementBegin
INSERT INTO v1_worker_slot_config (tenant_id, worker_id, slot_type, max_units)
SELECT
    "tenantId",
    "id",
    'default'::text,
    "maxRuns"
FROM "Worker"
WHERE "maxRuns" IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO v1_worker_slot_config (tenant_id, worker_id, slot_type, max_units)
SELECT
    "tenantId",
    "id",
    'durable'::text,
    "durableMaxRuns"
FROM "Worker"
WHERE "durableMaxRuns" IS NOT NULL AND "durableMaxRuns" > 0
ON CONFLICT DO NOTHING;

INSERT INTO v1_step_slot_request (tenant_id, step_id, slot_type, units)
SELECT
    "tenantId",
    "id",
    CASE WHEN "isDurable" THEN 'durable'::text ELSE 'default'::text END,
    1
FROM "Step"
ON CONFLICT DO NOTHING;

INSERT INTO v1_task_runtime_slot (
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    worker_id,
    slot_type,
    units
)
SELECT
    tenant_id,
    task_id,
    task_inserted_at,
    retry_count,
    worker_id,
    CASE
        WHEN slot_group = 'DURABLE_SLOTS'::v1_worker_slot_group THEN 'durable'::text
        ELSE 'default'::text
    END,
    1
FROM v1_task_runtime
WHERE worker_id IS NOT NULL
ON CONFLICT DO NOTHING;

ALTER TABLE "Worker"
    DROP COLUMN IF EXISTS "maxRuns",
    DROP COLUMN IF EXISTS "durableMaxRuns";

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS v1_task_runtime_slot_tenant_worker_type_idx;
DROP INDEX IF EXISTS v1_step_slot_request_step_idx;
DROP TABLE IF EXISTS v1_task_runtime_slot;
DROP TABLE IF EXISTS v1_step_slot_request;
DROP TABLE IF EXISTS v1_worker_slot_config;

DROP INDEX IF EXISTS v1_task_runtime_tenantId_workerId_slotGroup_idx;

ALTER TABLE v1_task_runtime
    DROP COLUMN IF EXISTS slot_group;

ALTER TABLE "Step"
    DROP COLUMN IF EXISTS "isDurable";

ALTER TABLE "Worker"
    ADD COLUMN IF NOT EXISTS "maxRuns" INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS "durableMaxRuns" INTEGER NOT NULL DEFAULT 0;

DROP TYPE IF EXISTS v1_worker_slot_group;
-- +goose StatementEnd
