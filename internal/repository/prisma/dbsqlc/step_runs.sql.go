// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.24.0
// source: step_runs.sql

package dbsqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const resolveLaterStepRuns = `-- name: ResolveLaterStepRuns :many
WITH currStepRun AS (
  SELECT id, "createdAt", "updatedAt", "deletedAt", "tenantId", "jobRunId", "stepId", "nextId", "order", "workerId", "tickerId", status, input, output, "requeueAfter", "scheduleTimeoutAt", error, "startedAt", "finishedAt", "timeoutAt", "cancelledAt", "cancelledReason", "cancelledError"
  FROM "StepRun"
  WHERE
    "id" = $1::uuid AND
    "tenantId" = $2::uuid
)
UPDATE
    "StepRun" as sr
SET "status" = CASE
    -- When the given step run has failed or been cancelled, then all later step runs are cancelled
    WHEN (cs."status" = 'FAILED' OR cs."status" = 'CANCELLED') THEN 'CANCELLED'
    ELSE sr."status"
    END,
    -- When the previous step run timed out, the cancelled reason is set
    "cancelledReason" = CASE
    WHEN (cs."status" = 'CANCELLED' AND cs."cancelledReason" = 'TIMED_OUT'::text) THEN 'PREVIOUS_STEP_TIMED_OUT'
    WHEN (cs."status" = 'CANCELLED') THEN 'PREVIOUS_STEP_CANCELLED'
    ELSE NULL
    END
FROM
    currStepRun cs
WHERE
    sr."jobRunId" = (
        SELECT "jobRunId"
        FROM "StepRun"
        WHERE "id" = $1::uuid
    ) AND
    sr."order" > (
        SELECT "order"
        FROM "StepRun"
        WHERE "id" = $1::uuid
    ) AND
    sr."tenantId" = $2::uuid
RETURNING sr.id, sr."createdAt", sr."updatedAt", sr."deletedAt", sr."tenantId", sr."jobRunId", sr."stepId", sr."nextId", sr."order", sr."workerId", sr."tickerId", sr.status, sr.input, sr.output, sr."requeueAfter", sr."scheduleTimeoutAt", sr.error, sr."startedAt", sr."finishedAt", sr."timeoutAt", sr."cancelledAt", sr."cancelledReason", sr."cancelledError"
`

type ResolveLaterStepRunsParams struct {
	Steprunid pgtype.UUID `json:"steprunid"`
	Tenantid  pgtype.UUID `json:"tenantid"`
}

func (q *Queries) ResolveLaterStepRuns(ctx context.Context, db DBTX, arg ResolveLaterStepRunsParams) ([]*StepRun, error) {
	rows, err := db.Query(ctx, resolveLaterStepRuns, arg.Steprunid, arg.Tenantid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*StepRun
	for rows.Next() {
		var i StepRun
		if err := rows.Scan(
			&i.ID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
			&i.TenantId,
			&i.JobRunId,
			&i.StepId,
			&i.NextId,
			&i.Order,
			&i.WorkerId,
			&i.TickerId,
			&i.Status,
			&i.Input,
			&i.Output,
			&i.RequeueAfter,
			&i.ScheduleTimeoutAt,
			&i.Error,
			&i.StartedAt,
			&i.FinishedAt,
			&i.TimeoutAt,
			&i.CancelledAt,
			&i.CancelledReason,
			&i.CancelledError,
		); err != nil {
			return nil, err
		}
		items = append(items, &i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateStepRun = `-- name: UpdateStepRun :one
UPDATE
    "StepRun"
SET
    "requeueAfter" = COALESCE($1::timestamp, "requeueAfter"),
    "scheduleTimeoutAt" = COALESCE($2::timestamp, "scheduleTimeoutAt"),
    "startedAt" = COALESCE($3::timestamp, "startedAt"),
    "finishedAt" = COALESCE($4::timestamp, "finishedAt"),
    "status" = CASE 
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE COALESCE($5, "status")
    END,
    "input" = COALESCE($6::jsonb, "input"),
    "output" = COALESCE($7::jsonb, "output"),
    "error" = COALESCE($8::text, "error"),
    "cancelledAt" = COALESCE($9::timestamp, "cancelledAt"),
    "cancelledReason" = COALESCE($10::text, "cancelledReason")
WHERE 
  "id" = $11::uuid AND
  "tenantId" = $12::uuid
RETURNING "StepRun".id, "StepRun"."createdAt", "StepRun"."updatedAt", "StepRun"."deletedAt", "StepRun"."tenantId", "StepRun"."jobRunId", "StepRun"."stepId", "StepRun"."nextId", "StepRun"."order", "StepRun"."workerId", "StepRun"."tickerId", "StepRun".status, "StepRun".input, "StepRun".output, "StepRun"."requeueAfter", "StepRun"."scheduleTimeoutAt", "StepRun".error, "StepRun"."startedAt", "StepRun"."finishedAt", "StepRun"."timeoutAt", "StepRun"."cancelledAt", "StepRun"."cancelledReason", "StepRun"."cancelledError"
`

type UpdateStepRunParams struct {
	RequeueAfter      pgtype.Timestamp  `json:"requeueAfter"`
	ScheduleTimeoutAt pgtype.Timestamp  `json:"scheduleTimeoutAt"`
	StartedAt         pgtype.Timestamp  `json:"startedAt"`
	FinishedAt        pgtype.Timestamp  `json:"finishedAt"`
	Status            NullStepRunStatus `json:"status"`
	Input             []byte            `json:"input"`
	Output            []byte            `json:"output"`
	Error             pgtype.Text       `json:"error"`
	CancelledAt       pgtype.Timestamp  `json:"cancelledAt"`
	CancelledReason   pgtype.Text       `json:"cancelledReason"`
	ID                pgtype.UUID       `json:"id"`
	Tenantid          pgtype.UUID       `json:"tenantid"`
}

func (q *Queries) UpdateStepRun(ctx context.Context, db DBTX, arg UpdateStepRunParams) (*StepRun, error) {
	row := db.QueryRow(ctx, updateStepRun,
		arg.RequeueAfter,
		arg.ScheduleTimeoutAt,
		arg.StartedAt,
		arg.FinishedAt,
		arg.Status,
		arg.Input,
		arg.Output,
		arg.Error,
		arg.CancelledAt,
		arg.CancelledReason,
		arg.ID,
		arg.Tenantid,
	)
	var i StepRun
	err := row.Scan(
		&i.ID,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
		&i.TenantId,
		&i.JobRunId,
		&i.StepId,
		&i.NextId,
		&i.Order,
		&i.WorkerId,
		&i.TickerId,
		&i.Status,
		&i.Input,
		&i.Output,
		&i.RequeueAfter,
		&i.ScheduleTimeoutAt,
		&i.Error,
		&i.StartedAt,
		&i.FinishedAt,
		&i.TimeoutAt,
		&i.CancelledAt,
		&i.CancelledReason,
		&i.CancelledError,
	)
	return &i, err
}
