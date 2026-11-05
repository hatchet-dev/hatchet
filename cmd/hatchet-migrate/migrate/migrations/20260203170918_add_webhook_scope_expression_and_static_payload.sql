-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook
    ADD COLUMN scope_expression TEXT,
    ADD COLUMN static_payload JSONB;

ALTER TABLE v1_incoming_webhook
    ADD CONSTRAINT v1_incoming_webhook_scope_expression_not_empty
    CHECK (scope_expression IS NULL OR LENGTH(scope_expression) > 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_incoming_webhook
    DROP CONSTRAINT IF EXISTS v1_incoming_webhook_scope_expression_not_empty;

ALTER TABLE v1_incoming_webhook
    DROP COLUMN IF EXISTS scope_expression,
    DROP COLUMN IF EXISTS static_payload;
-- +goose StatementEnd
