-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY idx_workflow_triggers_id_version_id_tenant_id ON "WorkflowTriggers" ("id", "workflowVersionId", "tenantId");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY idx_workflow_triggers_id_version_id_tenant_id;
-- +goose StatementEnd
