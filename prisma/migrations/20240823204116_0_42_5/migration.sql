-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "priority" INTEGER;

-- AlterTable
ALTER TABLE "WorkflowRun" ADD COLUMN     "priority" INTEGER;

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN     "defaultPriority" INTEGER;
