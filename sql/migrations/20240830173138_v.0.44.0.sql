-- atlas:txmode none

-- Create enum type "WorkflowRunEventType"
CREATE TYPE "WorkflowRunEventType" AS ENUM ('PENDING', 'QUEUED', 'RUNNING', 'SUCCEEDED', 'RETRIED', 'FAILED', 'QUEUE_DEPTH');
-- Create "WorkflowRunEvent" table
CREATE TABLE "WorkflowRunEvent" ("id" uuid NOT NULL, "createdAt" timestamptz(3) NOT NULL DEFAULT CURRENT_TIMESTAMP, "tenantId" uuid NOT NULL, "workflowRunId" uuid NOT NULL, "eventType" "WorkflowRunEventType" NOT NULL, PRIMARY KEY ("id", "createdAt"), CONSTRAINT "WorkflowRunEvent_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant" ("id") ON UPDATE CASCADE ON DELETE CASCADE, CONSTRAINT "WorkflowRunEvent_workflowRunId_fkey" FOREIGN KEY ("workflowRunId") REFERENCES "WorkflowRun" ("id") ON UPDATE CASCADE ON DELETE CASCADE);


SELECT * from create_hypertable('"WorkflowRunEvent"', by_range('createdAt',  INTERVAL '1 minute'));

CREATE  MATERIALIZED VIEW "WorkflowRunEventView"
   WITH (timescaledb.continuous,
         timescaledb.materialized_only = false
         )
   AS
      SELECT
         time_bucket('1 minute', "createdAt") AS minute,

        "tenantId",
        COUNT(*) FILTER (WHERE "eventType" = 'PENDING') AS pending_count,
        COUNT(*) FILTER (WHERE "eventType" = 'QUEUED') AS queued_count,
        COUNT(*) FILTER (WHERE "eventType" = 'RUNNING') AS running_count,
        COUNT(*) FILTER (WHERE "eventType" = 'SUCCEEDED') AS succeeded_count,
        COUNT(*) FILTER (WHERE "eventType" = 'RETRIED') AS retried_count,
        COUNT(*) FILTER (WHERE "eventType" = 'FAILED') AS failed_count,
        COUNT(*) FILTER (WHERE "eventType" = 'QUEUE_DEPTH') AS queue_depth


      FROM "WorkflowRunEvent"
      GROUP BY minute, "tenantId"
      ORDER BY minute;