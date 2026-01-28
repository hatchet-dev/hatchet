-- +goose Up
-- +goose StatementBegin
-- Add scope_expression and static_payload to v1_incoming_webhook
-- scope_expression: CEL expression to extract scope from webhook payload (optional)
-- static_payload: Static JSON payload to attach to events from this webhook (optional)
ALTER TABLE v1_incoming_webhook
    ADD COLUMN scope_expression TEXT,
    ADD COLUMN static_payload JSONB;

-- Add constraint to prevent empty scope_expression (if provided)
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
