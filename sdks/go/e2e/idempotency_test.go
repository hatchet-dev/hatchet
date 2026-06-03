//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	legacyclient "github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/stretchr/testify/require"
)

const (
	idempotencyEventKey        = "go-e2e:idempotency-example"
	idempotentTaskName         = "go-e2e-idempotent-task"
	idempotentShortWindowName  = "go-e2e-idempotent-task-short-window"
)

type idempotencyInput struct {
	ID string `json:"id"`
}

var (
	idempotencySetupOnce      sync.Once
	idempotencySetupErr       error
	idempotencyLegacyClient   legacyclient.Client
	idempotentTask            *hatchet.StandaloneTask
	idempotentTaskShortWindow *hatchet.StandaloneTask
)

func setupIdempotencyWorker(t *testing.T) (legacyclient.Client, *hatchet.StandaloneTask, *hatchet.StandaloneTask) {
	t.Helper()

	idempotencySetupOnce.Do(func() {
		idempotencyLegacyClient, idempotencySetupErr = legacyclient.New()
		if idempotencySetupErr != nil {
			return
		}

		idempotentTask = sharedClient.NewStandaloneTask(
			idempotentTaskName,
			func(ctx hatchet.Context, input idempotencyInput) (map[string]string, error) {
				return map[string]string{"result": fmt.Sprintf("Hello, world from task %s", input.ID)}, nil
			},
			hatchet.WithWorkflowIdempotency(hatchet.IdempotencyConfig{
				Expression: "input.id",
				TTL:        time.Minute,
			}),
			hatchet.WithWorkflowEvents(idempotencyEventKey),
		)

		idempotentTaskShortWindow = sharedClient.NewStandaloneTask(
			idempotentShortWindowName,
			func(ctx hatchet.Context, input idempotencyInput) (map[string]string, error) {
				return map[string]string{"result": fmt.Sprintf("Hello, world from task %s", input.ID)}, nil
			},
			hatchet.WithWorkflowIdempotency(hatchet.IdempotencyConfig{
				Expression: "input.id",
				TTL:        2 * time.Second,
			}),
		)

		worker, err := sharedClient.NewWorker(
			"e2e-idempotency-worker",
			hatchet.WithWorkflows(idempotentTask, idempotentTaskShortWindow),
		)
		if err != nil {
			idempotencySetupErr = err
			return
		}

		_, idempotencySetupErr = worker.Start()
		if idempotencySetupErr != nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		pollUntil(t, ctx, func() (bool, error) {
			if _, err := sharedClient.Workflows().Get(ctx, idempotentTask.GetName()); err != nil {
				return false, nil
			}
			if _, err := sharedClient.Workflows().Get(ctx, idempotentTaskShortWindow.GetName()); err != nil {
				return false, nil
			}

			return true, nil
		})
	})

	require.NoError(t, idempotencySetupErr)

	return idempotencyLegacyClient, idempotentTask, idempotentTaskShortWindow
}

func listRunsByTestRunID(ctx context.Context, testRunID string) (*rest.V1TaskSummaryList, error) {
	limit := int64(20)
	metadata := []string{"test_run_id:" + testRunID}

	return sharedClient.Runs().List(ctx, rest.V1WorkflowRunListParams{
		Since:              time.Now().Add(-5 * time.Minute),
		Limit:              &limit,
		AdditionalMetadata: &metadata,
		OnlyTasks:          false,
	})
}

func listEventsByTestRunID(ctx context.Context, legacy legacyclient.Client, testRunID string) (*rest.V1EventList, error) {
	limit := int64(10)
	metadata := []string{"test_run_id:" + testRunID}

	resp, err := legacy.API().V1EventListWithResponse(
		ctx,
		uuid.MustParse(legacy.TenantId()),
		&rest.V1EventListParams{
			Since:              &[]time.Time{time.Now().Add(-5 * time.Minute)}[0],
			Limit:              &limit,
			AdditionalMetadata: &metadata,
		},
	)
	if err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected event list status %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}

