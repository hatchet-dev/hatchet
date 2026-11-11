package workflows

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workflowStub struct {
	repository.WorkflowRepository
}

func (workflowStub) GetWorkflowByName(context.Context, uuid.UUID, string) (*sqlcv1.Workflow, error) {
	return &sqlcv1.Workflow{ID: uuid.New()}, nil
}

// scheduleStub records the cron it is asked to create and fails, so the handler stops before
// publishing to the message queue.
type scheduleStub struct {
	repository.WorkflowScheduleRepository
	created *string
	err     error
}

func (s scheduleStub) CreateCronWorkflow(_ context.Context, _ uuid.UUID, opts *repository.CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error) {
	*s.created = opts.Cron
	if s.err != nil {
		return nil, s.err
	}
	return nil, errors.New("stub: not stored")
}

type repositoryStub struct {
	repository.Repository
	schedules scheduleStub
}

func (repositoryStub) Workflows() repository.WorkflowRepository { return workflowStub{} }

func (s repositoryStub) WorkflowSchedules() repository.WorkflowScheduleRepository {
	return s.schedules
}

func TestCronWorkflowTriggerCreateValidatesExpression(t *testing.T) {
	cases := []struct {
		name    string
		cron    string
		invalid bool
	}{
		{name: "day of week 7 is rejected", cron: "0 9 * * 1-7", invalid: true},
		{name: "descriptor is stored", cron: "@daily"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var created string
			svc := NewWorkflowService(&server.ServerConfig{
				Layer:     &database.Layer{V1: repositoryStub{schedules: scheduleStub{created: &created}}},
				Validator: validator.NewDefaultValidator(),
			})

			ctx := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
			ctx.Set("tenant", &sqlcv1.Tenant{ID: uuid.New()})

			resp, err := svc.CronWorkflowTriggerCreate(ctx, gen.CronWorkflowTriggerCreateRequestObject{
				Workflow: "workflow",
				Body:     &gen.CreateCronWorkflowTriggerRequest{CronName: "name", CronExpression: tc.cron},
			})
			if !tc.invalid {
				require.Equal(t, tc.cron, created)
				require.ErrorContains(t, err, "could not create cron trigger: stub: not stored")
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.Empty(t, created, "an invalid expression must not be stored")
			badRequest, ok := resp.(gen.CronWorkflowTriggerCreate400JSONResponse)
			require.True(t, ok, "expected a 400 response, got %T", resp)
			require.Len(t, badRequest.Errors, 1)
			require.Equal(t, "cron", *badRequest.Errors[0].Field)
		})
	}
}

func TestCronCreationDistinguishesStorageFailuresFromDuplicates(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		var cause error = errors.New("private database marker")
		if duplicate {
			cause = &pgconn.PgError{Code: "23505", Message: "private constraint marker"}
		}
		var created string
		svc := NewWorkflowService(&server.ServerConfig{
			Layer:     &database.Layer{V1: repositoryStub{schedules: scheduleStub{created: &created, err: cause}}},
			Validator: validator.NewDefaultValidator(),
		})
		ctx := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
		ctx.Set("tenant", &sqlcv1.Tenant{ID: uuid.New()})
		resp, err := svc.CronWorkflowTriggerCreate(ctx, gen.CronWorkflowTriggerCreateRequestObject{
			Workflow: "workflow",
			Body:     &gen.CreateCronWorkflowTriggerRequest{CronName: "name", CronExpression: "@daily"},
		})
		if duplicate {
			require.NoError(t, err)
			out := httptest.NewRecorder()
			require.NoError(t, resp.VisitCronWorkflowTriggerCreateResponse(out))
			require.Equal(t, http.StatusBadRequest, out.Code)
			require.Contains(t, out.Body.String(), "already exists")
			require.NotContains(t, out.Body.String(), "private")
		} else {
			require.Nil(t, resp)
			require.ErrorIs(t, err, cause, "middleware must receive the original storage failure")
		}
	}
}
