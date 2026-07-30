-- name: CountAllTenants :one
SELECT COUNT(*) FROM "Tenant";

-- name: CountAllUsers :one
SELECT COUNT(*) FROM "User";

-- name: CountAllWorkflows :one
SELECT COUNT(*) FROM "Workflow" WHERE "deletedAt" IS NULL;

-- name: CountAllWorkers :one
SELECT COUNT(*) FROM "Worker" WHERE "deletedAt" IS NULL;

-- name: CountAllWorkersByLanguage :many
SELECT COALESCE("language"::text, 'UNKNOWN')::text AS language, COUNT(*) AS count
FROM "Worker"
WHERE "deletedAt" IS NULL
GROUP BY "language";
