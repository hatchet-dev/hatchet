-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" ADD COLUMN "idempotencyKeyExpression" TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion" DROP COLUMN "idempotencyKeyExpression";
-- +goose StatementEnd
