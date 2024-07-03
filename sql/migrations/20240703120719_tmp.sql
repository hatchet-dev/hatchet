-- Modify "WorkerAffinity" table
ALTER TABLE "WorkerAffinity" ADD COLUMN "required" boolean NOT NULL DEFAULT false;
-- Create index "WorkerAffinity_workerId_idx" to table: "WorkerAffinity"
CREATE INDEX "WorkerAffinity_workerId_idx" ON "WorkerAffinity" ("workerId");
