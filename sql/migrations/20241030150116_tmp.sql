-- Modify "WorkflowTriggerCronRef" table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "id" uuid NOT NULL DEFAULT uuid_generate_v4();
-- Create index "WorkflowTriggerCronRef_id_key" to table: "WorkflowTriggerCronRef"
CREATE UNIQUE INDEX "WorkflowTriggerCronRef_id_key" ON "WorkflowTriggerCronRef" ("id");
