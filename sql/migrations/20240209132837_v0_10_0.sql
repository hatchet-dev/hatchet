-- AlterTable
ALTER TABLE "Step" ADD COLUMN     "customUserData" JSONB;

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "inputSchema" JSONB;
