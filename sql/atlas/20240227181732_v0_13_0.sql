-- DropForeignKey
ALTER TABLE "Worker" DROP CONSTRAINT "Worker_dispatcherId_fkey";

-- AlterTable
ALTER TABLE "Worker" ALTER COLUMN "dispatcherId" DROP NOT NULL;

-- AddForeignKey
ALTER TABLE "Worker" ADD CONSTRAINT "Worker_dispatcherId_fkey" FOREIGN KEY ("dispatcherId") REFERENCES "Dispatcher"("id") ON DELETE SET NULL ON UPDATE CASCADE;
