-- Create enum type "WorkflowKind"
CREATE TYPE "WorkflowKind" AS ENUM ('FUNCTION', 'DURABLE', 'DAG');
-- Modify "WorkflowVersion" table
ALTER TABLE "WorkflowVersion" ADD COLUMN "kind" "WorkflowKind" NOT NULL DEFAULT 'DAG';
