-- +goose Up
-- Add value to enum type: "StepRunStatus"
ALTER TYPE "StepRunStatus" ADD VALUE 'CANCELLING';
-- Add value to enum type: "StepRunEventReason"
ALTER TYPE "StepRunEventReason" ADD VALUE 'SENT_TO_WORKER';
-- Modify "StepRun" table
ALTER TABLE "StepRun" ADD COLUMN "queue" text NOT NULL DEFAULT 'default';
-- Create "Queue" table
CREATE TABLE "Queue" ("id" bigserial NOT NULL, "tenantId" uuid NOT NULL, "name" text NOT NULL, PRIMARY KEY ("id"));
-- Create index "Queue_tenantId_name_key" to table: "Queue"
CREATE UNIQUE INDEX "Queue_tenantId_name_key" ON "Queue" ("tenantId", "name");
-- Create "QueueItem" table
CREATE TABLE "QueueItem" ("id" bigserial NOT NULL, "stepRunId" uuid NULL, "stepId" uuid NULL, "actionId" text NULL, "scheduleTimeoutAt" timestamp(3) NULL, "stepTimeout" text NULL, "priority" integer NOT NULL DEFAULT 1, "isQueued" boolean NOT NULL, "tenantId" uuid NOT NULL, "queue" text NOT NULL, "sticky" "StickyStrategy" NULL, "desiredWorkerId" uuid NULL, PRIMARY KEY ("id"));
-- Create index "QueueItem_isQueued_priority_tenantId_queue_id_idx" to table: "QueueItem"
CREATE INDEX "QueueItem_isQueued_priority_tenantId_queue_id_idx" ON "QueueItem" ("isQueued", "priority", "tenantId", "queue", "id");

-- Create queues based on unique actions for all step runs in a pending assignment state
WITH steps AS (
    SELECT
        DISTINCT ON (s."id")
        s."actionId",
        s."tenantId"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    WHERE
        sr."status" = 'PENDING_ASSIGNMENT'
), unique_actions AS (
    SELECT
        -- distinct on actionId, tenantId
        DISTINCT ON (s."actionId", s."tenantId")
        s."actionId",
        s."tenantId"
    FROM
        steps s
)
INSERT INTO "Queue" (
    "tenantId",
    "name"
)
SELECT
    ua."tenantId",
    ua."actionId"
FROM
    unique_actions ua
ON CONFLICT ("tenantId", "name") DO NOTHING;

-- Set all existing step runs in a pending assignment state the the default queue named after their actionId
-- This query is idempotent and can run multiple times.
WITH step_runs AS (
    SELECT
        sr."id",
        s."actionId"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    WHERE
        sr."status" = 'PENDING_ASSIGNMENT'
        AND sr."queue" = 'default'
)
UPDATE
    "StepRun" sr
SET
    "queue" = sr2."actionId"
FROM
    step_runs sr2
WHERE
    sr."id" = sr2."id";

-- For all step runs in a pending assignment state, insert them into the queue based on their created at time. 
-- This query is idempotent and can run multiple times.
WITH pending_assignments AS (
    SELECT
        sr."id",
        sr."tenantId",
        sr."scheduleTimeoutAt",
        s."actionId",
        s."id" AS "stepId",
        s."timeout" AS "stepTimeout"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    WHERE
        sr."status" = 'PENDING_ASSIGNMENT'
), pending_assignments_with_qi AS (
    SELECT
        pa."id"
    FROM
        pending_assignments pa
    JOIN
        "QueueItem" qi ON pa."id" = qi."stepRunId"
)
INSERT INTO "QueueItem" (
    "stepRunId",
    "stepId",
    "actionId",
    "scheduleTimeoutAt",
    "stepTimeout",
    "priority",
    "isQueued",
    "tenantId",
    "queue"
)
SELECT
    pa."id",
    pa."stepId",
    pa."actionId",
    pa."scheduleTimeoutAt",
    pa."stepTimeout",
    1,
    true,
    pa."tenantId",
    pa."actionId"
FROM
    pending_assignments pa
WHERE
    pa."id" NOT IN (SELECT "id" FROM pending_assignments_with_qi);
