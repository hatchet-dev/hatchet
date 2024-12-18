-- Add value to enum type: "ConcurrencyLimitStrategy"
ALTER TYPE "ConcurrencyLimitStrategy" ADD VALUE 'CANCEL_NEWEST';
-- Add value to enum type: "WorkflowRunStatus"
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'CANCELLING';
-- Add value to enum type: "WorkflowRunStatus"
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'CANCELLED';