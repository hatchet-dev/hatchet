-- Modify "WorkflowTriggerCronRef" table
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "deletedAt" timestamp(3) NULL, ADD COLUMN "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- Modify "WorkflowTriggerScheduledRef" table
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, ADD COLUMN "deletedAt" timestamp(3) NULL, ADD COLUMN "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP;
