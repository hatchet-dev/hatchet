-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" ADD COLUMN "inputJsonSchema" JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" DROP COLUMN "inputJsonSchema";
-- +goose StatementEnd
