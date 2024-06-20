/*
  Warnings:

  - The values [annual] on the enum `TenantSubscriptionPeriod` will be removed. If these variants are still used in the database, this will fail.

*/
-- AlterEnum
BEGIN;
CREATE TYPE "TenantSubscriptionPeriod_new" AS ENUM ('monthly', 'yearly');
ALTER TABLE "TenantSubscription" ALTER COLUMN "period" TYPE "TenantSubscriptionPeriod_new" USING ("period"::text::"TenantSubscriptionPeriod_new");
ALTER TYPE "TenantSubscriptionPeriod" RENAME TO "TenantSubscriptionPeriod_old";
ALTER TYPE "TenantSubscriptionPeriod_new" RENAME TO "TenantSubscriptionPeriod";
DROP TYPE "TenantSubscriptionPeriod_old";
COMMIT;
