-- +goose Up
-- +goose StatementBegin
ALTER TABLE "TenantOIDCGroupMapping"
    ADD CONSTRAINT "TenantOIDCGroupMapping_group_check"
    CHECK (length("group") BETWEEN 1 AND 255);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "TenantOIDCGroupMapping"
    DROP CONSTRAINT "TenantOIDCGroupMapping_group_check";
-- +goose StatementEnd