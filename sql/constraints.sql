-- NOTE: this is a SQL script file that contains the constraints for the database
-- it is needed because prisma does not support constraints yet

-- Modify "QueueItem" table
ALTER TABLE "QueueItem" ADD CONSTRAINT "QueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);

-- Modify "InternalQueueItem" table
ALTER TABLE "InternalQueueItem" ADD CONSTRAINT "InternalQueueItem_priority_check" CHECK ("priority" >= 1 AND "priority" <= 4);
