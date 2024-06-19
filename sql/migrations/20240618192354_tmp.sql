-- Create enum type "TenantSubscriptionStatus"
CREATE TYPE "TenantSubscriptionStatus" AS ENUM ('active', 'pending', 'terminated', 'canceled');
-- Create "TenantSubscription" table
CREATE TABLE "TenantSubscription" ("tenantId" uuid NOT NULL, "planCode" text NOT NULL, "status" "TenantSubscriptionStatus" NOT NULL, PRIMARY KEY ("tenantId"), CONSTRAINT "TenantSubscription_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "TenantSubscription_tenantId_key" to table: "TenantSubscription"
CREATE UNIQUE INDEX "TenantSubscription_tenantId_key" ON "TenantSubscription" ("tenantId");
