-- +goose Up
-- +goose NO TRANSACTION
ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'TASK_RUN';

ALTER TYPE "LimitResource" ADD VALUE IF NOT EXISTS 'WORKER_SLOT';
-- Insert WORKER_SLOT entries (limitValue = 1000x WORKER limit, customValueMeter = true)
INSERT INTO "TenantResourceLimit" (
    "id",
    "createdAt",
    "updatedAt",
    "resource",
    "tenantId",
    "limitValue",
    "alarmValue",
    "value",
    "window",
    "lastRefill",
    "customValueMeter"
)
SELECT
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    'WORKER_SLOT',
    "tenantId",
    least(("limitValue"::bigint) * 1000, 2147483647)::integer,
    least(("alarmValue"::bigint) * 1000, 2147483647)::integer,
    0,
    "window",
    "lastRefill",
    true
FROM "TenantResourceLimit"
WHERE "resource" = 'WORKER'
ON CONFLICT ("tenantId", "resource") DO NOTHING;

-- Insert TASK_RUN entries (limitValue = 10x WORKFLOW_RUN limit, customValueMeter = false)
INSERT INTO "TenantResourceLimit" (
    "id",
    "createdAt",
    "updatedAt",
    "resource",
    "tenantId",
    "limitValue",
    "alarmValue",
    "value",
    "window",
    "lastRefill",
    "customValueMeter"
)
SELECT
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    'TASK_RUN',
    "tenantId",
    least(("limitValue"::bigint) * 10, 2147483647)::integer,
    least(("alarmValue"::bigint) * 10, 2147483647)::integer,
    0,
    "window",
    "lastRefill",
    false
FROM "TenantResourceLimit"
WHERE "resource" = 'WORKFLOW_RUN'
ON CONFLICT ("tenantId", "resource") DO NOTHING;

-- +goose Down
-- +goose StatementBegin
DELETE FROM "TenantResourceLimit"
WHERE "resource" IN ('WORKER_SLOT', 'TASK_RUN');
-- +goose StatementEnd
