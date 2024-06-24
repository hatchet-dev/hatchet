-- Add value to enum type: "LimitResource"
ALTER TYPE "LimitResource" ADD VALUE 'TENANT_MEMBER';
-- Drop "TenantSubscription" table
DROP TABLE "TenantSubscription";
-- Drop enum type "TenantSubscriptionStatus"
DROP TYPE "TenantSubscriptionStatus";
-- Drop enum type "TenantSubscriptionPeriod"
DROP TYPE "TenantSubscriptionPeriod";
-- Drop enum type "TenantSubscriptionPlanCodes"
DROP TYPE "TenantSubscriptionPlanCodes";
