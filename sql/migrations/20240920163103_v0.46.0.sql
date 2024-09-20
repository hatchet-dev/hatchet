-- Create enum type "StepExpressionKind"
CREATE TYPE "StepExpressionKind" AS ENUM ('DYNAMIC_RATE_LIMIT_KEY', 'DYNAMIC_RATE_LIMIT_VALUE', 'DYNAMIC_RATE_LIMIT_UNITS');
-- Create enum type "StepRateLimitKind"
CREATE TYPE "StepRateLimitKind" AS ENUM ('DEFAULT', 'DYNAMIC');
-- Modify "StepRateLimit" table
ALTER TABLE "StepRateLimit" DROP CONSTRAINT "StepRateLimit_stepId_fkey", ADD COLUMN "kind" "StepRateLimitKind" NOT NULL DEFAULT 'DEFAULT';
-- Create index "idx_workflowrun_concurrency" to table: "WorkflowRun"
CREATE INDEX "idx_workflowrun_concurrency" ON "WorkflowRun" ("concurrencyGroupId", "createdAt");
-- Create index "idx_workflowrun_main" to table: "WorkflowRun"
CREATE INDEX "idx_workflowrun_main" ON "WorkflowRun" ("tenantId", "deletedAt", "status", "workflowVersionId", "createdAt");
-- Create "StepExpression" table
CREATE TABLE "StepExpression" ("key" text NOT NULL, "stepId" uuid NOT NULL, "expression" text NOT NULL, "kind" "StepExpressionKind" NOT NULL, PRIMARY KEY ("key", "stepId"));
-- Create "StepRunExpressionEval" table
CREATE TABLE "StepRunExpressionEval" ("key" text NOT NULL, "stepRunId" uuid NOT NULL, "valueStr" text NULL, "valueInt" integer NULL, "kind" "StepExpressionKind" NOT NULL, PRIMARY KEY ("key", "stepRunId"));
