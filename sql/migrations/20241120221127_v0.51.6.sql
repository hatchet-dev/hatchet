-- Modify "Event" table
ALTER TABLE "Event" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "JobRun" table
ALTER TABLE "JobRun" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "JobRunLookupData" table
ALTER TABLE "JobRunLookupData" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
-- Modify "WorkflowRunTriggeredBy" table
ALTER TABLE "WorkflowRunTriggeredBy" ADD COLUMN "identityId" bigint NOT NULL GENERATED ALWAYS AS IDENTITY;
