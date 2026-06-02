package worker

import (
	"fmt"
	"sync"
)

// BatchActionItem is what the batch fn receives per item.
type BatchActionItem struct {
	Ctx   HatchetContext
	Index int32
}

// BatchWrappedFn is the low-level batch function type registered with the worker.
// It receives all items in the batch and must return the same number of outputs in the same order.
type BatchWrappedFn func(items []BatchActionItem) ([]interface{}, error)

type batchItemResult struct {
	output interface{}
	err    error
}

type batchItemState struct {
	resultChan chan batchItemResult
	item       BatchActionItem
}

type batchGroupState struct {
	items        map[int32]*batchItemState
	actionId     string
	mu           sync.Mutex
	expectedSize int32
	started      bool
	flushed      bool
}

func (w *Worker) getOrCreateBatchState(batchId, actionId string) *batchGroupState {
	v, _ := w.batchStates.LoadOrStore(batchId, &batchGroupState{
		actionId: actionId,
		items:    make(map[int32]*batchItemState),
	})
	return v.(*batchGroupState)
}

func (w *Worker) handleBatchItem(ctx HatchetContext, actionId string) (interface{}, error) {
	batchId := ctx.BatchId()
	if batchId == nil {
		return nil, fmt.Errorf("batch item for action %s has no batch ID", actionId)
	}

	batchIndex := ctx.BatchIndex()
	if batchIndex == nil {
		return nil, fmt.Errorf("batch item for action %s has no batch index", actionId)
	}

	resultChan := make(chan batchItemResult, 1)

	itemState := &batchItemState{
		item: BatchActionItem{
			Index: *batchIndex,
			Ctx:   ctx,
		},
		resultChan: resultChan,
	}

	state := w.getOrCreateBatchState(*batchId, actionId)
	state.mu.Lock()
	state.items[*batchIndex] = itemState
	started := state.started
	expectedSize := state.expectedSize
	itemCount := int32(len(state.items)) //nolint:gosec
	state.mu.Unlock()

	if started && itemCount >= expectedSize {
		w.maybeFlushBatch(*batchId, state)
	}

	// Block until the batch fn resolves this item.
	result := <-resultChan
	return result.output, result.err
}

func (w *Worker) handleStartBatch(actionId, batchId string, expectedSize int32) {
	state := w.getOrCreateBatchState(batchId, actionId)
	state.mu.Lock()
	state.started = true
	state.expectedSize = expectedSize
	itemCount := int32(len(state.items)) //nolint:gosec
	state.mu.Unlock()

	if itemCount >= expectedSize {
		w.maybeFlushBatch(batchId, state)
	}
}

func (w *Worker) maybeFlushBatch(batchId string, state *batchGroupState) {
	state.mu.Lock()
	if state.flushed || !state.started || int32(len(state.items)) < state.expectedSize { //nolint:gosec
		state.mu.Unlock()
		return
	}
	state.flushed = true

	// Collect items ordered by index.
	ordered := make([]*batchItemState, state.expectedSize)
	for idx, s := range state.items {
		if idx < state.expectedSize {
			ordered[idx] = s
		}
	}
	actionId := state.actionId
	state.mu.Unlock()

	batchFnAny, ok := w.batchFns.Load(actionId)
	if !ok {
		err := fmt.Errorf("no batch fn registered for action %s", actionId)
		for _, s := range ordered {
			if s != nil {
				s.resultChan <- batchItemResult{err: err}
			}
		}
		w.batchStates.Delete(batchId)
		return
	}
	fn := batchFnAny.(BatchWrappedFn)

	items := make([]BatchActionItem, 0, len(ordered))
	for _, s := range ordered {
		if s != nil {
			items = append(items, s.item)
		}
	}

	go func() {
		defer w.batchStates.Delete(batchId)

		outputs, err := fn(items)

		for i, s := range ordered {
			if s == nil {
				continue
			}
			switch {
			case err != nil:
				s.resultChan <- batchItemResult{err: err}
			case i < len(outputs):
				s.resultChan <- batchItemResult{output: outputs[i]}
			default:
				s.resultChan <- batchItemResult{err: fmt.Errorf("batch fn returned fewer outputs than inputs (expected %d, got %d)", len(ordered), len(outputs))}
			}
		}
	}()
}
