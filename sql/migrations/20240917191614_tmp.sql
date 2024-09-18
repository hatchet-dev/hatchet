-- Drop index "StepRun_id_tenantId_idx" from table: "StepRun"
DROP INDEX "StepRun_id_tenantId_idx";
-- Modify "StepRunEvent" table
ALTER TABLE "StepRunEvent" ALTER COLUMN "stepRunId" SET NOT NULL;
-- Create index "StepRunEvent_stepRunId_idx" to table: "StepRunEvent"
CREATE INDEX "StepRunEvent_stepRunId_idx" ON "StepRunEvent" ("stepRunId");
-- Modify "StepRunResultArchive" table
ALTER TABLE "StepRunResultArchive" ALTER COLUMN "stepRunId" SET NOT NULL;
-- Modify "TimeoutQueueItem" table
ALTER TABLE "TimeoutQueueItem" ALTER COLUMN "stepRunId" SET NOT NULL;
-- Create index "TimeoutQueueItem_stepRunId_retryCount_key" to table: "TimeoutQueueItem"
CREATE UNIQUE INDEX "TimeoutQueueItem_stepRunId_retryCount_key" ON "TimeoutQueueItem" ("stepRunId", "retryCount");
-- Create index "WorkflowRun_parentId_parentStepRunId_childKey_key" to table: "WorkflowRun"
CREATE UNIQUE INDEX "WorkflowRun_parentId_parentStepRunId_childKey_key" ON "WorkflowRun" ("parentId", "parentStepRunId", "childKey");
-- Create index "WorkflowTriggerScheduledRef_parentId_parentStepRunId_childK_key" to table: "WorkflowTriggerScheduledRef"
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_parentId_parentStepRunId_childK_key" ON "WorkflowTriggerScheduledRef" ("parentId", "parentStepRunId", "childKey");
-- Modify "LogLine" table
ALTER TABLE "LogLine" ADD CONSTRAINT "LogLine_stepRunId_fkey" FOREIGN KEY ("stepRunId") REFERENCES "StepRun" ("id") ON UPDATE CASCADE ON DELETE SET NULL;
-- Modify "_StepRunOrder" table
ALTER TABLE "_StepRunOrder" ALTER COLUMN "A" SET NOT NULL, ALTER COLUMN "B" SET NOT NULL, ADD CONSTRAINT "_StepRunOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "StepRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE, ADD CONSTRAINT "_StepRunOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "StepRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE;
-- Create index "_StepRunOrder_AB_unique" to table: "_StepRunOrder"
CREATE UNIQUE INDEX "_StepRunOrder_AB_unique" ON "_StepRunOrder" ("A", "B");
-- Create index "_StepRunOrder_B_index" to table: "_StepRunOrder"
CREATE INDEX "_StepRunOrder_B_index" ON "_StepRunOrder" ("B");
