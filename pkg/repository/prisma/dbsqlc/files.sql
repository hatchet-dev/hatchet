-- name: GetFileByID :one
SELECT
    *
FROM
    "File"
WHERE
    "deletedAt" IS NOT NULL AND
    "id" = @id::uuid;

-- name: CreateFile :one
INSERT INTO "File" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "fileName",
    "filePath",
    "tenantId",
    "additionalMetadata"
) VALUES (
    @id::uuid,
    coalesce(sqlc.narg('createdAt')::timestamp, CURRENT_TIMESTAMP),
    coalesce(sqlc.narg('updatedAt')::timestamp, CURRENT_TIMESTAMP),
    @deletedAt::timestamp,
    @fileName::text,
    @filePath::text,
    @tenantId::uuid,
    @additionalMetadata::jsonb
) RETURNING *;


-- name: ListFilesByIDs :many
SELECT
    *
FROM
    "File" as files
WHERE
    files."deletedAt" IS NOT NULL AND
    "tenantId" = @tenantId::uuid AND
    "id" = ANY (sqlc.arg('ids')::uuid[]);
