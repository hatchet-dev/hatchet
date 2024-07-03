-- Create index "WorkerAffinity_workerId_key_key" to table: "WorkerAffinity"
CREATE UNIQUE INDEX "WorkerAffinity_workerId_key_key" ON "WorkerAffinity" ("workerId", "key");
