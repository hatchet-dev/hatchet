-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS "Step_dagOrchestrator_tenant_action_idx"
    ON "Step" ("tenantId", "actionId")
    INCLUDE ("jobId")
    WHERE "isDagOrchestrator" AND "deletedAt" IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "Step_dagOrchestrator_tenant_action_idx";
-- +goose StatementEnd
