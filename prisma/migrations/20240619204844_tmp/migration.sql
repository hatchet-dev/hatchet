/*
  Warnings:

  - The `planCode` column on the `TenantSubscription` table would be dropped and recreated. This will lead to data loss if there is data in the column.

*/
-- CreateEnum
CREATE TYPE "TenantSubscriptionPlanCodes" AS ENUM ('free', 'starter', 'growth', 'enterprise');

-- CreateEnum
CREATE TYPE "TenantSubscriptionPeriod" AS ENUM ('monthly', 'annual');

-- AlterTable
ALTER TABLE "TenantSubscription" ADD COLUMN     "period" "TenantSubscriptionPeriod",
DROP COLUMN "planCode",
ADD COLUMN     "planCode" "TenantSubscriptionPlanCodes" NOT NULL DEFAULT 'free';
