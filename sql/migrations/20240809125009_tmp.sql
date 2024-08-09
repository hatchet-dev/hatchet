-- Modify "QueueItem" table
ALTER TABLE "QueueItem" ADD COLUMN "desiredWorkerId" uuid NULL, ADD COLUMN "sticky" "StickyStrategy" NULL;
