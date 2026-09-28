//go:build integration

package repository_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// TestCreateTasks_WritesPayloadAndQueueItem checks the rows CreateTasks writes alongside the
// task: exactly one TASK_INPUT payload row holding the input, and, for a queued task without a
// concurrency strategy, exactly one queue item that matches the one returned with the task.
func TestCreateTasks_WritesPayloadAndQueueItem(t *testing.T) {
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "create-tasks-rows",
			Slug: fmt.Sprintf("create-tasks-rows-%s", tenantId.String()),
		})
		require.NoError(t, err)

		input := map[string]interface{}{"hello": "world", "n": float64(1)}
		inputBytes, err := json.Marshal(input)
		require.NoError(t, err)

		t.Run("queued task without concurrency", func(t *testing.T) {
			task := triggerSingleTask(t, ctx, conf, tenantId, "plain-workflow", nil, inputBytes)

			// v1_task.input is always '{}': the payload row is the only copy of the input
			require.JSONEq(t, `{}`, string(task.Input))

			stored := listQueueItems(t, ctx, conf, task)
			require.Len(t, stored, 1, "exactly one queue item for the task")
			require.NotNil(t, task.QueueItem, "CreateTasks returns the queue item it wrote")
			require.Equal(t, *stored[0], *task.QueueItem, "returned queue item matches the stored row")

			qi := task.QueueItem
			require.Equal(t, task.TenantID, qi.TenantID)
			require.Equal(t, task.Queue, qi.Queue)
			require.Equal(t, task.ID, qi.TaskID)
			require.Equal(t, task.InsertedAt, qi.TaskInsertedAt)
			require.Equal(t, task.ExternalID, qi.ExternalID)
			require.Equal(t, task.ActionID, qi.ActionID)
			require.Equal(t, task.StepID, qi.StepID)
			require.Equal(t, task.WorkflowID, qi.WorkflowID)
			require.Equal(t, task.WorkflowRunID, qi.WorkflowRunID)
			require.Equal(t, task.StepTimeout, qi.StepTimeout)
			require.Equal(t, task.Sticky, qi.Sticky)
			require.Equal(t, task.DesiredWorkerID, qi.DesiredWorkerID)
			require.Equal(t, task.RetryCount, qi.RetryCount)
			require.Equal(t, task.DesiredWorkerLabel, qi.DesiredWorkerLabel)
			require.Equal(t, task.BatchKey, qi.BatchKey)
			require.True(t, qi.ScheduleTimeoutAt.Valid, "schedule_timeout_at is set from the task's schedule timeout")
			require.True(t, qi.ScheduleTimeoutAt.Time.After(task.InsertedAt.Time), "schedule_timeout_at is in the future of the insert")

			if task.Priority.Valid {
				require.Equal(t, task.Priority.Int32, qi.Priority)
			} else {
				require.Equal(t, int32(1), qi.Priority, "a task without a priority gets the trigger's default of 1")
			}

			assertSinglePayloadRow(t, ctx, conf, task, input)
		})

		t.Run("plain insert takes the trigger path and writes the same queue item", func(t *testing.T) {
			// CreateTasks writes its queue items through store_queue_items_for_tasks and the
			// v1_task insert trigger writes them through the same function for every other
			// writer of v1_task: a copy of a CreateTasks task inserted with a plain INSERT
			// must end up with an identical queue item, written exactly once by the trigger.
			viaStatement := triggerSingleTask(t, ctx, conf, tenantId, "trigger-path-workflow", nil, inputBytes)
			require.NotNil(t, viaStatement.QueueItem)

			plainExternalId := uuid.New()

			var (
				plainId         int64
				plainInsertedAt pgtype.Timestamptz
			)

			require.NoError(t, conf.Pool.QueryRow(ctx, `
				INSERT INTO v1_task (
					tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, workflow_version_id, workflow_run_id,
					schedule_timeout, step_timeout, priority, sticky, desired_worker_id, external_id, display_name, input,
					retry_count, step_index, initial_state, batch_key, desired_worker_label
				)
				SELECT
					tenant_id, queue, action_id, step_id, step_readable_id, workflow_id, workflow_version_id, $3,
					schedule_timeout, step_timeout, priority, sticky, desired_worker_id, $3, display_name, input,
					retry_count, step_index, initial_state, batch_key, desired_worker_label
				FROM v1_task
				WHERE id = $1 AND inserted_at = $2
				RETURNING id, inserted_at`,
				viaStatement.ID, viaStatement.InsertedAt, plainExternalId,
			).Scan(&plainId, &plainInsertedAt))

			viaTrigger := listQueueItemsByKey(t, ctx, conf, plainId, plainInsertedAt, viaStatement.RetryCount)
			require.Len(t, viaTrigger, 1, "the trigger writes exactly one queue item for a plain insert")

			// the trigger's other work still runs
			var lookupTaskId int64
			require.NoError(t, conf.Pool.QueryRow(ctx, `SELECT task_id FROM v1_lookup_table WHERE external_id = $1`, plainExternalId).Scan(&lookupTaskId))
			require.Equal(t, plainId, lookupTaskId)

			expected := *viaStatement.QueueItem
			actual := *viaTrigger[0]

			// the identity columns differ by construction ...
			require.Equal(t, plainId, actual.TaskID)
			require.Equal(t, plainInsertedAt, actual.TaskInsertedAt)
			require.Equal(t, plainExternalId, actual.ExternalID)
			require.Equal(t, plainExternalId, actual.WorkflowRunID)
			require.NotEqual(t, expected.ID, actual.ID)

			// ... and schedule_timeout_at is the transaction timestamp plus the schedule
			// timeout, where the transaction timestamp is also the task's inserted_at, so the
			// two offsets must agree
			expectedOffset := expected.ScheduleTimeoutAt.Time.Sub(expected.TaskInsertedAt.Time)
			actualOffset := actual.ScheduleTimeoutAt.Time.Sub(actual.TaskInsertedAt.Time)
			require.InDelta(t, expectedOffset.Seconds(), actualOffset.Seconds(), 0.01, "both paths set schedule_timeout_at from the schedule timeout")

			// every other column must be identical
			actual.ID = expected.ID
			actual.TaskID = expected.TaskID
			actual.TaskInsertedAt = expected.TaskInsertedAt
			actual.ExternalID = expected.ExternalID
			actual.WorkflowRunID = expected.WorkflowRunID
			actual.ScheduleTimeoutAt = expected.ScheduleTimeoutAt
			require.Equal(t, expected, actual, "the trigger path and the CreateTasks path write the same queue item")
		})

		t.Run("task with a concurrency strategy has no queue item", func(t *testing.T) {
			maxRuns := int32(1)
			strategy := "GROUP_ROUND_ROBIN"

			task := triggerSingleTask(t, ctx, conf, tenantId, "concurrency-workflow", []repo.CreateConcurrencyOpts{
				{
					Expression:    "input.hello",
					MaxRuns:       &maxRuns,
					LimitStrategy: &strategy,
				},
			}, inputBytes)

			require.Nil(t, task.QueueItem, "a task waiting on a concurrency slot carries no queue item")
			require.Empty(t, listQueueItems(t, ctx, conf, task), "no queue item is written until a slot is granted")

			assertSinglePayloadRow(t, ctx, conf, task, input)
		})

		t.Run("event trigger writes the event payload", func(t *testing.T) {
			eventKey := "create-tasks-rows:event"
			desc := "event-triggered workflow"
			_, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
				Name:          "event-workflow",
				Description:   &desc,
				EventTriggers: []string{eventKey},
				Tasks: []repo.CreateStepOpts{
					{
						ReadableId: "my-task",
						Action:     "test:run",
					},
				},
			})
			require.NoError(t, err)

			withData := repo.EventTriggerOpts{ExternalId: uuid.New(), SeenAt: time.Now().UTC(), Key: eventKey, Data: inputBytes}
			// an empty event input writes no payload row, the rule Store applied before the fold
			empty := repo.EventTriggerOpts{ExternalId: uuid.New(), SeenAt: time.Now().UTC(), Key: eventKey, Data: []byte(`{}`)}

			result, err := conf.V1.Triggers().TriggerFromEvents(ctx, tenantId, []repo.EventTriggerOpts{withData, empty})
			require.NoError(t, err)
			require.Len(t, result.Tasks, 2, "each event triggers the workflow once")

			fromWithData := 0

			for _, task := range result.Tasks {
				require.NotNil(t, task.QueueItem)
				require.Len(t, listQueueItems(t, ctx, conf, task), 1)

				if task.TriggeringEventExternalID != nil && *task.TriggeringEventExternalID == withData.ExternalId {
					fromWithData++
					assertSinglePayloadRow(t, ctx, conf, task, input)
				}
			}

			require.Equal(t, 1, fromWithData, "one task carries the event's input")

			require.JSONEq(t, string(inputBytes), string(listEventPayloads(t, ctx, conf, tenantId, withData)[0]), "the event input is stored as USER_EVENT_INPUT")
			require.Empty(t, listEventPayloads(t, ctx, conf, tenantId, empty), "an empty event input writes no payload row")
		})

		return nil
	})
}

