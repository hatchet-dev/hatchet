/*
  Warnings:

  - The primary key for the `Action` table will be changed. If it partially fails, the table could be left without primary key constraint.
  - The `refreshToken` column on the `UserOAuth` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - A unique constraint covering the columns `[id]` on the table `Action` will be added. If there are existing duplicate values, this will fail.
  - A unique constraint covering the columns `[tenantId,actionId]` on the table `Action` will be added. If there are existing duplicate values, this will fail.
  - Added the required column `actionId` to the `Action` table without a default value. This is not possible if the table is not empty.
  - Changed the type of `id` on the `Action` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Changed the type of `accessToken` on the `UserOAuth` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.
  - Added the required column `checksum` to the `WorkflowVersion` table without a default value. This is not possible if the table is not empty.
  - Changed the type of `A` on the `_ActionToWorker` table. No cast exists, the column would be dropped and recreated, which cannot be done if there is data, since the column is required.

*/
-- DropForeignKey
ALTER TABLE "Step" DROP CONSTRAINT "Step_actionId_tenantId_fkey";

-- DropForeignKey
ALTER TABLE "_ActionToWorker" DROP CONSTRAINT "_ActionToWorker_A_fkey";

-- DropIndex
DROP INDEX "Action_tenantId_id_key";

-- DropIndex
DROP INDEX "WorkflowVersion_workflowId_version_key";

-- AlterTable
ALTER TABLE "Action" DROP CONSTRAINT "Action_pkey",
ADD COLUMN     "actionId" TEXT NOT NULL,
DROP COLUMN "id",
ADD COLUMN     "id" UUID NOT NULL,
ADD CONSTRAINT "Action_pkey" PRIMARY KEY ("id");

-- AlterTable
ALTER TABLE "UserOAuth" DROP COLUMN "accessToken",
ADD COLUMN     "accessToken" BYTEA NOT NULL,
DROP COLUMN "refreshToken",
ADD COLUMN     "refreshToken" BYTEA;

-- AlterTable
ALTER TABLE "WorkflowVersion" ADD COLUMN "checksum" TEXT;

-- Add a default random string value to existing rows
UPDATE "WorkflowVersion"
SET "checksum" = md5(random()::text || clock_timestamp()::text);

-- Make the checksum column NOT NULL
ALTER TABLE "WorkflowVersion" ALTER COLUMN "checksum" SET NOT NULL;

-- Update the version column to allow NULL
ALTER TABLE "WorkflowVersion" ALTER COLUMN "version" DROP NOT NULL;

-- AlterTable
ALTER TABLE "_ActionToWorker" DROP COLUMN "A",
ADD COLUMN     "A" UUID NOT NULL;

-- CreateTable
CREATE TABLE "APIToken" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expiresAt" TIMESTAMP(3),
    "revoked" BOOLEAN NOT NULL DEFAULT false,
    "name" TEXT,
    "tenantId" UUID,

    CONSTRAINT "APIToken_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "APIToken_id_key" ON "APIToken"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Action_id_key" ON "Action"("id");

-- CreateIndex
CREATE UNIQUE INDEX "Action_tenantId_actionId_key" ON "Action"("tenantId", "actionId");

-- CreateIndex
CREATE UNIQUE INDEX "_ActionToWorker_AB_unique" ON "_ActionToWorker"("A", "B");

-- AddForeignKey
ALTER TABLE "APIToken" ADD CONSTRAINT "APIToken_tenantId_fkey" FOREIGN KEY ("tenantId") REFERENCES "Tenant"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "Step" ADD CONSTRAINT "Step_actionId_tenantId_fkey" FOREIGN KEY ("actionId", "tenantId") REFERENCES "Action"("actionId", "tenantId") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_ActionToWorker" ADD CONSTRAINT "_ActionToWorker_A_fkey" FOREIGN KEY ("A") REFERENCES "Action"("id") ON DELETE CASCADE ON UPDATE CASCADE;
