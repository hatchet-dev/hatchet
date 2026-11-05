-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" ADD COLUMN "createWorkflowVersionOpts" JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" DROP COLUMN "createWorkflowVersionOpts";
-- +goose StatementEnd
