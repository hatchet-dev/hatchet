/*
  Warnings:

  - You are about to drop the `TenantSubscription` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "TenantSubscription" DROP CONSTRAINT "TenantSubscription_tenantId_fkey";

-- DropTable
DROP TABLE "TenantSubscription";

-- DropEnum
DROP TYPE "TenantSubscriptionPeriod";

-- DropEnum
DROP TYPE "TenantSubscriptionPlanCodes";

-- DropEnum
DROP TYPE "TenantSubscriptionStatus";
