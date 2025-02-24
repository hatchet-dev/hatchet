-- AlterEnum
-- This migration adds more than one value to an enum.
-- With PostgreSQL versions 11 and earlier, this is not possible
-- in a single migration. This can be worked around by creating
-- multiple migrations, each migration adding only one value to
-- the enum.


ALTER TYPE "StepRunEventReason" ADD VALUE 'TIMED_OUT';
ALTER TYPE "StepRunEventReason" ADD VALUE 'REASSIGNED';
ALTER TYPE "StepRunEventReason" ADD VALUE 'SLOT_RELEASED';

-- AlterTable
ALTER TABLE "StepRun" ADD COLUMN     "semaphoreReleased" BOOLEAN NOT NULL DEFAULT false;
