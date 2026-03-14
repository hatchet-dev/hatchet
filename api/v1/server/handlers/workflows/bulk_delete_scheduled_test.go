package workflows

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeWorkflowScheduleRepository struct {
	listRows         []*sqlcv1.ListScheduledWorkflowsRow
	listCount        int64
	listCalls        int
	bulkDeleteCalls  int
	bulkDeleteResult []uuid.UUID
}

func (f *fakeWorkflowScheduleRepository) ListScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, opts *v1.ListScheduledWorkflowsOpts) ([]*sqlcv1.ListScheduledWorkflowsRow, int64, error) {
	f.listCalls++
	return f.listRows, f.listCount, nil
}

func (f *fakeWorkflowScheduleRepository) DeleteScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) error {
	panic("unexpected call to DeleteScheduledWorkflow")
}

func (f *fakeWorkflowScheduleRepository) GetScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	panic("unexpected call to GetScheduledWorkflow")
}

func (f *fakeWorkflowScheduleRepository) UpdateScheduledWorkflow(ctx context.Context, tenantId, scheduledWorkflowId uuid.UUID, triggerAt time.Time) error {
	panic("unexpected call to UpdateScheduledWorkflow")
}

func (f *fakeWorkflowScheduleRepository) ScheduledWorkflowMetaByIds(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) (map[uuid.UUID]v1.ScheduledWorkflowMeta, error) {
	panic("unexpected call to ScheduledWorkflowMetaByIds")
}

func (f *fakeWorkflowScheduleRepository) BulkDeleteScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, scheduledWorkflowIds []uuid.UUID) ([]uuid.UUID, error) {
	f.bulkDeleteCalls++
	return f.bulkDeleteResult, nil
}

func (f *fakeWorkflowScheduleRepository) BulkUpdateScheduledWorkflows(ctx context.Context, tenantId uuid.UUID, updates []v1.ScheduledWorkflowUpdate) ([]uuid.UUID, error) {
	panic("unexpected call to BulkUpdateScheduledWorkflows")
}

func (f *fakeWorkflowScheduleRepository) CreateScheduledWorkflow(ctx context.Context, tenantId uuid.UUID, opts *v1.CreateScheduledWorkflowRunForWorkflowOpts) (*sqlcv1.ListScheduledWorkflowsRow, error) {
	panic("unexpected call to CreateScheduledWorkflow")
}

func (f *fakeWorkflowScheduleRepository) CreateCronWorkflow(ctx context.Context, tenantId uuid.UUID, opts *v1.CreateCronWorkflowTriggerOpts) (*sqlcv1.ListCronWorkflowsRow, error) {
	panic("unexpected call to CreateCronWorkflow")
}

func (f *fakeWorkflowScheduleRepository) ListCronWorkflows(ctx context.Context, tenantId uuid.UUID, opts *v1.ListCronWorkflowsOpts) ([]*sqlcv1.ListCronWorkflowsRow, int64, error) {
	panic("unexpected call to ListCronWorkflows")
}

func (f *fakeWorkflowScheduleRepository) GetCronWorkflow(ctx context.Context, tenantId, cronWorkflowId uuid.UUID) (*sqlcv1.ListCronWorkflowsRow, error) {
	panic("unexpected call to GetCronWorkflow")
}

func (f *fakeWorkflowScheduleRepository) DeleteCronWorkflow(ctx context.Context, tenantId, id uuid.UUID) error {
	panic("unexpected call to DeleteCronWorkflow")
}

func (f *fakeWorkflowScheduleRepository) UpdateCronWorkflow(ctx context.Context, tenantId, id uuid.UUID, opts *v1.UpdateCronOpts) error {
	panic("unexpected call to UpdateCronWorkflow")
}

func (f *fakeWorkflowScheduleRepository) DeleteInvalidCron(ctx context.Context, id uuid.UUID) error {
	panic("unexpected call to DeleteInvalidCron")
}