// listEventPayloads returns the USER_EVENT_INPUT payload contents stored for the event with the
// given identity (there is at most one), after checking the event row itself exists.
func listEventPayloads(t *testing.T, ctx context.Context, conf *database.Layer, tenantId uuid.UUID, event repo.EventTriggerOpts) [][]byte {
	t.Helper()

	rows, err := conf.Pool.Query(ctx, `
		SELECT e.id, p.location::text, p.external_location_key, p.inline_content
		FROM v1_event e
		LEFT JOIN v1_payload p ON p.tenant_id = e.tenant_id AND p.id = e.id AND p.inserted_at = e.seen_at AND p.type = 'USER_EVENT_INPUT'
		WHERE e.tenant_id = $1 AND e.external_id = $2 AND e.seen_at = $3`,
		tenantId, event.ExternalId, event.SeenAt,
	)
	require.NoError(t, err)
	defer rows.Close()

	payloads := make([][]byte, 0, 1)
	events := 0

	for rows.Next() {
		events++

		var (
			eventId             int64
			location            *string
			externalLocationKey *string
			inlineContent       []byte
		)

		require.NoError(t, rows.Scan(&eventId, &location, &externalLocationKey, &inlineContent))

		if location == nil {
			continue
		}

		require.Equal(t, "INLINE", *location)
		require.Nil(t, externalLocationKey)
		payloads = append(payloads, inlineContent)
	}

	require.NoError(t, rows.Err())
	require.Equal(t, 1, events, "exactly one v1_event row for the event")

	return payloads
}

