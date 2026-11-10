-- +goose NO TRANSACTION

-- +goose Up
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "WorkflowTriggerCronRef_pkey"
    ON "WorkflowTriggerCronRef" ("id");

ALTER TABLE "WorkflowTriggerCronRef"
    ADD CONSTRAINT "WorkflowTriggerCronRef_pkey"
    PRIMARY KEY USING INDEX "WorkflowTriggerCronRef_pkey";

-- +goose Down
ALTER TABLE "WorkflowTriggerCronRef" DROP CONSTRAINT IF EXISTS "WorkflowTriggerCronRef_pkey";

DROP INDEX CONCURRENTLY IF EXISTS "WorkflowTriggerCronRef_pkey";
