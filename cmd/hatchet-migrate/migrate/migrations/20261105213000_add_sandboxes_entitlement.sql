-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenant_entitlement ADD COLUMN sandboxes BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN IF EXISTS sandboxes;
-- +goose StatementEnd
