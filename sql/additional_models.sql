
CREATE  MATERIALIZED VIEW "WorkflowRunEventView"
   WITH (timescaledb.continuous
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
      ORDER BY minute
      WITH NO DATA;
