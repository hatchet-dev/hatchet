-- +goose Up
-- +goose StatementBegin
ALTER TABLE "WorkflowTriggerEventRef"
ADD COLUMN expression TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "WorkflowTriggerEventRef"
DROP COLUMN expression;
-- +goose StatementEnd
