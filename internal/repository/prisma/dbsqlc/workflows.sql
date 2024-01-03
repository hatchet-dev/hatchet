-- name: CountWorkflows :one
SELECT
    count(workflows) OVER() AS total
FROM
    "Workflow" as workflows 
WHERE
    workflows."tenantId" = $1 AND
    (
        sqlc.narg('eventKey')::text IS NULL OR
        workflows."id" IN (
            SELECT 
                DISTINCT ON(t1."workflowId") t1."workflowId"
            FROM 
                "WorkflowVersion" AS t1
                LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = t1."id" 
            WHERE 
                (
                    j2."id" IN (
                        SELECT 
                            t3."parentId" 
                        FROM 
                            "public"."WorkflowTriggerEventRef" AS t3
                        WHERE 
                            t3."eventKey" = sqlc.narg('eventKey')::text
                            AND t3."parentId" IS NOT NULL
                    ) 
                    AND j2."id" IS NOT NULL 
                    AND t1."workflowId" IS NOT NULL
                )
            ORDER BY 
                t1."workflowId" DESC, t1."order" DESC
        )
    );

-- name: ListWorkflowsLatestRuns :many
SELECT
    DISTINCT ON (workflow."id") sqlc.embed(runs), workflow."id" as "workflowId"
FROM
    "WorkflowRun" as runs
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
    runs."tenantId" = $1 AND
    (
        sqlc.narg('eventKey')::text IS NULL OR
        workflow."id" IN (
            SELECT 
                DISTINCT ON(t1."workflowId") t1."workflowId"
            FROM 
                "WorkflowVersion" AS t1
                LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = t1."id" 
            WHERE 
                (
                    j2."id" IN (
                        SELECT 
                            t3."parentId" 
                        FROM 
                            "public"."WorkflowTriggerEventRef" AS t3
                        WHERE 
                            t3."eventKey" = sqlc.narg('eventKey')::text
                            AND t3."parentId" IS NOT NULL
                    ) 
                    AND j2."id" IS NOT NULL 
                    AND t1."workflowId" IS NOT NULL
                )
            ORDER BY 
                t1."workflowId" DESC, t1."order" DESC
        )
    )
ORDER BY 
    workflow."id" DESC, runs."createdAt" DESC; 

-- name: ListWorkflows :many
SELECT 
    sqlc.embed(workflows)
FROM (
    SELECT
        DISTINCT ON(workflows."id") workflows.*
    FROM
        "Workflow" as workflows 
    LEFT JOIN
        (
            SELECT * FROM "WorkflowVersion" as workflowVersion ORDER BY workflowVersion."order" DESC LIMIT 1
        ) as workflowVersion ON workflows."id" = workflowVersion."workflowId"
    LEFT JOIN
        "WorkflowTriggers" as workflowTrigger ON workflowVersion."id" = workflowTrigger."workflowVersionId"
    LEFT JOIN
        "WorkflowTriggerEventRef" as workflowTriggerEventRef ON workflowTrigger."id" = workflowTriggerEventRef."parentId"
    WHERE
        workflows."tenantId" = $1 
        AND
        (
            sqlc.narg('eventKey')::text IS NULL OR
            workflows."id" IN (
                SELECT 
                    DISTINCT ON(t1."workflowId") t1."workflowId"
                FROM 
                    "WorkflowVersion" AS t1
                    LEFT JOIN "WorkflowTriggers" AS j2 ON j2."workflowVersionId" = t1."id" 
                WHERE 
                    (
                        j2."id" IN (
                            SELECT 
                                t3."parentId" 
                            FROM 
                                "public"."WorkflowTriggerEventRef" AS t3
                            WHERE 
                                t3."eventKey" = sqlc.narg('eventKey')::text
                                AND t3."parentId" IS NOT NULL
                        ) 
                        AND j2."id" IS NOT NULL 
                        AND t1."workflowId" IS NOT NULL
                    )
                ORDER BY 
                    t1."workflowId" DESC
            )
        )
    ORDER BY workflows."id" DESC
) as workflows
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN workflows."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then workflows."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);