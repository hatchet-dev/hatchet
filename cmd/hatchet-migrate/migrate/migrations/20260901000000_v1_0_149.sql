-- +goose Up
-- +goose StatementBegin
ALTER TABLE "TenantMember" ADD COLUMN "oidcIssuer" TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "TenantMember" DROP COLUMN "oidcIssuer";
-- +goose StatementEnd