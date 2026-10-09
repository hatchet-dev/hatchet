//go:build !e2e && !load && !rampup && !integration

package workflowruns

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestActiveCountOptsStatuses(t *testing.T) {
	onWorker := gen.ONWORKER
	evicted := gen.EVICTED

	cases := []struct {
		name          string
		statuses      *[]gen.V1TaskStatus
		runningFilter *gen.V1RunningFilter
		want          []sqlcv1.V1ReadableStatusOlap
	}{
		{
			name: "defaults to queued and running",
			want: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:     "empty list defaults to queued and running",
			statuses: &[]gen.V1TaskStatus{},
			want: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:     "drops finished statuses",
			statuses: &[]gen.V1TaskStatus{gen.V1TaskStatusCOMPLETED, gen.V1TaskStatusQUEUED, gen.V1TaskStatusFAILED},
			want:     []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapQUEUED},
		},
		{
			name:          "running on worker only",
			statuses:      &[]gen.V1TaskStatus{gen.V1TaskStatusRUNNING},
			runningFilter: &onWorker,
			want:          []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapRUNNING},
		},
		{
			name:          "running evicted only",
			runningFilter: &evicted,
			want: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapEVICTED,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, ok := activeCountOpts(gen.V1WorkflowRunActiveCountParams{
				Since:         time.Now().Add(-time.Hour),
				Before:        time.Now(),
				Statuses:      tc.statuses,
				RunningFilter: tc.runningFilter,
			}, false)

			require.True(t, ok)
			assert.ElementsMatch(t, tc.want, opts.Statuses)
		})
	}
}

func TestActiveCountOptsOnlyFinishedStatuses(t *testing.T) {
	_, ok := activeCountOpts(gen.V1WorkflowRunActiveCountParams{
		Since:    time.Now().Add(-time.Hour),
		Before:   time.Now(),
		Statuses: &[]gen.V1TaskStatus{gen.V1TaskStatusCOMPLETED, gen.V1TaskStatusCANCELLED},
	}, false)

	assert.False(t, ok)
}

func TestActiveCountOptsFilters(t *testing.T) {
	since := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	workflowIds := []uuid.UUID{uuid.New()}
	idempotencyKeys := []string{"key-1"}
	and := gen.AND

	opts, ok := activeCountOpts(gen.V1WorkflowRunActiveCountParams{
		Since:                      since,
		Before:                     before,
		WorkflowIds:                &workflowIds,
		AdditionalMetadata:         &[]string{"env:prod", "region:us:east", "malformed"},
		AdditionalMetadataOperator: &and,
		IdempotencyKeys:            &idempotencyKeys,
	}, false)

	require.True(t, ok)
	assert.Equal(t, since, opts.CreatedAfter)
	require.NotNil(t, opts.FinishedBefore)
	assert.Equal(t, before.Add(-time.Microsecond), *opts.FinishedBefore)
	assert.Equal(t, workflowIds, opts.WorkflowIds)
	assert.Equal(t, map[string]interface{}{"env": "prod", "region": "us:east"}, opts.AdditionalMetadata)
	assert.Equal(t, v1.AdditionalMetadataOperatorAnd, opts.AdditionalMetadataOperator)
	assert.Equal(t, &idempotencyKeys, opts.IdempotencyKeys)
}
