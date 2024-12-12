-- +goose Up
-- CreateTable
CREATE TABLE "WorkerSemaphore" (
    "workerId" UUID NOT NULL,
    "slots" INTEGER NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "WorkerSemaphore_workerId_key" ON "WorkerSemaphore"("workerId");

-- AddForeignKey
ALTER TABLE "WorkerSemaphore" ADD CONSTRAINT "WorkerSemaphore_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
