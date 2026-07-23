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

// These tests exercise the CEL display-name mechanism end-to-end against a live
// database: a workflow-level expression on the workflow version (evaluated in
// createDAGs -> v1_dag.display_name) and a per-step expression on the step
// (evaluated in insertTasks -> v1_task.display_name). Expressions are declared in
// the workflow/task definition and evaluated at trigger time against the run input.

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

	require.NoError(t, r.Tasks().UpdateTablePartitions(ctx))

	return tenantId
}

func strPtr(s string) *string { return &s }

func int32Ptr(i int32) *int32 { return &i }

// stepCfg describes a single task definition, optionally with a display-name expression.
type stepCfg struct {
	readableId  string
	action      string
	parents     []string
	displayExpr *string
}

// putWorkflow registers a workflow version with an optional workflow-level display-name
// expression, optional event triggers, and the given step definitions.
func putWorkflow(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	name string,
	workflowExpr *string,
	eventTriggers []string,
	steps []stepCfg,
) string {
	t.Helper()

	tasks := make([]repo.CreateStepOpts, 0, len(steps))
	for _, s := range steps {
		action := s.action
		if action == "" {
			action = "test:run"
		}
		tasks = append(tasks, repo.CreateStepOpts{
			ReadableId:  s.readableId,
			Action:      action,
			Parents:     s.parents,
			DisplayName: s.displayExpr,
		})
	}

	desc := "display-name test workflow"
	_, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:          name,
		Description:   &desc,
		DisplayName:   workflowExpr,
		EventTriggers: eventTriggers,
		Tasks:         tasks,
	})
	require.NoError(t, err)

	return name
}

// triggerRun drives the real manual-trigger path (NewTriggerTaskData ->
// PopulateExternalIds -> TriggerFromWorkflowNames) with the given raw run input.
func triggerRun(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	workflowName string,
	input string,
) ([]*repo.V1TaskWithPayload, []*repo.DAGWithData) {
	t.Helper()

	req := &v1contracts.TriggerWorkflowRequest{
		Name:  workflowName,
		Input: input,
	}

	ttd, err := r.Triggers().NewTriggerTaskData(ctx, tenantId, req, nil)
	require.NoError(t, err)

	opts := []*repo.WorkflowNameTriggerOpts{{TriggerTaskData: ttd}}

	err = r.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
	require.NoError(t, err)

	tasks, dags, _, _, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return tasks, dags
}

// triggerRunMany drives one TriggerFromWorkflowNames call carrying a per-item input,
// mirroring run_many / bulk trigger. Returns the created tasks.
func triggerRunMany(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	workflowName string,
	inputs []string,
) []*repo.V1TaskWithPayload {
	t.Helper()

	opts := make([]*repo.WorkflowNameTriggerOpts, 0, len(inputs))
	for _, input := range inputs {
		req := &v1contracts.TriggerWorkflowRequest{
			Name:  workflowName,
			Input: input,
		}
		ttd, err := r.Triggers().NewTriggerTaskData(ctx, tenantId, req, nil)
		require.NoError(t, err)
		opts = append(opts, &repo.WorkflowNameTriggerOpts{TriggerTaskData: ttd})
	}

	err := r.Triggers().PopulateExternalIdsForWorkflow(ctx, tenantId, opts)
	require.NoError(t, err)

	tasks, _, _, _, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return tasks
}

// triggerFromEvent drives the event-trigger path (prepareTriggerFromEvents +
// ListWorkflowsForEvents), returning the created tasks and DAGs.
func triggerFromEvent(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantId uuid.UUID,
	eventKey string,
	input string,
) (*repo.TriggerFromEventsResult, error) {
	t.Helper()

	opts := []repo.EventTriggerOpts{
		{
			ExternalId: uuid.New(),
			SeenAt:     time.Now(),
			Key:        eventKey,
			Data:       []byte(input),
		},
	}

	return r.Triggers().TriggerFromEvents(ctx, tenantId, opts)
}

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
	input string,
) (*repo.WorkflowNameTriggerOpts, []*repo.V1TaskWithPayload) {
	t.Helper()

	req := &v1contracts.TriggerWorkflowRequest{
		Name:       childWorkflowName,
		Input:      input,
		ChildIndex: int32Ptr(childIndex),
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

	tasks, _, _, _, err := r.Triggers().TriggerFromWorkflowNames(ctx, tenantId, opts)
	require.NoError(t, err)

	return opt, tasks
}

func TestTriggerDisplayName_NamesSingleTaskRunFromStepExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-single-step")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-single-step-wf", nil, nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.name")},
		})

		tasks, dags := triggerRun(t, ctx, r, tenantId, wf, `{"name":"Acme Corp"}`)

		require.Len(t, dags, 0, "single-task workflow creates no DAG")
		require.Len(t, tasks, 1)
		require.Equal(t, "Acme Corp", tasks[0].DisplayName, "the task IS the run, named by its step expression")

		return nil
	})
}