type repositoryWithWorkflowSchedules struct {
	v1.Repository
	workflowSchedules v1.WorkflowScheduleRepository
}

func (r repositoryWithWorkflowSchedules) WorkflowSchedules() v1.WorkflowScheduleRepository {
	return r.workflowSchedules
}

func newTestWorkflowService(repo v1.WorkflowScheduleRepository) *WorkflowService {
	return &WorkflowService{
		config: &server.ServerConfig{
			Layer: &database.Layer{
				V1: repositoryWithWorkflowSchedules{
					workflowSchedules: repo,
				},
			},
		},
	}
}

func newBulkDeleteContext(t *testing.T) echo.Context {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/test/workflows/scheduled/bulk-delete", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.Set("tenant", &sqlcv1.Tenant{ID: uuid.New()})

	return ctx
}

func TestWorkflowScheduledBulkDeleteReturns200WhenFilterMatchesNothing(t *testing.T) {
	repo := &fakeWorkflowScheduleRepository{}
	svc := newTestWorkflowService(repo)

	filter := gen.ScheduledWorkflowsBulkDeleteFilter{
		AdditionalMetadata: &[]string{"userId:123"},
	}

	resp, err := svc.WorkflowScheduledBulkDelete(newBulkDeleteContext(t), gen.WorkflowScheduledBulkDeleteRequestObject{
		Body: &gen.WorkflowScheduledBulkDeleteJSONRequestBody{
			Filter: &filter,
		},
	})

	require.NoError(t, err)

	success, ok := resp.(gen.WorkflowScheduledBulkDelete200JSONResponse)
	require.True(t, ok)
	require.Empty(t, success.DeletedIds)
	require.Empty(t, success.Errors)
	require.Equal(t, 1, repo.listCalls)
	require.Zero(t, repo.bulkDeleteCalls)
}

func TestWorkflowScheduledBulkDeleteReturns200WithErrorsWhenFilterFindsOnlyCodeDefinedRuns(t *testing.T) {
	repo := &fakeWorkflowScheduleRepository{
		listRows: []*sqlcv1.ListScheduledWorkflowsRow{
			{
				ID:     uuid.New(),
				Method: sqlcv1.WorkflowTriggerScheduledRefMethodsDEFAULT,
			},
		},
		listCount: 1,
	}
	svc := newTestWorkflowService(repo)

	filter := gen.ScheduledWorkflowsBulkDeleteFilter{}

	resp, err := svc.WorkflowScheduledBulkDelete(newBulkDeleteContext(t), gen.WorkflowScheduledBulkDeleteRequestObject{
		Body: &gen.WorkflowScheduledBulkDeleteJSONRequestBody{
			Filter: &filter,
		},
	})

	require.NoError(t, err)

	success, ok := resp.(gen.WorkflowScheduledBulkDelete200JSONResponse)
	require.True(t, ok)
	require.Empty(t, success.DeletedIds)
	require.Len(t, success.Errors, 1)
	require.Equal(t, "Cannot delete scheduled run created via code definition.", success.Errors[0].Error)
	require.Equal(t, 1, repo.listCalls)
	require.Zero(t, repo.bulkDeleteCalls)
}

func TestWorkflowScheduledBulkDeleteStillValidatesMissingIdsAndFilter(t *testing.T) {
	svc := newTestWorkflowService(&fakeWorkflowScheduleRepository{})

	resp, err := svc.WorkflowScheduledBulkDelete(newBulkDeleteContext(t), gen.WorkflowScheduledBulkDeleteRequestObject{
		Body: &gen.WorkflowScheduledBulkDeleteJSONRequestBody{},
	})

	require.NoError(t, err)

	badRequest, ok := resp.(gen.WorkflowScheduledBulkDelete400JSONResponse)
	require.True(t, ok)
	require.Len(t, badRequest.Errors, 1)
	require.Equal(t, "Provide scheduledWorkflowRunIds or filter.", badRequest.Errors[0].Description)
}
