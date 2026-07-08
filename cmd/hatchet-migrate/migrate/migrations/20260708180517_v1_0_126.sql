-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    ADD COLUMN "idempotencyKeyExpression" TEXT,
    ADD COLUMN "idempotencyKeyTtlMs" BIGINT
    ;

ALTER TYPE v1_cel_evaluation_failure_source ADD VALUE IF NOT EXISTS 'IDEMPOTENCY_KEY';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    DROP COLUMN "idempotencyKeyExpression",
    DROP COLUMN "idempotencyKeyTtlMs"
;
-- +goose StatementEnd
