-- Create enum type "WorkflowTriggerCronRefMethods"
CREATE TYPE "WorkflowTriggerCronRefMethods" AS ENUM ('DEFAULT', 'API');
-- Create enum type "WorkflowTriggerScheduledRefMethods"
CREATE TYPE "WorkflowTriggerScheduledRefMethods" AS ENUM ('DEFAULT', 'API');
-- Modify "WorkflowTriggerCronRef" table
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "name" text NULL, ADD COLUMN "id" uuid NOT NULL, ADD COLUMN "method" "WorkflowTriggerCronRefMethods" NOT NULL DEFAULT 'DEFAULT', ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_cron_name_key" UNIQUE ("parentId", "cron", "name");
-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "method" "WorkflowTriggerScheduledRefMethods" NOT NULL DEFAULT 'DEFAULT';
-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey", ADD COLUMN "cronName" text NULL, ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") REFERENCES "WorkflowTriggerCronRef" ("parentId", "cron", "name") ON UPDATE CASCADE ON DELETE SET NULL;
-- Drop index "WorkflowTriggerCronRef_parentId_cron_key" from table: "WorkflowTriggerCronRef"
DROP INDEX "WorkflowTriggerCronRef_parentId_cron_key";
