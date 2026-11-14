-- +goose Up
-- +goose StatementBegin
ALTER TABLE "TenantMember" ADD COLUMN "canViewPayloads" boolean NOT NULL DEFAULT true;
ALTER TABLE "TenantInviteLink" ADD COLUMN "canViewPayloads" boolean NOT NULL DEFAULT true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "TenantMember" DROP COLUMN "canViewPayloads";
ALTER TABLE "TenantInviteLink" DROP COLUMN "canViewPayloads";
-- +goose StatementEnd
