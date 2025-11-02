-- name: CheckBloat :one
SELECT COUNT(*) FROM "Workflow" WHERE "deletedAt" IS NULL;