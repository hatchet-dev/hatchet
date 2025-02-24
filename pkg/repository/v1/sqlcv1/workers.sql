-- name: ListManyWorkerLabels :many
SELECT
    "id",
    "key",
    "intValue",
    "strValue",
    "createdAt",
    "updatedAt",
    "workerId"
FROM "WorkerLabel" wl
WHERE wl."workerId" = ANY(@workerIds::uuid[]);
