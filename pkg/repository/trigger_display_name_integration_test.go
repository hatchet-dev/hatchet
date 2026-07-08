//go:build integration

package repository_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func runTriggerTest(t *testing.T, test func(conf *database.Layer) error) {
	t.Helper()
	// `internal/testutils.Prepare` constructs a server config and requires a RabbitMQ URL.
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")
	testutils.RunTestWithDatabase(t, test)
}

// createTriggerTenant creates a fresh tenant so each test's rows are isolated.
func createTriggerTenant(t *testing.T, ctx context.Context, r repo.Repository, name string) uuid.UUID {
	t.Helper()

	tenantId := uuid.New()
	_, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
		ID:   &tenantId,
		Name: name,
		Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
	})
	require.NoError(t, err)

	return tenantId
}

// putSingleTaskWorkflow registers a single-task (non-DAG) workflow and returns its name.
func putSingleTaskWorkflow(t *testing.T, ctx context.Context, r repo.Repository, tenantId uuid.UUID, name string) string {
	t.Helper()

	desc := "display-name test workflow"
	_, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Tasks: []repo.CreateStepOpts{
			{ReadableId: "single-step", Action: "test:run"},
		},
	})
	require.NoError(t, err)

	return name
}

// putDagWorkflow registers a multi-step (DAG) workflow: step-two depends on step-one.
func putDagWorkflow(t *testing.T, ctx context.Context, r repo.Repository, tenantId uuid.UUID, name string) string {
	t.Helper()

	desc := "display-name DAG test workflow"
	_, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Tasks: []repo.CreateStepOpts{
			{ReadableId: "step-one", Action: "test:run"},
			{ReadableId: "step-two", Action: "test:run", Parents: []string{"step-one"}},
		},
	})
	require.NoError(t, err)

	return name
}

// triggerRun drives the real trigger path (NewTriggerTaskData -> PopulateExternalIds ->
// TriggerFromWorkflowNames) with the given display name and returns the created rows.
func triggerRun(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	workflowName string,
	displayName *string,
) ([]*repo.V1TaskWithPayload, []*repo.DAGWithData) {
	t.Helper()

	req := &v1contracts.TriggerWorkflowRequest{
		Name:        workflowName,
		Input:       "{}",
		DisplayName: displayName,
	}

	ttd, err := r.Triggers().NewTriggerTaskData(ctx, tenantId, req, nil)
	require.NoError(t, err)

	opts := []*repo.WorkflowNameTriggerOpts{{TriggerTaskData: ttd}}

	err = r.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
	require.NoError(t, err)

	tasks, dags, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return tasks, dags
}

// triggerRunMany drives one TriggerFromWorkflowNames call carrying a per-item display
// name for each element, mirroring run_many / bulk trigger. Returns the created tasks.
func triggerRunMany(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	workflowName string,
	displayNames []*string,
) []*repo.V1TaskWithPayload {
	t.Helper()

	opts := make([]*repo.WorkflowNameTriggerOpts, 0, len(displayNames))
	for _, dn := range displayNames {
		req := &v1contracts.TriggerWorkflowRequest{
			Name:        workflowName,
			Input:       "{}",
			DisplayName: dn,
		}
		ttd, err := r.Triggers().NewTriggerTaskData(ctx, tenantId, req, nil)
		require.NoError(t, err)
		opts = append(opts, &repo.WorkflowNameTriggerOpts{TriggerTaskData: ttd})
	}

	err := r.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
	require.NoError(t, err)

	tasks, _, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return tasks
}

func strPtr(s string) *string { return &s }

func int32Ptr(i int32) *int32 { return &i }

// spawnChildOnce runs one non-durable child spawn round: build the trigger data with the
// parent + child index, populate external ids (which dedups on the child spawn key), and
// trigger only if the opt was not deduped. Returns the (mutated) opt and any created tasks.
func spawnChildOnce(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	childWorkflowName string,
	parent *sqlcv1.FlattenExternalIdsRow,
	childIndex int32,
	displayName *string,
) (*repo.WorkflowNameTriggerOpts, []*repo.V1TaskWithPayload) {
	t.Helper()

	req := &v1contracts.TriggerWorkflowRequest{
		Name:        childWorkflowName,
		Input:       "{}",
		DisplayName: displayName,
		ChildIndex:  int32Ptr(childIndex),
	}

	ttd, err := r.Triggers().NewTriggerTaskData(ctx, tenantId, req, parent)
	require.NoError(t, err)

	opt := &repo.WorkflowNameTriggerOpts{TriggerTaskData: ttd}
	opts := []*repo.WorkflowNameTriggerOpts{opt}

	err = r.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
	require.NoError(t, err)

	if opt.ShouldSkip {
		return opt, nil
	}

	tasks, _, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return opt, tasks
}

