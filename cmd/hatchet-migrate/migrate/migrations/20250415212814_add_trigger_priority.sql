-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowTriggerScheduledRef" ADD COLUMN "priority" INTEGER NOT NULL DEFAULT 1;
ALTER TABLE "WorkflowTriggerScheduledRef" ADD CONSTRAINT "WorkflowTriggerScheduledRef_priority_check" CHECK (
    priority >= 1 AND priority <= 4
);

ALTER TABLE "WorkflowTriggerCronRef" ADD COLUMN "priority" INTEGER NOT NULL DEFAULT 1;
ALTER TABLE "WorkflowTriggerCronRef" ADD CONSTRAINT "WorkflowTriggerCronRef_priority_check" CHECK (
    priority >= 1 AND priority <= 4
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowTriggerScheduledRef" DROP CONSTRAINT "WorkflowTriggerScheduledRef_priority_check";
ALTER TABLE "WorkflowTriggerScheduledRef" DROP COLUMN "priority";

ALTER TABLE "WorkflowTriggerCronRef" DROP CONSTRAINT "WorkflowTriggerCronRef_priority_check";
ALTER TABLE "WorkflowTriggerCronRef" DROP COLUMN "priority";
-- +goose StatementEnd
