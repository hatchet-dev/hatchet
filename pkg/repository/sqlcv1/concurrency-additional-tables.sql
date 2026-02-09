-- sqlc needs this table model in order to generate queries, though this is never written to the schema
CREATE TABLE tmp_workflow_concurrency_slot (LIKE v1_workflow_concurrency_slot INCLUDING ALL);