func TestIdempotencyDirectTrigger(t *testing.T) {
	_, idempotentTask, _ := setupIdempotencyWorker(t)
	ctx := newTestContext(t)
	testRunID := uniqueID()

	ref1, err := idempotentTask.RunNoWait(
		ctx,
		idempotencyInput{ID: testRunID},
		hatchet.WithRunMetadata(map[string]string{"test_run_id": testRunID}),
	)
	require.NoError(t, err)

	_, err = idempotentTask.RunNoWait(ctx, idempotencyInput{ID: testRunID})
	idempErr, ok := hatchet.IsIdempotencyCollisionError(err)
	require.True(t, ok)
	require.Equal(t, ref1.RunId, idempErr.ExistingRunExternalId)

	var runs *rest.V1TaskSummaryList
	pollUntil(t, ctx, func() (bool, error) {
		var pollErr error
		runs, pollErr = listRunsByTestRunID(ctx, testRunID)
		if pollErr != nil {
			return false, pollErr
		}
		return len(runs.Rows) == 1, nil
	})

	require.NotNil(t, runs)
	require.Len(t, runs.Rows, 1)
	require.Equal(t, ref1.RunId, runs.Rows[0].Metadata.Id)
}

func TestIdempotencyShortWindow(t *testing.T) {
	_, _, idempotentTaskShortWindow := setupIdempotencyWorker(t)
	ctx := newTestContext(t)
	testRunID := uniqueID()

	for i := 0; i < 4; i++ {
		if i == 1 {
			_, err := idempotentTaskShortWindow.RunNoWait(
				ctx,
				idempotencyInput{ID: testRunID},
				hatchet.WithRunMetadata(map[string]string{"test_run_id": testRunID}),
			)
			idempErr, ok := hatchet.IsIdempotencyCollisionError(err)
			require.True(t, ok)
			require.NotEmpty(t, idempErr.ExistingRunExternalId)
		} else {
			_, err := idempotentTaskShortWindow.RunNoWait(
				ctx,
				idempotencyInput{ID: testRunID},
				hatchet.WithRunMetadata(map[string]string{"test_run_id": testRunID}),
			)
			require.NoError(t, err)
		}

		if i != 3 {
			time.Sleep(time.Duration(float64(i)*float64(time.Second) + 1.5*float64(time.Second)))
		}
	}

	var runs *rest.V1TaskSummaryList
	pollUntil(t, ctx, func() (bool, error) {
		var pollErr error
		runs, pollErr = listRunsByTestRunID(ctx, testRunID)
		if pollErr != nil {
			return false, pollErr
		}
		return len(runs.Rows) >= 3, nil
	})

	require.NotNil(t, runs)
	require.Len(t, runs.Rows, 3)
}

func TestIdempotencyEventTrigger(t *testing.T) {
	legacy, _, _ := setupIdempotencyWorker(t)
	ctx := newTestContext(t)
	testRunID := uniqueID()

	err := sharedClient.Events().Push(
		ctx,
		idempotencyEventKey,
		idempotencyInput{ID: testRunID},
		legacyclient.WithEventMetadata(map[string]string{"test_run_id": testRunID}),
	)
	require.NoError(t, err)
	err = sharedClient.Events().Push(
		ctx,
		idempotencyEventKey,
		idempotencyInput{ID: testRunID},
		legacyclient.WithEventMetadata(map[string]string{"test_run_id": testRunID}),
	)
	require.NoError(t, err)

	var runs *rest.V1TaskSummaryList
	pollUntil(t, ctx, func() (bool, error) {
		var pollErr error
		runs, pollErr = listRunsByTestRunID(ctx, testRunID)
		if pollErr != nil {
			return false, pollErr
		}
		return len(runs.Rows) == 1, nil
	})

	require.NotNil(t, runs)
	require.Len(t, runs.Rows, 1)

	var events *rest.V1EventList
	pollUntil(t, ctx, func() (bool, error) {
		var pollErr error
		events, pollErr = listEventsByTestRunID(ctx, legacy, testRunID)
		if pollErr != nil {
			return false, pollErr
		}
		return events.Rows != nil && len(*events.Rows) == 2, nil
	})

	require.NotNil(t, events)
	require.Len(t, *events.Rows, 2)

	triggeredRunIDs := make(map[string]struct{})
	for _, event := range *events.Rows {
		for _, triggeredRun := range derefTriggeredRuns(event.TriggeredRuns) {
			triggeredRunIDs[triggeredRun.WorkflowRunId.String()] = struct{}{}
		}
	}

	require.Len(t, triggeredRunIDs, 1)

	var triggeredRunID string
	for runID := range triggeredRunIDs {
		triggeredRunID = runID
	}

	pollUntil(t, ctx, func() (bool, error) {
		status, pollErr := sharedClient.Runs().GetStatus(ctx, triggeredRunID)
		if pollErr != nil {
			return false, pollErr
		}
		return *status == rest.V1TaskStatusCOMPLETED, nil
	})
}

func derefTriggeredRuns(triggeredRuns *[]rest.V1EventTriggeredRun) []rest.V1EventTriggeredRun {
	if triggeredRuns == nil {
		return nil
	}

	return *triggeredRuns
}
