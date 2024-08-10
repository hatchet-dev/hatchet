/*
  Warnings:

  - You are about to drop the column `data` on the `File` table. All the data in the column will be lost.
  - Added the required column `fileName` to the `File` table without a default value. This is not possible if the table is not empty.
  - Added the required column `filePath` to the `File` table without a default value. This is not possible if the table is not empty.

*/
-- AlterTable
ALTER TABLE "File" DROP COLUMN "data",
ADD COLUMN     "fileName" TEXT NOT NULL,
ADD COLUMN     "filePath" TEXT NOT NULL;