func TestTriggerDisplayName_NamesSingleTaskRunFromWorkflowExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-single-wf-expr")
		// Only a workflow-level expression; the single step has none.
		wf := putWorkflow(t, ctx, r, tenantId, "dn-single-wf-expr-wf", strPtr("input.name"), nil, []stepCfg{
			{readableId: "single-step"},
		})

		tasks, _ := triggerRun(t, ctx, r, tenantId, wf, `{"name":"Workflow Named"}`)

		require.Len(t, tasks, 1)
		require.Equal(t, "Workflow Named", tasks[0].DisplayName,
			"a single-task run inherits the workflow-level expression when the step has none")

		return nil
	})
}

func TestTriggerDisplayName_PrefersStepExprOverWorkflowExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-precedence")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-precedence-wf", strPtr("input.workflowName"), nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.stepName")},
		})

		tasks, _ := triggerRun(t, ctx, r, tenantId, wf, `{"workflowName":"WF","stepName":"STEP"}`)

		require.Len(t, tasks, 1)
		require.Equal(t, "STEP", tasks[0].DisplayName,
			"the step-level expression takes precedence over the workflow-level one on a single-task run")

		return nil
	})
}

func TestTriggerDisplayName_NamesDagRunFromWorkflowExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-dag-wf")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-dag-wf-wf", strPtr("input.name"), nil, []stepCfg{
			{readableId: "step-one"},
			{readableId: "step-two", parents: []string{"step-one"}},
		})

		// Ordinary top-level input; guards the finding-#1 raw-input shape: createDAGs
		// must unmarshal the bare run-input map, NOT a wrapped TaskInput.
		_, dags := triggerRun(t, ctx, r, tenantId, wf, `{"name":"Acme Corp"}`)

		require.Len(t, dags, 1, "multi-step workflow creates a DAG")
		require.Equal(t, "Acme Corp", dags[0].DisplayName,
			"the DAG row is named by the workflow-level expression against the raw run input")

		return nil
	})
}

func TestTriggerDisplayName_NamesDagStepsFromStepExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-dag-steps")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-dag-steps-wf", strPtr("input.name"), nil, []stepCfg{
			{readableId: "step-one", displayExpr: strPtr("'one-' + input.name")},
			{readableId: "step-two", parents: []string{"step-one"}, displayExpr: strPtr("'two-' + input.name")},
		})

		tasks, dags := triggerRun(t, ctx, r, tenantId, wf, `{"name":"Acme"}`)

		require.Len(t, dags, 1)
		require.Equal(t, "Acme", dags[0].DisplayName)

		// Only the root task (step-one) is inserted immediately; step-two is created
		// lazily once its parent completes. Assert every inserted task is named by its
		// own step expression, not a generated fallback.
		require.GreaterOrEqual(t, len(tasks), 1)
		for _, task := range tasks {
			require.Contains(t, []string{"one-Acme", "two-Acme"}, task.DisplayName,
				"each DAG step task is named by its own per-step expression")
		}

		return nil
	})
}

func TestTriggerDisplayName_FallsBackWhenNoExpr(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-none")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-none-wf", nil, nil, []stepCfg{
			{readableId: "single-step"},
		})

		tasks, _ := triggerRun(t, ctx, r, tenantId, wf, `{"name":"ignored"}`)

		require.Len(t, tasks, 1)
		require.Regexp(t, `^single-step-\d+$`, tasks[0].DisplayName,
			"no expression preserves the generated <readableId>-<unix> label")

		return nil
	})
}

func TestTriggerDisplayName_FallsBackOnMissingKey(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-missing")
		// DAG so we can assert both createDAGs and insertTasks tolerate an eval error.
		wf := putWorkflow(t, ctx, r, tenantId, "dn-missing-wf", strPtr("input.absent"), nil, []stepCfg{
			{readableId: "step-one", displayExpr: strPtr("input.alsoAbsent")},
			{readableId: "step-two", parents: []string{"step-one"}},
		})

		tasks, dags := triggerRun(t, ctx, r, tenantId, wf, `{"name":"present"}`)

		require.Len(t, dags, 1, "a missing-key expression must not fail DAG creation")
		require.Regexp(t, `^dn-missing-wf-\d+$`, dags[0].DisplayName,
			"the DAG falls back to <workflowName>-<unix> on eval error")

		require.GreaterOrEqual(t, len(tasks), 1, "a missing-key expression must not fail task insert")
		require.Regexp(t, `^step-one-\d+$`, tasks[0].DisplayName,
			"the task falls back to <readableId>-<unix> on eval error")

		return nil
	})
}

func TestTriggerDisplayName_FallsBackOnNonString(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-nonstring")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-nonstring-wf", nil, nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.count")},
		})

		tasks, _ := triggerRun(t, ctx, r, tenantId, wf, `{"count":42}`)

		require.Len(t, tasks, 1)
		require.Regexp(t, `^single-step-\d+$`, tasks[0].DisplayName,
			"a non-string result falls back to the generated name")

		return nil
	})
}

