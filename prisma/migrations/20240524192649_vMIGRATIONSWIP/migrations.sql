-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "webhook" BOOLEAN NOT NULL DEFAULT false;

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "webhook" TEXT;

-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "webhookSecret" TEXT;
