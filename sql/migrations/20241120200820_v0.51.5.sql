-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "identityId" bigserial NOT NULL, ADD CONSTRAINT "step_run_identity_id_unique" UNIQUE ("identityId", "status");
