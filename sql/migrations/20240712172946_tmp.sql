-- Create index "GetGroupKeyRun_deletedAt_idx" to table: "GetGroupKeyRun"
CREATE INDEX "GetGroupKeyRun_deletedAt_idx" ON "GetGroupKeyRun" ("deletedAt");
-- Create index "JobRun_deletedAt_idx" to table: "JobRun"
CREATE INDEX "JobRun_deletedAt_idx" ON "JobRun" ("deletedAt");
-- Create index "StepRun_deletedAt_idx" to table: "StepRun"
CREATE INDEX "StepRun_deletedAt_idx" ON "StepRun" ("deletedAt");
-- Create index "WorkflowVersion_deletedAt_idx" to table: "WorkflowVersion"
CREATE INDEX "WorkflowVersion_deletedAt_idx" ON "WorkflowVersion" ("deletedAt");
