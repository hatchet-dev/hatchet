-- +goose Up
-- +goose StatementBegin

ALTER TABLE tenant_entitlement
ADD COLUMN IF NOT EXISTS strict_additional_metadata_filters BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE tenant_entitlement
DROP COLUMN IF EXISTS strict_additional_metadata_filters;

-- +goose StatementEnd
