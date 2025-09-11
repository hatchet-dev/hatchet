-- name: IsTenantAlertActive :one
WITH active_setting AS (
    SELECT
        1
    FROM
        "TenantAlertingSettings" as alerts
    WHERE
        "tenantId" = @tenantId::uuid AND
        (
            "lastAlertedAt" IS NULL OR
            "lastAlertedAt" <= NOW() - convert_duration_to_interval(alerts."maxFrequency")
        ) AND
        "enableWorkflowRunFailureAlerts" = true
)
SELECT
    EXISTS (
        select 1 from active_setting
    ) as "isActive",
    "lastAlertedAt" as "lastAlertedAt"
FROM
    "TenantAlertingSettings" as alerts
WHERE
    "tenantId" = @tenantId::uuid;
