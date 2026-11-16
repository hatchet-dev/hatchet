-- +goose Up
-- +goose NO TRANSACTION
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS "ix_WorkflowTriggerScheduledRef_triggerAt_tickerId" ON "WorkflowTriggerScheduledRef" ("triggerAt", "tickerId");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "ix_WorkflowTriggerScheduledRef_triggerAt_tickerId";
-- +goose StatementEnd
