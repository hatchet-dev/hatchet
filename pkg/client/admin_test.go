package client

import (
	"strings"
	"testing"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func requestsWithInputSize(count, inputBytes int) []*v1contracts.TriggerWorkflowRequest {
	requests := make([]*v1contracts.TriggerWorkflowRequest, count)
	for i := range requests {
		requests[i] = &v1contracts.TriggerWorkflowRequest{
			Name:  "test",
			Input: strings.Repeat("a", inputBytes),
		}
	}
	return requests
}

func TestNextChunkEnd(t *testing.T) {
	small := requestsWithInputSize(3, 10)
	if end := nextChunkEnd(small, 0); end != 3 {
		t.Errorf("small requests: expected end 3, got %d", end)
	}

	capped := requestsWithInputSize(bulkTriggerMaxChunkSize+5, 10)
	if end := nextChunkEnd(capped, 0); end != bulkTriggerMaxChunkSize {
		t.Errorf("count cap: expected end %d, got %d", bulkTriggerMaxChunkSize, end)
	}
	if end := nextChunkEnd(capped, bulkTriggerMaxChunkSize); end != len(capped) {
		t.Errorf("count cap remainder: expected end %d, got %d", len(capped), end)
	}

	large := requestsWithInputSize(3, bulkTriggerMaxChunkBytes/2)
	if end := nextChunkEnd(large, 0); end != 1 {
		t.Errorf("byte cap: expected end 1, got %d", end)
	}

	oversized := requestsWithInputSize(2, bulkTriggerMaxChunkBytes+1)
	if end := nextChunkEnd(oversized, 0); end != 1 {
		t.Errorf("oversized request: expected end 1, got %d", end)
	}
	if end := nextChunkEnd(oversized, 1); end != 2 {
		t.Errorf("oversized request remainder: expected end 2, got %d", end)
	}
}
