-- CreateEnum
CREATE TYPE "TenantSubscriptionStatus" AS ENUM ('active', 'pending', 'terminated', 'canceled');

-- CreateTable
CREATE TABLE "TenantSubscription" (
    "tenantId" UUID NOT NULL,
    "planCode" TEXT NOT NULL,
    "status" "TenantSubscriptionStatus" NOT NULL,

    CONSTRAINT "TenantSubscription_pkey" PRIMARY KEY ("tenantId")
);

-- CreateIndex
CREATE UNIQUE INDEX "TenantSubscription_tenantId_key" ON "TenantSubscription"("tenantId");

-- AddForeignKey
ALTER TABLE "TenantSubscription" ADD CONSTRAINT "TenantSubscription_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;
