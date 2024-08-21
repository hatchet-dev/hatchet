-- DropIndex
DROP INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx_2";

-- CreateIndex
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx" ON "QueueItem"("isQueued", "priority" DESC, "tenantId", "queue", "id");
