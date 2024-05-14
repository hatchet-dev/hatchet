-- AlterEnum
ALTER TYPE "StepRunEventReason" ADD VALUE 'SLOT_RELEASED';

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "semaphoreReleased" BOOLEAN NOT NULL DEFAULT false;
