-- +goose Up
-- +goose StatementBegin
ALTER TABLE v1_task ADD COLUMN max_reassignments_allowed INTEGER;
ALTER TABLE "Step" ADD COLUMN "maxReassignmentsAllowed" INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE v1_task DROP COLUMN max_reassignments_allowed;
ALTER TABLE "Step" DROP COLUMN "maxReassignmentsAllowed";
-- +goose StatementEnd
