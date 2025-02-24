-- Modify "Event" table
ALTER TABLE "Event" ALTER COLUMN "createdAt" TYPE timestamp, ALTER COLUMN "createdAt" SET DEFAULT clock_timestamp();
