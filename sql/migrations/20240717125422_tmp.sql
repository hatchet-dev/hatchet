-- Create enum type "StickyStrategy"
CREATE TYPE "StickyStrategy" AS ENUM ('SOFT', 'HARD');
-- Modify "WorkflowVersion" table
ALTER TABLE "WorkflowVersion" ADD COLUMN "sticky" "StickyStrategy" NULL;
-- Create "WorkflowRunStickyState" table
CREATE TABLE "WorkflowRunStickyState" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "tenantId" uuid NOT NULL, "workflowRunId" uuid NOT NULL, "desiredWorkerId" uuid NULL, "strategy" "StickyStrategy" NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "WorkflowRunStickyState_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "WorkflowRunStickyState_workflowRunId_key" to table: "WorkflowRunStickyState"
CREATE UNIQUE INDEX "WorkflowRunStickyState_workflowRunId_key" ON "WorkflowRunStickyState" ("workflowRunId");
