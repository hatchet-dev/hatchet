-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "alertMemberEmails" BOOLEAN NOT NULL DEFAULT true;

-- AlterTable
ALTER TABLE "TenantAlertingSettings" ADD COLUMN     "enableExpiringTokenAlerts" BOOLEAN NOT NULL DEFAULT true,
ADD COLUMN     "enableWorkflowRunFailureAlerts" BOOLEAN NOT NULL DEFAULT false;
