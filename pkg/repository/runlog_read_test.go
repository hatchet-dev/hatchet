//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// A task that finished after midnight has its bundle on the day it was created, while its output row is
// stored under the next day. Once both days are dropped, the readers serve the run from the log exactly
// as core served it.
func TestRunLogReader_ServesARunFinishedOnTheNextDay(t *testing.T) {
	f := newRunLogFixture(t)
	f.repo.payloadStore.(*payloadStoreRepositoryImpl).runLog = f.repo.runLog

	day := f.daysAgo(4)
	next := day.AddDate(0, 0, 1)

	task := f.insertTask(fixtureTask{insertedAt: day.Add(23 * time.Hour)})
	f.exec(`DELETE FROM v1_queue_item WHERE task_id = $1 AND task_inserted_at = $2`, task.ID, task.InsertedAt)

	finishedAt := next.Add(time.Hour)
	ref := uuidV7At(finishedAt)

	rows, err := f.pool.Query(f.ctx, `
		INSERT INTO v1_task_event (inserted_at, tenant_id, task_id, task_inserted_at, retry_count, event_type, payload_reference_id)
		VALUES ($1, $2, $3, $4, 0, 'COMPLETED', $5)
		RETURNING *
	`, finishedAt, f.tenant, task.ID, task.InsertedAt, ref)
	require.NoError(t, err)

	event, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[sqlcv1.V1TaskEvent])
	require.NoError(t, err)

	output := (&TaskOutputEvent{
		EventType:      sqlcv1.V1TaskEventTypeCOMPLETED,
		TaskExternalId: task.ExternalID,
		TaskId:         task.ID,
		StepReadableID: task.StepReadableID,
		Output:         []byte(`{"v": 42}`),
	}).Bytes()

	f.storePayload(taskEventDataStoreOpts(f.tenant, event, output))

	// how a dashboard read through OLAP asks for the output: by reference, at the OLAP event's time
	outputOpt := RetrievePayloadOpts{
		TenantId:           f.tenant,
		InsertedAt:         event.InsertedAt,
		Type:               sqlcv1.V1PayloadTypeTASKEVENTDATA,
		ExternalId:         ref,
		PayloadReferenceId: &ref,
	}

	detailsBefore, err := f.repo.GetWorkflowRunResultDetails(f.ctx, f.tenant, task.ExternalID)
	require.NoError(t, err)
	require.NotNil(t, detailsBefore)

	finalizedBefore, err := f.repo.ListFinalizedWorkflowRuns(f.ctx, f.tenant, []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Len(t, finalizedBefore, 1)

	require.NoError(t, f.maintain())

	require.Equal(t, runLogStateDropped, f.dayState(day))
	require.Equal(t, runLogStateDropped, f.dayState(next))
	require.False(t, f.partitionExists(partitionName("v1_payload", next)), "the output row must be gone from core")

	found, err := f.repo.payloadStore.Retrieve(f.ctx, nil, outputOpt)
	require.NoError(t, err)
	assert.JSONEq(t, string(output), string(found[outputOpt]))

	detailsAfter, err := f.repo.GetWorkflowRunResultDetails(f.ctx, f.tenant, task.ExternalID)
	require.NoError(t, err)
	assert.Equal(t, detailsBefore, detailsAfter)
	assert.JSONEq(t, `{"v": 42}`, string(detailsAfter.ReadableIdToDetails[StepReadableId(task.StepReadableID)].OutputPayload))

	finalizedAfter, err := f.repo.ListFinalizedWorkflowRuns(f.ctx, f.tenant, []uuid.UUID{task.ExternalID, uuid.New()})
	require.NoError(t, err)
	assert.JSONEq(t, mustJSON(t, finalizedBefore), mustJSON(t, finalizedAfter))

	archived, err := f.repo.ListArchivedExternalIds(f.ctx, f.tenant, []uuid.UUID{task.ExternalID, uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{task.ExternalID}, archived)

	// without the run log the dropped run is gone, as before
	f.repo.runLog = nil
	f.repo.payloadStore.(*payloadStoreRepositoryImpl).runLog = nil

	details, err := f.repo.GetWorkflowRunResultDetails(f.ctx, f.tenant, task.ExternalID)
	require.NoError(t, err)
	assert.Nil(t, details)

	found, err = f.repo.payloadStore.Retrieve(f.ctx, nil, outputOpt)
	require.NoError(t, err)
	assert.Empty(t, found)
}
