-- +goose Up
-- CreateEnum
CREATE TYPE "TenantMemberRole" AS ENUM ('OWNER', 'ADMIN', 'MEMBER');

-- CreateEnum
CREATE TYPE "WorkflowRunStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED');

-- CreateEnum
CREATE TYPE "JobRunStatus" AS ENUM ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED');

-- CreateEnum
CREATE TYPE "StepRunStatus" AS ENUM ('PENDING', 'PENDING_ASSIGNMENT', 'ASSIGNED', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED');

-- CreateEnum
CREATE TYPE "WorkerStatus" AS ENUM ('ACTIVE', 'INACTIVE');

-- CreateTable
CREATE TABLE "User" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "email" TEXT NOT NULL,
    "emailVerified" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT,

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "UserPassword" (
    "hash" TEXT NOT NULL,
    "userId" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "UserSession" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID,
    "data" JSONB,
    "expiresAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "UserSession_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Tenant" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "name" TEXT NOT NULL,
    "slug" TEXT NOT NULL,

    CONSTRAINT "Tenant_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantMember" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "role" "TenantMemberRole" NOT NULL,

    CONSTRAINT "TenantMember_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Event" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "key" TEXT NOT NULL,
    "tenantId" UUID NOT NULL,
    "replayedFromId" UUID,
    "data" JSONB,

    CONSTRAINT "Event_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowTag" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "color" TEXT NOT NULL DEFAULT '#93C5FD',

    CONSTRAINT "WorkflowTag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Workflow" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,

    CONSTRAINT "Workflow_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowVersion" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "version" TEXT NOT NULL,
    "order" SMALLSERIAL NOT NULL,
    "workflowId" UUID NOT NULL,

    CONSTRAINT "WorkflowVersion_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowTriggers" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "workflowVersionId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "WorkflowTriggers_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowTriggerEventRef" (
    "parentId" UUID NOT NULL,
    "eventKey" TEXT NOT NULL
);

-- CreateTable
CREATE TABLE "WorkflowTriggerCronRef" (
    "parentId" UUID NOT NULL,
    "cron" TEXT NOT NULL,
    "tickerId" UUID
);

-- CreateTable
CREATE TABLE "WorkflowTriggerScheduledRef" (
    "id" UUID NOT NULL,
    "parentId" UUID NOT NULL,
    "triggerAt" TIMESTAMP(3) NOT NULL,
    "tickerId" UUID,
    "input" JSONB,

    CONSTRAINT "WorkflowTriggerScheduledRef_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Job" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowVersionId" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "timeout" TEXT,

    CONSTRAINT "Job_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Action" (
    "id" TEXT NOT NULL,
    "description" TEXT,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "Action_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Step" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "readableId" TEXT,
    "tenantId" UUID NOT NULL,
    "jobId" UUID NOT NULL,
    "actionId" TEXT NOT NULL,
    "timeout" TEXT,

    CONSTRAINT "Step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowRun" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowVersionId" UUID NOT NULL,
    "status" "WorkflowRunStatus" NOT NULL DEFAULT 'PENDING',
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),

    CONSTRAINT "WorkflowRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkflowRunTriggeredBy" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "parentId" TEXT NOT NULL,
    "eventId" UUID,
    "cronParentId" UUID,
    "cronSchedule" TEXT,
    "scheduledId" UUID,

    CONSTRAINT "WorkflowRunTriggeredBy_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "JobRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "workflowRunId" TEXT NOT NULL,
    "jobId" UUID NOT NULL,
    "tickerId" UUID,
    "status" "JobRunStatus" NOT NULL DEFAULT 'PENDING',
    "result" JSONB,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,

    CONSTRAINT "JobRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "JobRunLookupData" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "jobRunId" UUID NOT NULL,
    "tenantId" UUID NOT NULL,
    "data" JSONB,

    CONSTRAINT "JobRunLookupData_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "StepRun" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "jobRunId" UUID NOT NULL,
    "stepId" UUID NOT NULL,
    "order" SMALLSERIAL NOT NULL,
    "workerId" UUID,
    "tickerId" UUID,
    "status" "StepRunStatus" NOT NULL DEFAULT 'PENDING',
    "input" JSONB,
    "output" JSONB,
    "requeueAfter" TIMESTAMP(3),
    "scheduleTimeoutAt" TIMESTAMP(3),
    "error" TEXT,
    "startedAt" TIMESTAMP(3),
    "finishedAt" TIMESTAMP(3),
    "timeoutAt" TIMESTAMP(3),
    "cancelledAt" TIMESTAMP(3),
    "cancelledReason" TEXT,
    "cancelledError" TEXT,

    CONSTRAINT "StepRun_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Dispatcher" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "lastHeartbeatAt" TIMESTAMP(3),
    "isActive" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "Dispatcher_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Ticker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeatAt" TIMESTAMP(3),
    "isActive" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "Ticker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Worker" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "tenantId" UUID NOT NULL,
    "lastHeartbeatAt" TIMESTAMP(3),
    "name" TEXT NOT NULL,
    "status" "WorkerStatus" NOT NULL DEFAULT 'ACTIVE',
    "dispatcherId" UUID NOT NULL,

    CONSTRAINT "Worker_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "Service" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deletedAt" TIMESTAMP(3),
    "name" TEXT NOT NULL,
    "description" TEXT,
    "tenantId" UUID NOT NULL,

    CONSTRAINT "Service_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "_WorkflowToWorkflowTag" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_ActionToWorker" (
    "A" TEXT NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_StepOrder" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_StepRunOrder" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_ServiceToWorker" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "User_id_key" ON "User"("id");

-- CreateIndex
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- CreateIndex
CREATE UNIQUE INDEX "UserPassword_userId_key" ON "UserPassword"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "UserSession_id_key" ON "UserSession"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Tenant_id_key" ON "Tenant"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Tenant_slug_key" ON "Tenant"("slug");

-- CreateIndex
CREATE UNIQUE INDEX "TenantMember_id_key" ON "TenantMember"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantMember_tenantId_userId_key" ON "TenantMember"("tenantId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "Event_id_key" ON "Event"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTag_id_key" ON "WorkflowTag"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTag_tenantId_name_key" ON "WorkflowTag"("tenantId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "Workflow_id_key" ON "Workflow"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Workflow_tenantId_name_key" ON "Workflow"("tenantId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowVersion_id_key" ON "WorkflowVersion"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowVersion_workflowId_version_key" ON "WorkflowVersion"("workflowId", "version");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggers_id_key" ON "WorkflowTriggers"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggers_workflowVersionId_key" ON "WorkflowTriggers"("workflowVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerEventRef_parentId_eventKey_key" ON "WorkflowTriggerEventRef"("parentId", "eventKey");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerCronRef_parentId_cron_key" ON "WorkflowTriggerCronRef"("parentId", "cron");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowTriggerScheduledRef_id_key" ON "WorkflowTriggerScheduledRef"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Job_id_key" ON "Job"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Job_workflowVersionId_name_key" ON "Job"("workflowVersionId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "Action_tenantId_id_key" ON "Action"("tenantId", "id");

-- CreateIndex
CREATE UNIQUE INDEX "Step_id_key" ON "Step"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Step_jobId_readableId_key" ON "Step"("jobId", "readableId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRun_tenantId_id_key" ON "WorkflowRun"("tenantId", "id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_id_key" ON "WorkflowRunTriggeredBy"("id");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_parentId_key" ON "WorkflowRunTriggeredBy"("parentId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_scheduledId_key" ON "WorkflowRunTriggeredBy"("scheduledId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkflowRunTriggeredBy_tenantId_parentId_key" ON "WorkflowRunTriggeredBy"("tenantId", "parentId");

-- CreateIndex
CREATE UNIQUE INDEX "JobRun_id_key" ON "JobRun"("id");

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_id_key" ON "JobRunLookupData"("id");

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_jobRunId_key" ON "JobRunLookupData"("jobRunId");

-- CreateIndex
CREATE UNIQUE INDEX "JobRunLookupData_jobRunId_tenantId_key" ON "JobRunLookupData"("jobRunId", "tenantId");

-- CreateIndex
CREATE UNIQUE INDEX "StepRun_id_key" ON "StepRun"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Dispatcher_id_key" ON "Dispatcher"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Ticker_id_key" ON "Ticker"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Worker_id_key" ON "Worker"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Service_id_key" ON "Service"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Service_tenantId_name_key" ON "Service"("tenantId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "_WorkflowToWorkflowTag_AB_unique" ON "_WorkflowToWorkflowTag"("A", "B");

-- CreateIndex
CREATE INDEX "_WorkflowToWorkflowTag_B_index" ON "_WorkflowToWorkflowTag"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_ActionToWorker_AB_unique" ON "_ActionToWorker"("A", "B");

-- CreateIndex
CREATE INDEX "_ActionToWorker_B_index" ON "_ActionToWorker"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_StepOrder_AB_unique" ON "_StepOrder"("A", "B");

-- CreateIndex
CREATE INDEX "_StepOrder_B_index" ON "_StepOrder"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_StepRunOrder_AB_unique" ON "_StepRunOrder"("A", "B");

-- CreateIndex
CREATE INDEX "_StepRunOrder_B_index" ON "_StepRunOrder"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_ServiceToWorker_AB_unique" ON "_ServiceToWorker"("A", "B");

-- CreateIndex
CREATE INDEX "_ServiceToWorker_B_index" ON "_ServiceToWorker"("B");

-- AddForeignKey
ALTER TABLE "UserPassword" ADD CONSTRAINT "UserPassword_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "UserSession" ADD CONSTRAINT "UserSession_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantMember" ADD CONSTRAINT "TenantMember_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "TenantMember" ADD CONSTRAINT "TenantMember_userId_fkey" FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Event" ADD CONSTRAINT "Event_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Event" ADD CONSTRAINT "Event_replayedFromId_fkey" FOREIGN KEY ("replayedFromId") REFERENCES "Event"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTag" ADD CONSTRAINT "WorkflowTag_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Workflow" ADD CONSTRAINT "Workflow_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowVersion" ADD CONSTRAINT "WorkflowVersion_workflowId_fkey" FOREIGN KEY ("workflowId") REFERENCES "Workflow"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggers" ADD CONSTRAINT "WorkflowTriggers_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggers" ADD CONSTRAINT "WorkflowTriggers_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerEventRef" ADD CONSTRAINT "WorkflowTriggerEventRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowTriggers"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowTriggers"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowVersion"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Job" ADD CONSTRAINT "Job_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Job" ADD CONSTRAINT "Job_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Action" ADD CONSTRAINT "Action_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_jobId_fkey" FOREIGN KEY ("jobId") REFERENCES "Job"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_actionId_tenantId_fkey" FOREIGN KEY ("actionId", "tenantId") REFERENCES "Action"("id", "tenantId") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRun" ADD CONSTRAINT "WorkflowRun_workflowVersionId_fkey" FOREIGN KEY ("workflowVersionId") REFERENCES "WorkflowVersion"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_eventId_fkey" FOREIGN KEY ("eventId") REFERENCES "Event"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_fkey" FOREIGN KEY ("cronParentId", "cronSchedule") REFERENCES "WorkflowTriggerCronRef"("parentId", "cron") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_scheduledId_fkey" FOREIGN KEY ("scheduledId") REFERENCES "WorkflowTriggerScheduledRef"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_jobId_fkey" FOREIGN KEY ("jobId") REFERENCES "Job"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRun" ADD CONSTRAINT "JobRun_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRunLookupData" ADD CONSTRAINT "JobRunLookupData_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "JobRunLookupData" ADD CONSTRAINT "JobRunLookupData_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "StepRun" ADD CONSTRAINT "StepRun_tickerId_fkey" FOREIGN KEY ("tickerId") REFERENCES "Ticker"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_dispatcherId_fkey" FOREIGN KEY ("dispatcherId") REFERENCES "Dispatcher"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Service" ADD CONSTRAINT "Service_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WorkflowToWorkflowTag" ADD CONSTRAINT "_WorkflowToWorkflowTag_A_fkey" FOREIGN KEY ("A") REFERENCES "Workflow"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_WorkflowToWorkflowTag" ADD CONSTRAINT "_WorkflowToWorkflowTag_B_fkey" FOREIGN KEY ("B") REFERENCES "WorkflowTag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ActionToWorker" ADD CONSTRAINT "_ActionToWorker_A_fkey" FOREIGN KEY ("A") REFERENCES "Action"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ActionToWorker" ADD CONSTRAINT "_ActionToWorker_B_fkey" FOREIGN KEY ("B") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepOrder" ADD CONSTRAINT "_StepOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepOrder" ADD CONSTRAINT "_StepOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_A_fkey" FOREIGN KEY ("A") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_StepRunOrder" ADD CONSTRAINT "_StepRunOrder_B_fkey" FOREIGN KEY ("B") REFERENCES "StepRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ServiceToWorker" ADD CONSTRAINT "_ServiceToWorker_A_fkey" FOREIGN KEY ("A") REFERENCES "Service"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ServiceToWorker" ADD CONSTRAINT "_ServiceToWorker_B_fkey" FOREIGN KEY ("B") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
