-- Modify "WorkflowTriggerCronRef" table
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "name" text NULL, ADD COLUMN "id" uuid NOT NULL, ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_cron_name_key" UNIQUE ("parentId", "cron", "name");
-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey", ADD COLUMN "cronName" text NULL, ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") REFERENCES "WorkflowTriggerCronRef" ("parentId", "cron", "name") ON UPDATE CASCADE ON DELETE SET NULL;
