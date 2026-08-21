-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" ADD COLUMN "displayName" TEXT;
ALTER TABLE "Step" ADD COLUMN "displayName" TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Step" DROP COLUMN "displayName";
ALTER TABLE "WorkflowVersion" DROP COLUMN "displayName";
-- +goose StatementEnd
