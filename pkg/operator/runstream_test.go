//go:build !e2e && !load && !rampup && !integration

package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// Every kind resolves from its procedure and back, with its direction and message types.
func TestRunStreamKinds(t *testing.T) {
	for procedure, want := range map[string]struct {
		kind     RunStreamKind
		bidi     bool
		request  any
		response any
	}{
		"/Dispatcher/SubscribeToWorkflowRuns":    {RunStreamWorkflowRuns, true, &contracts.SubscribeToWorkflowRunsRequest{}, &contracts.WorkflowRunEvent{}},
		"/Dispatcher/SubscribeToWorkflowEvents":  {RunStreamWorkflowEvents, false, &contracts.SubscribeToWorkflowEventsRequest{}, &contracts.WorkflowEvent{}},
		"/v1.V1Dispatcher/ListenForDurableEvent": {RunStreamDurableEvents, true, &v1.ListenForDurableEventRequest{}, &v1.DurableEvent{}},
		"/v1.V1Streams/Subscribe":                {RunStreamDurableStreams, false, &v1.SubscribeStreamRequest{}, &v1.StreamMessage{}},
	} {
		t.Run(procedure, func(t *testing.T) {
			kind, ok := RunStreamKindOf(procedure)
			require.True(t, ok)
			assert.Equal(t, want.kind, kind)
			assert.Equal(t, procedure, kind.Procedure())
			assert.Equal(t, procedure, kind.String())
			assert.Equal(t, want.bidi, kind.Bidi())
			assert.IsType(t, want.request, kind.NewRequest())
			assert.IsType(t, want.response, kind.NewResponse())
		})
	}

	_, ok := RunStreamKindOf("/v1.V1Streams/Publish")
	assert.False(t, ok, "a unary procedure is no stream kind")

	unknown := RunStreamKind(99)
	assert.Nil(t, unknown.NewRequest())
	assert.Nil(t, unknown.NewResponse())
	assert.Equal(t, "RunStreamKind(99)", unknown.Procedure())
}
