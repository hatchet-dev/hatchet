-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "schedulerPartitionId" TEXT;

-- CreateTable
CREATE TABLE "SchedulerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),
    "name" TEXT,

    CONSTRAINT "SchedulerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "SchedulerPartition_id_key" ON "SchedulerPartition"("id");

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_schedulerPartitionId_fkey" FOREIGN KEY ("schedulerPartitionId") REFERENCES "SchedulerPartition"("id") ON DELETE SET NULL ON UPDATE SET NULL;
