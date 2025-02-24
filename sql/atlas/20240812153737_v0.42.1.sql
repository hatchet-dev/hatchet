-- atlas:txmode none

ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);

DROP INDEX CONCURRENTLY IF EXISTS "QueueItem_isQueued_priority_tenantId_queue_id_idx";

CREATE INDEX CONCURRENTLY IF NOT EXISTS "QueueItem_isQueued_priority_tenantId_queue_id_idx_2" ON "QueueItem" ("isQueued", "tenantId", "queue", "priority" DESC, "id");
