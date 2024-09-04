-- Modify "StepRunEvent" table
ALTER TABLE "StepRunEvent" ADD COLUMN "jobRunId" uuid NULL, ADD CONSTRAINT "StepRunEvent_jobRunId_fkey" FOREIGN KEY ("jobRunId") REFERENCES "JobRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE;
