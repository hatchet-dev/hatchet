-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    ADD COLUMN "idempotencyKeyExpression" TEXT,
    ADD COLUMN "idempotencyKeyTtlMs" BIGINT
    ;

ALTER TYPE v1_cel_evaluation_failure_source ADD VALUE IF NOT EXISTS 'IDEMPOTENCY_KEY';

ALTER TABLE v1_task ADD COLUMN idempotency_key TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowVersion"
    DROP COLUMN "idempotencyKeyExpression",
    DROP COLUMN "idempotencyKeyTtlMs"
;

ALTER TABLE v1_task DROP COLUMN idempotency_key;
-- +goose StatementEnd