// triggerSingleTask registers a one-task workflow (with the given concurrency entries, if
// any) and triggers it once with the given input, returning the created task.
func triggerSingleTask(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	tenantId uuid.UUID,
	name string,
	concurrency []repo.CreateConcurrencyOpts,
	input []byte,
) *repo.V1TaskWithPayload {
	t.Helper()

	desc := "test workflow"
	_, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Tasks: []repo.CreateStepOpts{
			{
				ReadableId:  "my-task",
				Action:      "test:run",
				Concurrency: concurrency,
			},
		},
	})
	require.NoError(t, err)

	tasks, dags, _, _, err := conf.V1.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*repo.WorkflowNameTriggerOpts{
		{
			TriggerTaskData: &repo.TriggerTaskData{
				WorkflowName: name,
				Data:         input,
			},
			ExternalId: uuid.New(),
		},
	})
	require.NoError(t, err)
	require.Empty(t, dags, "a single-task workflow creates no DAG")
	require.Len(t, tasks, 1)

	return tasks[0]
}

func listQueueItems(t *testing.T, ctx context.Context, conf *database.Layer, task *repo.V1TaskWithPayload) []*sqlcv1.V1QueueItem {
	t.Helper()

	return listQueueItemsByKey(t, ctx, conf, task.ID, task.InsertedAt, task.RetryCount)
}

func listQueueItemsByKey(t *testing.T, ctx context.Context, conf *database.Layer, taskId int64, taskInsertedAt pgtype.Timestamptz, retryCount int32) []*sqlcv1.V1QueueItem {
	t.Helper()

	rows, err := conf.Pool.Query(ctx, `
		SELECT id, tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id,
			schedule_timeout_at, step_timeout, priority, sticky, desired_worker_id, retry_count, desired_worker_label, batch_key
		FROM v1_queue_item
		WHERE task_id = $1 AND task_inserted_at = $2 AND retry_count = $3`,
		taskId, taskInsertedAt, retryCount,
	)
	require.NoError(t, err)
	defer rows.Close()

	items := make([]*sqlcv1.V1QueueItem, 0, 1)

	for rows.Next() {
		var qi sqlcv1.V1QueueItem
		require.NoError(t, rows.Scan(
			&qi.ID, &qi.TenantID, &qi.Queue, &qi.TaskID, &qi.TaskInsertedAt, &qi.ExternalID, &qi.ActionID, &qi.StepID, &qi.WorkflowID, &qi.WorkflowRunID,
			&qi.ScheduleTimeoutAt, &qi.StepTimeout, &qi.Priority, &qi.Sticky, &qi.DesiredWorkerID, &qi.RetryCount, &qi.DesiredWorkerLabel, &qi.BatchKey,
		))
		items = append(items, &qi)
	}

	require.NoError(t, rows.Err())

	return items
}

// assertSinglePayloadRow checks that exactly one TASK_INPUT payload row exists for the task,
// that it is inline, and that its content is the step run data wrapping the given input (as
// JSON: inline_content is JSONB, so the stored bytes are Postgres' rendering).
func assertSinglePayloadRow(t *testing.T, ctx context.Context, conf *database.Layer, task *repo.V1TaskWithPayload, input map[string]interface{}) {
	t.Helper()

	rows, err := conf.Pool.Query(ctx, `
		SELECT external_id, location::text, external_location_key, inline_content
		FROM v1_payload
		WHERE tenant_id = $1 AND id = $2 AND inserted_at = $3 AND type = 'TASK_INPUT'`,
		task.TenantID, task.ID, task.InsertedAt,
	)
	require.NoError(t, err)
	defer rows.Close()

	count := 0

	for rows.Next() {
		count++

		var (
			externalId          uuid.UUID
			location            string
			externalLocationKey *string
			inlineContent       []byte
		)

		require.NoError(t, rows.Scan(&externalId, &location, &externalLocationKey, &inlineContent))
		require.Equal(t, task.ExternalID, externalId)
		require.Equal(t, "INLINE", location)
		require.Nil(t, externalLocationKey)

		// the stored content is what the engine holds in memory for the task ...
		require.JSONEq(t, string(task.Payload), string(inlineContent))

		// ... and its input field is the input the trigger was given
		var stepRunData struct {
			Input map[string]interface{} `json:"input"`
		}
		require.NoError(t, json.Unmarshal(inlineContent, &stepRunData))
		require.Equal(t, input, stepRunData.Input)
	}

	require.NoError(t, rows.Err())
	require.Equal(t, 1, count, "exactly one TASK_INPUT payload row for the task")
}
