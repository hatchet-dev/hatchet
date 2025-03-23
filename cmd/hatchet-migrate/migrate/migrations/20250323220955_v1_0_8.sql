-- +goose Up
-- +goose StatementBegin
ALTER TYPE "LimitResource" ADD VALUE 'TASK_RUN';

ALTER TYPE "LimitResource" ADD VALUE 'WORKER_SLOT';

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
    "limitValue" * 1000,
    "alarmValue" * 1000,
    0,
    "window",
    "lastRefill",
    true
FROM "TenantResourceLimit"
WHERE "resource" = 'WORKER';

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
    "limitValue" * 10,
    "alarmValue" * 10,
    0,
    "window",
    "lastRefill",
    false
FROM "TenantResourceLimit"
WHERE "resource" = 'WORKFLOW_RUN';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM "TenantResourceLimit"
WHERE "resource" IN ('WORKER_SLOT', 'TASK_RUN');
-- +goose StatementEnd
