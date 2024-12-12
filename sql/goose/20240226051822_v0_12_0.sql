-- +goose Up
-- AlterEnum
ALTER TYPE "ConcurrencyLimitStrategy" ADD VALUE 'GROUP_ROUND_ROBIN';

-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "maxRuns" INTEGER;
