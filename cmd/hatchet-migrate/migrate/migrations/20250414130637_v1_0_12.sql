-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowRunTriggeredBy" DROP CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowRunTriggeredBy" ADD CONSTRAINT "WorkflowRunTriggeredBy_cronParentId_cronSchedule_cronName_fkey" FOREIGN KEY ("cronParentId", "cronSchedule", "cronName") REFERENCES "WorkflowTriggerCronRef"("parentId", "cron", "name") ON DELETE SET NULL ON UPDATE CASCADE;
-- +goose StatementEnd
