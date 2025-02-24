-- Add value to enum type: "ConcurrencyLimitStrategy"
ALTER TYPE "ConcurrencyLimitStrategy" ADD VALUE 'CANCEL_NEWEST';
-- Add value to enum type: "WorkflowRunStatus"
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'CANCELLING';
-- Add value to enum type: "WorkflowRunStatus"
ALTER TYPE "WorkflowRunStatus" ADD VALUE 'CANCELLED';
-- Create enum type "MessageQueueItemStatus"
CREATE TYPE "MessageQueueItemStatus" AS ENUM('PENDING', 'ASSIGNED');

-- Create "MessageQueue" table
CREATE TABLE
    "MessageQueue" (
        "name" text NOT NULL,
        "lastActive" TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP,
        "durable" boolean NOT NULL DEFAULT true,
        "autoDeleted" boolean NOT NULL DEFAULT false,
        "exclusive" boolean NOT NULL DEFAULT false,
        "exclusiveConsumerId" uuid NULL,
        PRIMARY KEY ("name")
    );

-- Create "MessageQueueItem" table
CREATE TABLE
    "MessageQueueItem" (
        "id" bigint NOT NULL GENERATED ALWAYS AS IDENTITY,
        "payload" jsonb NOT NULL,
        "readAfter" timestamp(3) NULL,
        "expiresAt" timestamp(3) NULL,
        "queueId" text,
        "status" "MessageQueueItemStatus" NOT NULL DEFAULT 'PENDING',
        PRIMARY KEY ("id"),
        CONSTRAINT "MessageQueueItem_queueId_fkey" FOREIGN KEY ("queueId") REFERENCES "MessageQueue" ("name") ON UPDATE NO ACTION ON DELETE SET NULL
    );

-- Create index "MessageQueueItem_queueId_expiresAt_readAfter_status_id_idx" to table: "MessageQueueItem"
CREATE INDEX "MessageQueueItem_queueId_expiresAt_readAfter_status_id_idx" ON "MessageQueueItem" (
    "expiresAt",
    "queueId",
    "readAfter",
    "status",
    "id"
);

-- Function to publish NOTIFY message on insert into MessageQueueItem
CREATE
OR REPLACE FUNCTION notify_message_queue_item () RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        NEW."queueId"::TEXT,
        NEW."id"::TEXT
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to invoke the notify function after insert
CREATE TRIGGER trigger_notify_message_queue_item
AFTER INSERT ON "MessageQueueItem" FOR EACH ROW
EXECUTE FUNCTION notify_message_queue_item ();

-- Update the existing function to prevent internal name or slug to be a no-op
CREATE
OR REPLACE FUNCTION prevent_internal_name_or_slug () RETURNS trigger AS $$
BEGIN
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
