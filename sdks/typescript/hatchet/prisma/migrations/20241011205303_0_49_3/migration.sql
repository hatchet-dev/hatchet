/*
  Warnings:

  - The primary key for the `EventKey` table will be changed. If it partially fails, the table could be left without primary key constraint.

*/
-- AlterTable
ALTER TABLE "EventKey" DROP CONSTRAINT "EventKey_pkey",
ADD COLUMN     "id" BIGSERIAL NOT NULL,
ADD CONSTRAINT "EventKey_pkey" PRIMARY KEY ("id");
