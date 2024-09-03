-- NOTE: this is a SQL script file that contains the constraints for the database
-- it is needed because prisma does not support constraints yet

-- Modify "QueueItem" table
ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);



-- CREATE TABLE "WorkflowRunEventView" (
--     "minute" TIMESTAMPTZ NOT NULL,
--     "tenantId" UUID NOT NULL,
--     "pending_count" INT NOT NULL,
--     "queued_count" INT NOT NULL,
--     "running_count" INT NOT NULL,
--     "succeeded_count" INT NOT NULL,
--     "retried_count" INT NOT NULL,
--     "failed_count" INT NOT NULL,
--     "queue_depth" INT NOT NULL,
--     PRIMARY KEY ("minute", "tenantId")
-- );


-- SELECT * from create_hypertable('"WorkflowRunEvent"', by_range('createdAt',  INTERVAL '1 minute'));