func TestTriggerDisplayName_KeepsFirstNameOnChildReTrigger(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-child")

		// A real parent run to spawn children from.
		parentWf := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-child-parent")
		parentTasks, _ := triggerRun(t, ctx, r, tenantId, parentWf, strPtr("Parent"))
		require.Len(t, parentTasks, 1)
		parent := parentTasks[0]

		parentRow := &sqlcv1.FlattenExternalIdsRow{
			ID:            parent.ID,
			InsertedAt:    parent.InsertedAt,
			ExternalID:    parent.ExternalID,
			WorkflowRunID: parent.WorkflowRunID,
		}

		childWf := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-child-wf")

		// First spawn: child index 0, display name "A" -> creates the child run.
		firstOpt, firstTasks := spawnChildOnce(t, ctx, r, tenantId, childWf, parentRow, 0, strPtr("A"))
		require.False(t, firstOpt.ShouldSkip, "first spawn is not deduped")
		require.Len(t, firstTasks, 1)
		require.Equal(t, "A", firstTasks[0].DisplayName)
		firstChildId := firstOpt.ExternalId

		// Second spawn: SAME (parent, childIndex, childKey) but display name "B".
		secondOpt, secondTasks := spawnChildOnce(t, ctx, r, tenantId, childWf, parentRow, 0, strPtr("B"))
		require.True(t, secondOpt.ShouldSkip,
			"re-trigger with same (parent, childIndex, childKey) is deduped; display name is not part of the key")
		require.Equal(t, firstChildId, secondOpt.ExternalId, "reuses the first child's external id")
		require.Nil(t, secondTasks, "no new run is created on re-trigger")

		// The existing child run keeps its original name "A" (never rewritten to "B").
		var storedName string
		err := conf.Pool.QueryRow(ctx,
			`SELECT display_name FROM v1_task WHERE tenant_id = $1 AND external_id = $2`,
			tenantId, firstChildId,
		).Scan(&storedName)
		require.NoError(t, err)
		require.Equal(t, "A", storedName, "first display name wins on re-trigger")

		return nil
	})
}

func TestTriggerDisplayName_NamesSingleTaskRun(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-single")
		workflowName := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-single-wf")

		tasks, dags := triggerRun(t, ctx, r, tenantId, workflowName, strPtr("Acme Corp"))

		require.Len(t, dags, 0, "single-task workflow creates no DAG")
		require.Len(t, tasks, 1)
		require.Equal(t, "Acme Corp", tasks[0].DisplayName, "the task IS the run, so it carries the display name")

		return nil
	})
}

func TestTriggerDisplayName_NamesDagRunNotSteps(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-dag")
		workflowName := putDagWorkflow(t, ctx, r, tenantId, "dn-dag-wf")

		tasks, dags := triggerRun(t, ctx, r, tenantId, workflowName, strPtr("Acme Corp"))

		require.Len(t, dags, 1, "multi-step workflow creates a DAG")
		require.Equal(t, "Acme Corp", dags[0].DisplayName, "the DAG row carries the display name")

		for _, task := range tasks {
			require.Regexp(t, `^step-(one|two)-\d+$`, task.DisplayName,
				"DAG step tasks keep their generated <readableId>-<unix> names")
		}

		return nil
	})
}

func TestTriggerDisplayName_FallsBackWhenNil(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-nil")
		workflowName := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-nil-wf")

		tasks, _ := triggerRun(t, ctx, r, tenantId, workflowName, nil)

		require.Len(t, tasks, 1)
		require.Regexp(t, `^single-step-\d+$`, tasks[0].DisplayName,
			"omitting display name preserves the generated <readableId>-<unix> label")

		return nil
	})
}

func TestTriggerDisplayName_TreatsWhitespaceAsUnset(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-ws")
		workflowName := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-ws-wf")

		tasks, _ := triggerRun(t, ctx, r, tenantId, workflowName, strPtr("   "))

		require.Len(t, tasks, 1)
		require.Regexp(t, `^single-step-\d+$`, tasks[0].DisplayName,
			"whitespace-only display name is treated as unset")

		return nil
	})
}

func TestTriggerDisplayName_TruncatesTo255Runes(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-trunc")
		workflowName := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-trunc-wf")

		// "世" is a 3-byte rune; 300 of them would split mid-rune under byte truncation.
		long := strings.Repeat("世", 300)
		tasks, _ := triggerRun(t, ctx, r, tenantId, workflowName, strPtr(long))

		require.Len(t, tasks, 1)
		require.Equal(t, 255, utf8.RuneCountInString(tasks[0].DisplayName),
			"stored display name is truncated to 255 runes")
		require.True(t, utf8.ValidString(tasks[0].DisplayName), "no multibyte rune was split")

		return nil
	})
}

func TestTriggerDisplayName_DistinctNamePerRunMany(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-many")
		workflowName := putSingleTaskWorkflow(t, ctx, r, tenantId, "dn-many-wf")

		tasks := triggerRunMany(t, ctx, r, tenantId, workflowName,
			[]*string{strPtr("Alpha"), strPtr("Bravo"), strPtr("Charlie")})

		require.Len(t, tasks, 3)
		names := map[string]bool{}
		for _, task := range tasks {
			names[task.DisplayName] = true
		}
		require.Equal(t, map[string]bool{"Alpha": true, "Bravo": true, "Charlie": true}, names,
			"each run_many item carries its own display name")

		return nil
	})
}
