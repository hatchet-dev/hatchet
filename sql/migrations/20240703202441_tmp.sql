-- Modify "Step" table
ALTER TABLE "Step" DROP COLUMN "desiredWorkerAffinity";
-- Create "StepDesiredWorkerLabel" table
CREATE TABLE "StepDesiredWorkerLabel" ("id" bigserial NOT NULL, "createdAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "updatedAt" timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "stepId" uuid NOT NULL, "key" text NOT NULL, "strValue" text NULL, "intValue" integer NULL, "required" boolean NOT NULL, "comparator" "WorkerLabelComparator" NOT NULL, "weight" integer NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step" ("id") ON UPDATE CASCADE ON DELETE CASCADE);
-- Create index "StepDesiredWorkerLabel_stepId_idx" to table: "StepDesiredWorkerLabel"
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel" ("stepId");
-- Create index "StepDesiredWorkerLabel_stepId_key_key" to table: "StepDesiredWorkerLabel"
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel" ("stepId", "key");
