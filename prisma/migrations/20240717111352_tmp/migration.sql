-- CreateEnum
CREATE TYPE "WorkerLabelComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');

-- CreateTable
CREATE TABLE "StepDesiredWorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stepId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,
    "required" BOOLEAN NOT NULL,
    "comparator" "WorkerLabelComparator" NOT NULL,
    "weight" INTEGER NOT NULL,

    CONSTRAINT "StepDesiredWorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "WorkerLabel" (
    "id" BIGSERIAL NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "workerId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "strValue" TEXT,
    "intValue" INTEGER,

    CONSTRAINT "WorkerLabel_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "StepDesiredWorkerLabel_stepId_idx" ON "StepDesiredWorkerLabel"("stepId");

-- CreateIndex
CREATE UNIQUE INDEX "StepDesiredWorkerLabel_stepId_key_key" ON "StepDesiredWorkerLabel"("stepId", "key");

-- CreateIndex
CREATE INDEX "WorkerLabel_workerId_idx" ON "WorkerLabel"("workerId");

-- CreateIndex
CREATE UNIQUE INDEX "WorkerLabel_workerId_key_key" ON "WorkerLabel"("workerId", "key");

-- AddForeignKey
ALTER TABLE "StepDesiredWorkerLabel" ADD CONSTRAINT "StepDesiredWorkerLabel_stepId_fkey" FOREIGN KEY ("stepId") REFERENCES "Step"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "WorkerLabel" ADD CONSTRAINT "WorkerLabel_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
