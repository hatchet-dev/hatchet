-- +goose Up
-- +goose StatementBegin
ALTER TABLE "Worker" ADD COLUMN "operatorId" UUID;

ALTER TABLE "Step" ADD COLUMN "isDagOrchestrator" BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE v1_task
    ADD COLUMN is_dag_orchestrator BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_dag_subtask BOOLEAN NOT NULL DEFAULT false
;

ALTER TABLE v1_tasks_olap
    ADD COLUMN is_dag_orchestrator BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_dag_subtask BOOLEAN NOT NULL DEFAULT false
;

ALTER TABLE "WorkflowVersion" ADD COLUMN "isUsingDagOperator" BOOLEAN NOT NULL DEFAULT false;

CREATE TYPE v1_operator_kind AS ENUM ('HTTP_API', 'DAG');

CREATE TABLE v1_operator (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    kind v1_operator_kind NOT NULL,
    config JSONB NOT NULL,
    worker_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT v1_operator_pkey PRIMARY KEY (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "Step" DROP COLUMN "isDagOrchestrator";
ALTER TABLE "WorkflowVersion" DROP COLUMN "isUsingDagOperator";
ALTER TABLE v1_tasks_olap
    DROP COLUMN is_dag_orchestrator,
    DROP COLUMN is_dag_subtask
;
ALTER TABLE v1_task
    DROP COLUMN is_dag_orchestrator,
    DROP COLUMN is_dag_subtask
;

DROP TABLE v1_operator;

DROP TYPE v1_operator_kind;

ALTER TABLE "Worker" DROP COLUMN "operatorId";
-- +goose StatementEnd
