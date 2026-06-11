-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" ADD COLUMN "taskNameToChildTaskNames" JSONB;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" DROP COLUMN "taskNameToChildTaskNames";
-- +goose StatementEnd
