-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    ADD COLUMN "idempotencyKeyExpression" TEXT,
    ADD COLUMN "idempotencyKeyTtl" BIGINT
    ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    DROP COLUMN "idempotencyKeyExpression",
    DROP COLUMN "idempotencyKeyTtl"
;
-- +goose StatementEnd
