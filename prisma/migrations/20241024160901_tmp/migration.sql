-- AlterTable
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN     "additionalMetadata" JSONB;

-- AlterTable
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN     "additionalMetadata" JSONB;
