-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowTriggerCronRef_parentId_name_idx" ON "WorkflowTriggerCronRef" ("parentId", "name");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "WorkflowTriggerCronRef_WorkflowTriggerCronRef_parentId_name_idx_name_idx";
-- +goose StatementEnd
