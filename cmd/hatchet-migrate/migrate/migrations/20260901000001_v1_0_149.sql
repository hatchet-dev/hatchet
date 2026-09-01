-- +goose Up
-- +goose StatementBegin
CREATE INDEX "TenantMember_userId_oidcIssuer_idx"
    ON "TenantMember" ("userId", "oidcIssuer")
    WHERE "oidcIssuer" IS NOT NULL;

CREATE TABLE "TenantOIDCGroupMapping" (
    "id" UUID NOT NULL DEFAULT gen_random_uuid(),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "group" TEXT NOT NULL,
    "role" "TenantMemberRole" NOT NULL,
    CONSTRAINT "TenantOIDCGroupMapping_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "TenantOIDCGroupMapping_role_check" CHECK ("role" <> 'OWNER'),
    CONSTRAINT "TenantOIDCGroupMapping_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE UNIQUE INDEX "TenantOIDCGroupMapping_tenantId_group_key"
    ON "TenantOIDCGroupMapping" ("tenantId", "group");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "TenantOIDCGroupMapping";
DROP INDEX "TenantMember_userId_oidcIssuer_idx";
-- +goose StatementEnd