func TestTriggerDisplayName_NamesEventTriggeredDagRun(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-event")
		eventKey := "dn.event.triggered"
		putWorkflow(t, ctx, r, tenantId, "dn-event-wf", strPtr("input.name"), []string{eventKey}, []stepCfg{
			{readableId: "step-one"},
			{readableId: "step-two", parents: []string{"step-one"}},
		})

		// Finding-#2 fix: the event path must thread the workflow-level expression
		// through ListWorkflowsForEvents + prepareTriggerFromEvents.
		result, err := triggerFromEvent(t, ctx, r, tenantId, eventKey, `{"name":"EventRun"}`)
		require.NoError(t, err)

		require.Len(t, result.Dags, 1, "the event triggers the DAG")
		require.Equal(t, "EventRun", result.Dags[0].DisplayName,
			"an event-triggered DAG run is named by the workflow-level expression")

		return nil
	})
}

func TestTriggerDisplayName_DistinctNamesAcrossRunMany(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-many")
		// The original #4259 scenario: a fanned-out batch, each named from its own input.
		wf := putWorkflow(t, ctx, r, tenantId, "dn-many-wf", nil, nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.name")},
		})

		tasks := triggerRunMany(t, ctx, r, tenantId, wf,
			[]string{`{"name":"Alpha"}`, `{"name":"Bravo"}`, `{"name":"Charlie"}`})

		require.Len(t, tasks, 3)
		names := map[string]bool{}
		for _, task := range tasks {
			names[task.DisplayName] = true
		}
		require.Equal(t, map[string]bool{"Alpha": true, "Bravo": true, "Charlie": true}, names,
			"each run_many item is named from its own input, not an identical generated label")

		return nil
	})
}

func TestTriggerDisplayName_KeepsFirstNameOnChildReTrigger(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-child")

		// A real parent run to spawn children from.
		parentWf := putWorkflow(t, ctx, r, tenantId, "dn-child-parent", nil, nil, []stepCfg{
			{readableId: "single-step"},
		})
		parentTasks, _ := triggerRun(t, ctx, r, tenantId, parentWf, `{}`)
		require.Len(t, parentTasks, 1)
		parent := parentTasks[0]

		parentRow := &sqlcv1.FlattenExternalIdsRow{
			ID:            parent.ID,
			InsertedAt:    parent.InsertedAt,
			ExternalID:    parent.ExternalID,
			WorkflowRunID: parent.WorkflowRunID,
		}

		childWf := putWorkflow(t, ctx, r, tenantId, "dn-child-wf", nil, nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.name")},
		})

		// First spawn: child index 0, input names it "A" -> creates the child run.
		firstOpt, firstTasks := spawnChildOnce(t, ctx, r, tenantId, childWf, parentRow, 0, `{"name":"A"}`)
		require.False(t, firstOpt.ShouldSkip, "first spawn is not deduped")
		require.Len(t, firstTasks, 1)
		require.Equal(t, "A", firstTasks[0].DisplayName)
		firstChildId := firstOpt.ExternalId

		// Second spawn: SAME (parent, childIndex, childKey) but input names it "B".
		secondOpt, secondTasks := spawnChildOnce(t, ctx, r, tenantId, childWf, parentRow, 0, `{"name":"B"}`)
		require.True(t, secondOpt.ShouldSkip,
			"re-trigger with same (parent, childIndex, childKey) is deduped; the name is not part of the key")
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

func TestTriggerDisplayName_TruncatesTo255Runes(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-trunc")
		wf := putWorkflow(t, ctx, r, tenantId, "dn-trunc-wf", nil, nil, []stepCfg{
			{readableId: "single-step", displayExpr: strPtr("input.name")},
		})

		// "世" is a 3-byte rune; 300 of them would split mid-rune under byte truncation.
		long := strings.Repeat("世", 300)
		tasks, _ := triggerRun(t, ctx, r, tenantId, wf, fmt.Sprintf(`{"name":%q}`, long))

		require.Len(t, tasks, 1)
		require.Equal(t, 255, utf8.RuneCountInString(tasks[0].DisplayName),
			"the evaluated display name is truncated to 255 runes")
		require.True(t, utf8.ValidString(tasks[0].DisplayName), "no multibyte rune was split")

		return nil
	})
}

func TestTriggerDisplayName_RejectsInvalidExprAtRegistration(t *testing.T) {
	runTriggerTest(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		r := conf.V1
		tenantId := createTriggerTenant(t, ctx, r, "dn-invalid")

		desc := "invalid expr"
		_, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "dn-invalid-wf",
			Description: &desc,
			DisplayName: strPtr("this is not (valid CEL"),
			Tasks: []repo.CreateStepOpts{
				{ReadableId: "single-step", Action: "test:run"},
			},
		})
		require.Error(t, err, "a malformed display-name expression is rejected at registration")

		return nil
	})
}
