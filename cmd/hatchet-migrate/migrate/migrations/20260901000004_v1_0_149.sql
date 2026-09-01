-- +goose Up
-- +goose StatementBegin
ALTER TABLE "TenantOIDCGroupMapping"
    ADD COLUMN "issuer" TEXT NOT NULL DEFAULT '';
ALTER TABLE "TenantOIDCGroupMapping"
    ALTER COLUMN "issuer" DROP DEFAULT;
DROP INDEX "TenantOIDCGroupMapping_tenantId_group_key";
CREATE UNIQUE INDEX "TenantOIDCGroupMapping_tenantId_issuer_group_key"
    ON "TenantOIDCGroupMapping" ("tenantId", "issuer", "group");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM "TenantOIDCGroupMapping" a
USING "TenantOIDCGroupMapping" b
WHERE a."tenantId" = b."tenantId"
  AND a."group" = b."group"
  AND a."id" > b."id";
DROP INDEX "TenantOIDCGroupMapping_tenantId_issuer_group_key";
CREATE UNIQUE INDEX "TenantOIDCGroupMapping_tenantId_group_key"
    ON "TenantOIDCGroupMapping" ("tenantId", "group");
ALTER TABLE "TenantOIDCGroupMapping" DROP COLUMN "issuer";
-- +goose StatementEnd