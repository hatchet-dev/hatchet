-- +goose Up
-- +goose StatementBegin
CREATE TYPE idempotency_method AS ENUM ('TTL', 'STATUS');

ALTER TABLE "WorkflowVersion"
ADD COLUMN "idempotencyMethod" idempotency_method;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE idempotency_method;
ALTER TABLE "WorkflowVersion"
DROP COLUMN "idempotencyMethod";
-- +goose StatementEnd
