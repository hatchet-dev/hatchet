package dispatcher

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type recordingStream struct {
	grpc.ServerStream
	mu   sync.Mutex
	sent []*contracts.DurableTaskResponse
}

func (r *recordingStream) Send(resp *contracts.DurableTaskResponse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sent = append(r.sent, resp)
	return nil
}

func (r *recordingStream) Recv() (*contracts.DurableTaskRequest, error) {
	return nil, nil
}

func (r *recordingStream) sentNodeIds() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := make([]int64, 0, len(r.sent))
	for _, resp := range r.sent {
		nodes = append(nodes, resp.GetEntryCompleted().GetRef().GetNodeId())
	}
	return nodes
}

func newTestInvocation() (*durableTaskInvocation, *recordingStream) {
	stream := &recordingStream{}
	l := zerolog.Nop()
	return &durableTaskInvocation{server: stream, l: &l}, stream
}

func orderPtr(v int64) *int64 { return &v }

func entryCompletedResp(taskId uuid.UUID, invocationCount int32, branchId, nodeId int64) *contracts.DurableTaskResponse {
	return &contracts.DurableTaskResponse{
		Message: &contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref: &contracts.DurableEventLogEntryRef{
					DurableTaskExternalId: taskId.String(),
					InvocationCount:       invocationCount,
					BranchId:              branchId,
					NodeId:                nodeId,
				},
			},
		},
	}
}

// A->B / C->D: node1 completes at order 2, node2 at order 1. Delivering node1 first
// (out of satisfied order) must hold it until node2 (order 1) is released.
func TestDeliverOrdered_HoldsOutOfOrderCompletion(t *testing.T) {
	inv, stream := newTestInvocation()
	taskId := uuid.New()

	if err := inv.deliverOrdered(taskId, 1, orderPtr(2), entryCompletedResp(taskId, 1, 1, 1)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}
	if got := stream.sentNodeIds(); len(got) != 0 {
		t.Fatalf("order 2 should be held until order 1 arrives, got sends %v", got)
	}

	if err := inv.deliverOrdered(taskId, 1, orderPtr(1), entryCompletedResp(taskId, 1, 1, 2)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}

	// order 1 (node 2) releases and drains the held order 2 (node 1), in that order.
	if got := stream.sentNodeIds(); !equalInt64(got, []int64{2, 1}) {
		t.Fatalf("expected sends in satisfied order [node2, node1], got %v", got)
	}
}

func TestDeliverOrdered_ContiguousReleasesImmediately(t *testing.T) {
	inv, stream := newTestInvocation()
	taskId := uuid.New()

	for order := int64(1); order <= 3; order++ {
		if err := inv.deliverOrdered(taskId, 1, orderPtr(order), entryCompletedResp(taskId, 1, 1, order)); err != nil {
			t.Fatalf("deliverOrdered: %v", err)
		}
	}

	if got := stream.sentNodeIds(); !equalInt64(got, []int64{1, 2, 3}) {
		t.Fatalf("expected in-order sends, got %v", got)
	}
}

// Memos (and old-engine completions) carry no satisfied_order and must pass straight
// through even while a later ordered completion is held.
func TestDeliverOrdered_NilOrderPassesThrough(t *testing.T) {
	inv, stream := newTestInvocation()
	taskId := uuid.New()

	// hold an ordered completion (order 2, waiting for 1)
	if err := inv.deliverOrdered(taskId, 1, orderPtr(2), entryCompletedResp(taskId, 1, 1, 5)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}
	// a memo (nil order) for node 9 should still be delivered
	if err := inv.deliverOrdered(taskId, 1, nil, entryCompletedResp(taskId, 1, 1, 9)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}

	if got := stream.sentNodeIds(); !equalInt64(got, []int64{9}) {
		t.Fatalf("memo should pass through while order 2 is held, got %v", got)
	}
}

// A completion re-delivered after its order was already released (reconnect / poll)
// bypasses the buffer and is sent again; the worker dedupes by node id.
func TestDeliverOrdered_RedeliveryBypasses(t *testing.T) {
	inv, stream := newTestInvocation()
	taskId := uuid.New()

	if err := inv.deliverOrdered(taskId, 1, orderPtr(1), entryCompletedResp(taskId, 1, 1, 1)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}
	if err := inv.deliverOrdered(taskId, 1, orderPtr(1), entryCompletedResp(taskId, 1, 1, 1)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}

	if got := stream.sentNodeIds(); !equalInt64(got, []int64{1, 1}) {
		t.Fatalf("expected re-delivery to be sent again, got %v", got)
	}
}

func TestStaleReleaseHolds(t *testing.T) {
	inv, _ := newTestInvocation()
	taskId := uuid.New()

	// hold order 2 (waiting for 1)
	if err := inv.deliverOrdered(taskId, 1, orderPtr(2), entryCompletedResp(taskId, 1, 1, 1)); err != nil {
		t.Fatalf("deliverOrdered: %v", err)
	}

	if got := inv.staleReleaseHolds(time.Hour); len(got) != 0 {
		t.Fatalf("hold should not be stale within the timeout, got %v", got)
	}

	// simulate the hold having sat past the timeout
	key := orderedReleaseKey{taskExternalId: taskId, invocationCount: 1}
	inv.releasesMu.Lock()
	inv.releases[key].oldestHoldAt = time.Now().Add(-2 * time.Hour)
	inv.releasesMu.Unlock()

	stale := inv.staleReleaseHolds(time.Hour)
	if len(stale) != 1 || stale[0] != key {
		t.Fatalf("expected the stalled key to be reported, got %v", stale)
	}
}

func equalInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
