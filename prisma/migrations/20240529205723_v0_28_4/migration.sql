/* Warnings:
- You are about to drop the `WorkerSemaphore` table. If the table is not empty, all the data it contains will be lost.
- Made the column `maxRuns` on table `Worker` required. This step will fail if there are existing NULL values in that column.
*/

-- DropForeignKey
ALTER TABLE "WorkerSemaphore" DROP CONSTRAINT IF EXISTS "WorkerSemaphore_workerId_fkey";

-- Update existing workers with NULL maxRuns to have a default value
UPDATE "Worker" SET "maxRuns" = 100 WHERE "maxRuns" IS NULL;

-- AlterTable
ALTER TABLE "Worker" ALTER COLUMN "maxRuns" SET NOT NULL,
                     ALTER COLUMN "maxRuns" SET DEFAULT 100;

-- DropTable
DROP TABLE IF EXISTS "WorkerSemaphore";

-- CreateTable
CREATE TABLE IF NOT EXISTS "WorkerSemaphoreSlot" (
    "id" UUID NOT NULL,
    "workerId" UUID NOT NULL,
    "stepRunId" UUID,
    CONSTRAINT "WorkerSemaphoreSlot_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX IF NOT EXISTS "WorkerSemaphoreSlot_id_key" ON "WorkerSemaphoreSlot"("id");

-- CreateIndex
CREATE UNIQUE INDEX IF NOT EXISTS "WorkerSemaphoreSlot_stepRunId_key" ON "WorkerSemaphoreSlot"("stepRunId");

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreSlot"
ADD CONSTRAINT "WorkerSemaphoreSlot_workerId_fkey"
FOREIGN KEY ("workerId") REFERENCES "Worker"("id")
ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerSemaphoreSlot"
ADD CONSTRAINT "WorkerSemaphoreSlot_stepRunId_fkey"
FOREIGN KEY ("stepRunId") REFERENCES "StepRun"("id")
ON DELETE CASCADE ON UPDATE CASCADE;

-- Create maxRun semaphore slots for each worker with a recent heartbeat
INSERT INTO "WorkerSemaphoreSlot" ("id", "workerId")
SELECT gen_random_uuid(), w.id
FROM "Worker" w
CROSS JOIN generate_series(1, COALESCE(w."maxRuns", 100))
WHERE w."lastHeartbeatAt" >= NOW() - INTERVAL '10 hours'
ON CONFLICT DO NOTHING;

-- -- Update a null slot for each step that is currently running or assigned
WITH step_run_counts_per_worker AS (
    SELECT "workerId", COUNT(*) AS "cnt"
    FROM "StepRun"
    WHERE "status" IN ('RUNNING', 'ASSIGNED')
    GROUP BY "workerId"
)
UPDATE "WorkerSemaphoreSlot" wss
SET "stepRunId" = wss3."srid"
FROM (
    SELECT DISTINCT sr."id" AS "srid", rns."id" AS "wssid"
    FROM (
        SELECT 
            "id",
            "workerId",
            ROW_NUMBER() OVER (PARTITION BY "workerId") AS "rowNumber"
        FROM "WorkerSemaphoreSlot" wss3
        WHERE wss3."stepRunId" IS NULL
    ) rns
    JOIN step_run_counts_per_worker sr_counts ON sr_counts."workerId" = rns."workerId"
    JOIN "StepRun" sr ON sr."workerId" = rns."workerId" AND sr."status" IN ('RUNNING', 'ASSIGNED')
    WHERE rns."rowNumber" <= sr_counts."cnt"
) wss3
WHERE wss."id" = wss3."wssid";