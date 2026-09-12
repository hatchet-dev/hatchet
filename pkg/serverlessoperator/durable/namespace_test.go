//go:build !e2e && !load && !rampup && !integration

package durable

import (
	"context"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// TestRelayConfinesNestedNamesToTheNamespace is the security F13 regression: workflow
// names in trigger_runs and user event keys in wait_for conditions are prefixed with the
// endpoint's namespace before either host sees them, the way the operator prefixes what it
// registers, so an endpoint can only reach resources of its own namespace. Names that
// already carry the prefix, sleep conditions, event scopes and memo keys are untouched.
func TestRelayConfinesNestedNamesToTheNamespace(t *testing.T) {
	setT(t)

	ep := newFakeEndpoint(t, func(t *testing.T, conn *websocket.Conn, _ *v1.ServerlessFirstFrame) {
		writeFrame(t, conn, `{"id":1,"request":{"triggerRuns":{"triggerOpts":[{"name":"privileged","input":"{}"},{"name":"ns_already","input":"{}"}]}}}`)
		require.NotNil(t, readResponse(t, conn).GetTriggerRunsAck())

		writeFrame(t, conn, `{"id":2,"request":{"waitFor":{"waitForConditions":{
			"sleepConditions":[{"base":{"readableDataKey":"sleep:1s","orGroupId":"g1"},"sleepFor":"1s"}],
			"userEventConditions":[
				{"base":{"readableDataKey":"payment","orGroupId":"g2"},"userEventKey":"payment","eventScope":"order-1"},
				{"base":{"readableDataKey":"ns_paid","orGroupId":"g3"},"userEventKey":"ns_paid"}
			]}}}}`)
		require.NotNil(t, readResponse(t, conn).GetWaitForAck())

		writeFrame(t, conn, `{"id":3,"request":{"memo":{"key":"cGF5bWVudA=="}}}`)
		require.NotNil(t, readResponse(t, conn).GetMemoAck())

		writeFrame(t, conn, `{"done":{"output":"{}"}}`)
	})

	ch := newFakeChannel()
	out := run(context.Background(), testParams(ep, ch))

	trigger := ch.next(t).GetTriggerRuns()
	require.NotNil(t, trigger)
	require.Len(t, trigger.TriggerOpts, 2)
	assert.Equal(t, "ns_privileged", trigger.TriggerOpts[0].Name)
	assert.Equal(t, "ns_already", trigger.TriggerOpts[1].Name)
	assert.Equal(t, testTaskId, trigger.DurableTaskExternalId)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_TriggerRunsAck{
		TriggerRunsAck: &v1.DurableTaskEventTriggerRunsAckResponse{},
	}})

	waitFor := ch.next(t).GetWaitFor()
	require.NotNil(t, waitFor)

	conds := waitFor.GetWaitForConditions()
	require.Len(t, conds.SleepConditions, 1)
	assert.Equal(t, "sleep:1s", conds.SleepConditions[0].Base.ReadableDataKey)
	require.Len(t, conds.UserEventConditions, 2)
	assert.Equal(t, "ns_payment", conds.UserEventConditions[0].UserEventKey)
	assert.Equal(t, "payment", conds.UserEventConditions[0].Base.ReadableDataKey, "readable keys are labels, not resources")
	assert.Equal(t, "order-1", conds.UserEventConditions[0].GetEventScope())
	assert.Equal(t, "ns_paid", conds.UserEventConditions[1].UserEventKey)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_WaitForAck{
		WaitForAck: &v1.DurableTaskEventWaitForAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 2}},
	}})

	memo := ch.next(t).GetMemo()
	require.NotNil(t, memo)
	assert.Equal(t, []byte("payment"), memo.Key)

	ch.reply(&v1.DurableTaskResponse{Message: &v1.DurableTaskResponse_MemoAck{
		MemoAck: &v1.DurableTaskEventMemoAckResponse{Ref: &v1.DurableEventLogEntryRef{NodeId: 3}},
	}})

	o := await(t, out)
	assert.Equal(t, KindCompleted, o.Kind)
}
