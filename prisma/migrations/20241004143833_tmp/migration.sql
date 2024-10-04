-- CreateEnum
CREATE TYPE "WorkerSDKS" AS ENUM ('UNKNOWN', 'GO', 'PYTHON', 'TYPESCRIPT');

-- AlterTable
ALTER TABLE "Worker" ADD COLUMN     "language" "WorkerSDKS",
ADD COLUMN     "languageVersion" TEXT,
ADD COLUMN     "os" TEXT,
ADD COLUMN     "runtimeExtra" TEXT,
ADD COLUMN     "sdkVersion" TEXT;
