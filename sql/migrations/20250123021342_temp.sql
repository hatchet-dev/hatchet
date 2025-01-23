-- Create enum type "WorkflowRunProcessingState"
CREATE TYPE "WorkflowRunProcessingState" AS ENUM ('WAITING', 'PROCESSING', 'DONE', 'ERROR');

-- Modify "WorkflowRun" table
ALTER TABLE "WorkflowRun"
ADD COLUMN "processingState" "WorkflowRunProcessingState" NOT NULL DEFAULT 'WAITING';

-- TODO add an index on processingState