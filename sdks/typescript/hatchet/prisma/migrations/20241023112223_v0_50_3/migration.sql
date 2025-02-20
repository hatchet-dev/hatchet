-- AlterTable
ALTER TABLE "Queue" ADD COLUMN     "lastActive" TIMESTAMP(3);

-- CreateIndex
CREATE INDEX "Queue_tenantId_lastActive_idx" ON "Queue"("tenantId", "lastActive");
