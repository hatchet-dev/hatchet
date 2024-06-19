-- Create enum type "TenantSubscriptionPeriod"
CREATE TYPE "TenantSubscriptionPeriod" AS ENUM ('monthly', 'annual');
-- Create enum type "TenantSubscriptionPlanCodes"
CREATE TYPE "TenantSubscriptionPlanCodes" AS ENUM ('free', 'starter', 'growth', 'enterprise');
-- Modify "TenantSubscription" table
ALTER TABLE "TenantSubscription" DROP COLUMN "planCode", ADD COLUMN "period" "TenantSubscriptionPeriod" NULL, ADD COLUMN "plan" "TenantSubscriptionPlanCodes" NOT NULL DEFAULT 'free';
