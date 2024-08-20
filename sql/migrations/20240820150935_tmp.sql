-- Create enum type "WorkerType"
CREATE TYPE "WorkerType" AS ENUM ('WEBHOOK', 'MANAGED', 'SELFHOSTED');
-- Create index "Action_actionId_tenantId_key" to table: "Action"
CREATE UNIQUE INDEX "Action_actionId_tenantId_key" ON "Action" ("actionId", "tenantId");
-- Drop index "QueueItem_isQueued_priority_tenantId_queue_id_idx_2" from table: "QueueItem"
DROP INDEX IF EXISTS "QueueItem_isQueued_priority_tenantId_queue_id_idx_2";
-- Modify "QueueItem" table
ALTER TABLE "QueueItem" DROP CONSTRAINT IF EXISTS "QueueItem_priority_check";
-- Create index "QueueItem_isQueued_priority_tenantId_queue_id_idx" to table: "QueueItem"
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx" ON "QueueItem" ("isQueued", "priority", "tenantId", "queue", "id");
-- Modify "Worker" table
ALTER TABLE "Worker" ADD COLUMN "type" "WorkerType" NOT NULL DEFAULT 'SELFHOSTED', ADD COLUMN "webhookId" uuid NULL, ADD CONSTRAINT "Worker_webhookId_fkey" FOREIGN KEY ("webhookId") REFERENCES "WebhookWorker" ("id") ON UPDATE CASCADE ON DELETE SET NULL;
-- Create index "Worker_webhookId_key" to table: "Worker"
CREATE UNIQUE INDEX "Worker_webhookId_key" ON "Worker" ("webhookId");
