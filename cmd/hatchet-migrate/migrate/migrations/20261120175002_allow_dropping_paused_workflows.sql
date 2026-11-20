-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Workflow" ADD COLUMN "pausedWorkflowQueueBehavior" "WorkflowPauseQueueBehavior";
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Workflow" DROP COLUMN "pausedWorkflowQueueBehavior";
-- +goose StatementEnd
