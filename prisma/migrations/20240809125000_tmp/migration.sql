-- AlterTable
ALTER TABLE "QueueItem" ADD COLUMN     "desiredWorkerId" UUID,
ADD COLUMN     "sticky" "StickyStrategy";
