package workflows

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type pausedControls struct {
	repository.TenantControlRepository
}

func (pausedControls) Admit(uuid.UUID, int) error { return repository.ErrTenantPaused }

type pausedRepositoryStub struct {
	repository.Repository
}

func (pausedRepositoryStub) TenantControl() repository.TenantControlRepository {
	return pausedControls{}
}

// Triggering a cron or a scheduled run now creates a run now, so a paused tenant is refused before
// anything is published: the config has no message queue, which publishing would need.
func TestTriggerNowIsRefusedForAPausedTenant(t *testing.T) {
	svc := &WorkflowService{config: &server.ServerConfig{Layer: &database.Layer{V1: pausedRepositoryStub{}}}}
	tenantID := uuid.New()

	newContext := func() echo.Context {
		ctx := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
		ctx.Set("tenant", &sqlcv1.Tenant{ID: tenantID})
		ctx.Set("cron-workflow", &sqlcv1.ListCronWorkflowsRow{})
		ctx.Set("scheduled-workflow-run", &sqlcv1.ListScheduledWorkflowsRow{TenantId: tenantID})

		return ctx
	}

	_, err := svc.WorkflowCronTrigger(newContext(), gen.WorkflowCronTriggerRequestObject{})
	require.ErrorIs(t, err, repository.ErrTenantPaused)

	_, err = svc.WorkflowScheduledTrigger(newContext(), gen.WorkflowScheduledTriggerRequestObject{})
	require.ErrorIs(t, err, repository.ErrTenantPaused)
}
