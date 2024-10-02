-- DropForeignKey
ALTER TABLE "Event" DROP CONSTRAINT "Event_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "GetGroupKeyRun" DROP CONSTRAINT "GetGroupKeyRun_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "Job" DROP CONSTRAINT "Job_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "JobRun" DROP CONSTRAINT "JobRun_tickerId_fkey";

-- DropForeignKey
ALTER TABLE "LogLine" DROP CONSTRAINT "LogLine_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "RateLimit" DROP CONSTRAINT "RateLimit_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "Step" DROP CONSTRAINT "Step_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "StepRateLimit" DROP CONSTRAINT "StepRateLimit_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_stepId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "StepRun" DROP CONSTRAINT "StepRun_tickerId_fkey";

-- DropForeignKey
ALTER TABLE "StreamEvent" DROP CONSTRAINT "StreamEvent_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "Worker" DROP CONSTRAINT "Worker_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "Workflow" DROP CONSTRAINT "Workflow_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRun" DROP CONSTRAINT "WorkflowRun_workflowVersionId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRunDedupe" DROP CONSTRAINT "WorkflowRunDedupe_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_eventId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_parentId_fkey";

-- DropForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_tenantId_fkey";

-- CreateIndex
CREATE INDEX "StepRunExpressionEval_stepRunId_idx" ON "StepRunExpressionEval"("stepRunId");
