/*
  Warnings:

  - You are about to drop the column `planCode` on the `TenantSubscription` table. All the data in the column will be lost.

*/
-- AlterTable
ALTER TABLE "TenantSubscription" DROP COLUMN "planCode",
ADD COLUMN     "plan" "TenantSubscriptionPlanCodes" NOT NULL DEFAULT 'free';
