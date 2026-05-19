-- +goose Up
-- +goose StatementBegin

LOCK TABLE "TenantResourceLimit" IN ACCESS EXCLUSIVE MODE;

DELETE FROM "TenantResourceLimit" WHERE resource = 'WORKFLOW_RUN';

CREATE TYPE "LimitResource_new" AS ENUM (
    'TASK_RUN',
    'EVENT',
    'WORKER',
    'WORKER_SLOT',
    'CRON',
    'SCHEDULE',
    'INCOMING_WEBHOOK'
);

ALTER TABLE "TenantResourceLimit"
    ALTER COLUMN resource TYPE "LimitResource_new"
    USING resource::text::"LimitResource_new";

ALTER TABLE "TenantResourceLimitAlert"
    ALTER COLUMN resource TYPE "LimitResource_new"
    USING resource::text::"LimitResource_new";

DROP TYPE "LimitResource" CASCADE;

ALTER TYPE "LimitResource_new" RENAME TO "LimitResource";

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

LOCK TABLE "TenantResourceLimit" IN ACCESS EXCLUSIVE MODE;

CREATE TYPE "LimitResource_old" AS ENUM (
    'WORKFLOW_RUN',
    'TASK_RUN',
    'EVENT',
    'WORKER',
    'WORKER_SLOT',
    'CRON',
    'SCHEDULE',
    'INCOMING_WEBHOOK'
);

ALTER TABLE "TenantResourceLimit"
    ALTER COLUMN resource TYPE "LimitResource_old"
    USING resource::text::"LimitResource_old";

DROP TYPE "LimitResource" CASCADE;

ALTER TYPE "LimitResource_old" RENAME TO "LimitResource";

-- +goose StatementEnd
