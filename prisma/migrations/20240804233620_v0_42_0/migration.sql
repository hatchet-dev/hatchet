-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "queue" TEXT NOT NULL DEFAULT 'default';

-- CreateIndex
CREATE INDEX "StepRun_queue_createdAt_idx" ON "StepRun"("queue", "createdAt");
