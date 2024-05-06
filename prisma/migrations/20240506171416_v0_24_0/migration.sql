-- AlterTable
ALTER TABLE "Event" ADD COLUMN     "additionalMetadata" JSONB;

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "additionalMetadata" JSONB;
