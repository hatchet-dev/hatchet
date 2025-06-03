-- name: V1ListTenantsByControllerPartitionId :many
SELECT
    *
FROM
    "Tenant" as tenants
WHERE
    "controllerPartitionId" = sqlc.arg('controllerPartitionId')::text
    AND "version" = 'V1'::"TenantMajorEngineVersion"
    AND (
        sqlc.arg('withFilter')::boolean = false
        OR (
            (
                sqlc.arg('withTimeoutTasks')::boolean = false
                OR EXISTS (
                    SELECT 1
                    FROM v1_task_runtime vtr
                    WHERE vtr.tenant_id = tenants.id
                        AND vtr.timeout_at <= NOW() + INTERVAL '5 seconds' -- NOTE: this is a 5 second buffer to "look ahead" to account for poll interval
                )
            )
            AND (
                sqlc.arg('withExpiredSleeps')::boolean = false
                OR EXISTS (
                    SELECT 1
                    FROM v1_durable_sleep vds
                    WHERE vds.tenant_id = tenants.id
                        AND vds.sleep_until <= CURRENT_TIMESTAMP + INTERVAL '5 seconds' -- NOTE: this is a 5 second buffer to "look ahead" to account for poll interval
                )
            )
            AND (
                sqlc.arg('withRetryQueueItems')::boolean = false
                OR EXISTS (
                    SELECT 1
                    FROM v1_retry_queue_item rqi
                    WHERE rqi.tenant_id = tenants.id
                        AND rqi.retry_after <= NOW() + INTERVAL '5 seconds' -- NOTE: this is a 5 second buffer to "look ahead" to account for poll interval
                    ORDER BY
                        rqi.task_id, rqi.task_inserted_at, rqi.task_retry_count
                )
            )
            AND (
                sqlc.arg('withReassignTasks')::boolean = false
                OR EXISTS (
                    SELECT 1
                    FROM "Worker" w
                    JOIN  v1_task_runtime runtime ON w."id" = runtime.worker_id
                    WHERE w."tenantId" = tenants.id
                        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds' -- NOTE: this is 30 seconds should match the heartbeat check
                )
            )
        )
    );
