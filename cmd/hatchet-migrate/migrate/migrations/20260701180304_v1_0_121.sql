-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY IF NOT EXISTS "StepExpression_stepId_idx" ON "StepExpression" ("stepId");
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS "StepExpression_stepId_idx";
-- +goose StatementEnd
