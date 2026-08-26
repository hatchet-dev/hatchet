//go:build !e2e && !load && !rampup && !integration

package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/sdks/go/internal/task"
)

func TestWorkflowDump_DefaultFilters(t *testing.T) {
	cases := []struct {
		name    string
		filters []types.DefaultFilter
		want    []*contracts.DefaultFilter
	}{
		{
			name:    "no filters",
			filters: nil,
			want:    []*contracts.DefaultFilter{},
		},
		{
			name: "single filter with no payload",
			filters: []types.DefaultFilter{
				{Expression: "input.type == 'order'", Scope: "workflow"},
			},
			want: []*contracts.DefaultFilter{
				{Expression: "input.type == 'order'", Scope: "workflow", Payload: mustMarshal(t, nil)},
			},
		},
		{
			name: "single filter with payload",
			filters: []types.DefaultFilter{
				{
					Expression: "input.region == payload.region",
					Scope:      "tenant",
					Payload:    map[string]any{"region": "us-east-1"},
				},
			},
			want: []*contracts.DefaultFilter{
				{
					Expression: "input.region == payload.region",
					Scope:      "tenant",
					Payload:    mustMarshal(t, map[string]any{"region": "us-east-1"}),
				},
			},
		},
		{
			name: "multiple filters",
			filters: []types.DefaultFilter{
				{Expression: "input.type == 'a'", Scope: "workflow"},
				{Expression: "input.type == 'b'", Scope: "workflow", Payload: map[string]any{"key": "val"}},
			},
			want: []*contracts.DefaultFilter{
				{Expression: "input.type == 'a'", Scope: "workflow", Payload: mustMarshal(t, nil)},
				{Expression: "input.type == 'b'", Scope: "workflow", Payload: mustMarshal(t, map[string]any{"key": "val"})},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wf := &workflowDeclarationImpl[any, any]{
				name:             "test-workflow",
				tasks:            []*task.TaskDeclaration[any]{},
				durableTasks:     []*task.DurableTaskDeclaration[any]{},
				taskFuncs:        make(map[string]any),
				durableTaskFuncs: make(map[string]any),
				outputSetters:    make(map[string]func(*any, any)),
				DefaultFilters:   tc.filters,
			}

			req, _, _, _ := wf.Dump()

			require.Len(t, req.DefaultFilters, len(tc.want))
			for i, wantFilter := range tc.want {
				got := req.DefaultFilters[i]
				assert.Equal(t, wantFilter.Expression, got.Expression)
				assert.Equal(t, wantFilter.Scope, got.Scope)
				assert.JSONEq(t, string(wantFilter.Payload), string(got.Payload))
			}
		})
	}
}

func TestWorkflowDump_ConcurrencyExceptStrategies(t *testing.T) {
	cases := []struct {
		name     string
		strategy types.WorkflowConcurrencyLimitStrategy
		want     contracts.ConcurrencyLimitStrategy
	}{
		{
			name:     "cancel except newest",
			strategy: types.CancelExceptNewest,
			want:     contracts.ConcurrencyLimitStrategy_CANCEL_EXCEPT_NEWEST,
		},
		{
			name:     "cancel except oldest",
			strategy: types.CancelExceptOldest,
			want:     contracts.ConcurrencyLimitStrategy_CANCEL_EXCEPT_OLDEST,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			maxRuns := int32(1)

			wf := &workflowDeclarationImpl[any, any]{
				name:             "test-workflow",
				tasks:            []*task.TaskDeclaration[any]{},
				durableTasks:     []*task.DurableTaskDeclaration[any]{},
				taskFuncs:        make(map[string]any),
				durableTaskFuncs: make(map[string]any),
				outputSetters:    make(map[string]func(*any, any)),
				Concurrency: []types.Concurrency{
					{
						Expression:    "input.group",
						MaxRuns:       &maxRuns,
						LimitStrategy: &tc.strategy,
					},
				},
			}

			req, _, _, _ := wf.Dump()

			require.Len(t, req.ConcurrencyArr, 1)
			require.NotNil(t, req.ConcurrencyArr[0].LimitStrategy)
			assert.Equal(t, tc.want, *req.ConcurrencyArr[0].LimitStrategy)
		})
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
