-- Drop index "QueueItem_isQueued_priority_tenantId_queue_id_idx_2" from table: "QueueItem"
DROP INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx_2";
-- Modify "QueueItem" table
ALTER TABLE "QueueItem" DROP CONSTRAINT "QueueItem_priority_check";
-- Create index "QueueItem_isQueued_priority_tenantId_queue_id_idx" to table: "QueueItem"
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx" ON "QueueItem" ("isQueued", "priority" DESC, "tenantId", "queue", "id");
