-- AlterTable
ALTER TABLE "Tenant" ADD COLUMN     "controllerPartitionId" TEXT,
ADD COLUMN     "workerPartitionId" TEXT;

-- CreateTable
CREATE TABLE "ControllerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),

    CONSTRAINT "ControllerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "TenantWorkerPartition" (
    "id" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastHeartbeat" TIMESTAMP(3),

    CONSTRAINT "TenantWorkerPartition_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "SecurityCheckIdent" (
    "id" UUID NOT NULL,

    CONSTRAINT "SecurityCheckIdent_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "ControllerPartition_id_key" ON "ControllerPartition"("id");

-- CreateIndex
CREATE UNIQUE INDEX "TenantWorkerPartition_id_key" ON "TenantWorkerPartition"("id");

-- CreateIndex
CREATE UNIQUE INDEX "SecurityCheckIdent_id_key" ON "SecurityCheckIdent"("id");

-- CreateIndex
CREATE INDEX "Tenant_controllerPartitionId_idx" ON "Tenant"("controllerPartitionId");

-- CreateIndex
CREATE INDEX "Tenant_workerPartitionId_idx" ON "Tenant"("workerPartitionId");

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_controllerPartitionId_fkey" FOREIGN KEY ("controllerPartitionId") REFERENCES "ControllerPartition"("id") ON DELETE SET NULL ON UPDATE SET NULL;

-- AddForeignKey
ALTER TABLE "Tenant" ADD CONSTRAINT "Tenant_workerPartitionId_fkey" FOREIGN KEY ("workerPartitionId") REFERENCES "TenantWorkerPartition"("id") ON DELETE SET NULL ON UPDATE SET NULL;
