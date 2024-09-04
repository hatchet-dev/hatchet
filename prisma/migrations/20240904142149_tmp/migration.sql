-- AlterTable
ALTER TABLE "StepRunEvent" ADD COLUMN     "jobRunId" UUID;

-- AddForeignKey
ALTER TABLE "StepRunEvent" ADD CONSTRAINT "StepRunEvent_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun"("id") ON DELETE CASCADE ON UPDATE CASCADE;
