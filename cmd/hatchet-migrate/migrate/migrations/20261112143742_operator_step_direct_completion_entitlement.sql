-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenant_entitlement ADD COLUMN operator_step_direct_completion BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenant_entitlement DROP COLUMN operator_step_direct_completion;
-- +goose StatementEnd
