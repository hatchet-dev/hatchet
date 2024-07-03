-- CreateEnum
CREATE TYPE "AffinityComparator" AS ENUM ('EQUAL', 'NOT_EQUAL', 'GREATER_THAN', 'GREATER_THAN_OR_EQUAL', 'LESS_THAN', 'LESS_THAN_OR_EQUAL');

-- CreateTable
CREATE TABLE "WorkerAffinity" (
    "id" BIGSERIAL NOT NULL,
    "workerId" UUID NOT NULL,
    "key" TEXT NOT NULL,
    "value" JSONB,
    "comparator" "AffinityComparator" NOT NULL DEFAULT 'EQUAL',
    "weight" INTEGER NOT NULL DEFAULT 100,

    CONSTRAINT "WorkerAffinity_pkey" PRIMARY KEY ("id")
);

-- AddForeignKey
ALTER TABLE "WorkerAffinity" ADD CONSTRAINT "WorkerAffinity_workerId_fkey" FOREIGN KEY ("workerId") REFERENCES "Worker"("id") ON DELETE CASCADE ON UPDATE CASCADE;
