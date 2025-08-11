-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- STEP 1: Update existing priority values > 3 to 3 in all relevant tables
-- ============================================================================

-- Update v1_task table (user-facing, from SDK)
UPDATE v1_task 
SET priority = 3 
WHERE priority > 3;

-- Update v1_queue_item table (derived from v1_task via triggers)
UPDATE v1_queue_item 
SET priority = 3 
WHERE priority > 3;

-- Update v1_workflow_concurrency_slot table (workflow-level concurrency)
UPDATE v1_workflow_concurrency_slot 
SET priority = 3 
WHERE priority > 3;

-- Update v1_concurrency_slot table (step-level concurrency)
UPDATE v1_concurrency_slot 
SET priority = 3 
WHERE priority > 3;

-- ============================================================================
-- STEP 2: Add check constraints to prevent users from setting priority > 3
-- ============================================================================

-- Add constraint to v1_task (primary user-facing table)
ALTER TABLE v1_task 
ADD CONSTRAINT v1_task_priority_user_limit 
CHECK (priority >= 1 AND priority <= 3);

-- Add constraint to v1_workflow_concurrency_slot (workflow-level)
ALTER TABLE v1_workflow_concurrency_slot 
ADD CONSTRAINT v1_workflow_concurrency_slot_priority_user_limit 
CHECK (priority >= 1 AND priority <= 3);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Remove the check constraints
ALTER TABLE v1_task DROP CONSTRAINT IF EXISTS v1_task_priority_user_limit;
ALTER TABLE v1_workflow_concurrency_slot DROP CONSTRAINT IF EXISTS v1_workflow_concurrency_slot_priority_user_limit;

-- Remove comments
COMMENT ON COLUMN v1_task.priority IS NULL;
COMMENT ON COLUMN v1_queue_item.priority IS NULL;
COMMENT ON COLUMN v1_workflow_concurrency_slot.priority IS NULL;
COMMENT ON COLUMN v1_concurrency_slot.priority IS NULL;

-- +goose StatementEnd
