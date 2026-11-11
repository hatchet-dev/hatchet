-- +goose Up
/* Warnings:
- You are about to drop the `WorkerSemaphore` table. If the table is not empty, all the data it contains will be lost.
- Made the column `maxRuns` on table `Worker` required. This step will fail if there are existing NULL values in that column.
*/


-- Update existing workers with NULL maxRuns to have a default value
UPDATE "Worker" SET "maxRuns" = 100 WHERE "maxRuns" IS NULL;

-- AlterTable
ALTER TABLE "Worker" ALTER COLUMN "maxRuns" SET NOT NULL,
                     ALTER COLUMN "maxRuns" SET DEFAULT 100;

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

-- CreateIndex
CREATE INDEX "WorkerSemaphoreSlot_workerId_idx" ON "WorkerSemaphoreSlot"("workerId");

-- Create maxRun semaphore slots for each worker with a recent heartbeat
INSERT INTO "WorkerSemaphoreSlot" ("id", "workerId")
SELECT gen_random_uuid(), w.id
FROM "Worker" w
CROSS JOIN generate_series(1, COALESCE(w."maxRuns", 100))
WHERE w."lastHeartbeatAt" >= NOW() - INTERVAL '10 hours'
ON CONFLICT DO NOTHING;

-- -- Update a null slot for each step that is currently running or assigned
-- +goose StatementBegin
DO $$
DECLARE
    sr RECORD;
    wss RECORD;
BEGIN
    -- Loop over each running or assigned step run
    FOR sr IN
        SELECT "id", "workerId"
        FROM "StepRun"
        WHERE "status" IN ('RUNNING', 'ASSIGNED')
    LOOP
        -- Find one available WorkerSemaphoreSlot for the current workerId
        SELECT "id"
        INTO wss
        FROM "WorkerSemaphoreSlot"
        WHERE "workerId" = sr."workerId" AND "stepRunId" IS NULL
        LIMIT 1;

        -- If an available slot is found, update it with the stepRunId
        IF wss.id IS NOT NULL THEN
            UPDATE "WorkerSemaphoreSlot"
            SET "stepRunId" = sr.id
            WHERE "id" = wss.id;
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd
