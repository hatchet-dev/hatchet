package hatchet

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v0Client "github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type durableTaskRuntime interface {
	DurableTaskListener() *v0Client.DurableTaskListener
	DurableEvictionSupported() bool
}

type durableEvictionHookProvider interface {
	DurableEvictionHook() worker.DurableEvictionHook
}

type childIndexProvider interface {
	NextChildIndex() int
}

func runDurableChildWorkflow(
	ctx context.Context,
	workflowName string,
	input any,
	runOpts *runOpts,
) (*WorkflowRunRef, bool, error) {
	hCtx, ok := ctx.(Context)
	if !ok {
		return nil, false, nil
	}

	runtime, ok := ctx.(durableTaskRuntime)
	if !ok || runtime.DurableTaskListener() == nil || !runtime.DurableEvictionSupported() {
		return nil, false, nil
	}

	trigger, err := buildDurableChildTrigger(hCtx, workflowName, input, runOpts)
	if err != nil {
		return nil, true, err
	}

	invocationCount := hCtx.DurableTaskInvocationCount()
	if invocationCount == 0 {
		invocationCount = 1
	}

	listener := runtime.DurableTaskListener()
	entries, err := listener.SendTriggerRunsRequest(
		hCtx.GetContext(),
		hCtx.StepRunId(),
		invocationCount,
		[]*v1.TriggerWorkflowRequest{trigger},
	)
	if err != nil {
		return nil, true, fmt.Errorf("failed to spawn durable child workflow: %w", err)
	}
	if len(entries) == 0 {
		return nil, true, fmt.Errorf("failed to spawn durable child workflow: no run entries returned")
	}

	entry := entries[0]
	resultFn := func() (*WorkflowResult, error) {
		payload, err := waitForDurableChildResult(ctx, listener, hCtx, invocationCount, workflowName, entry)
		if err != nil {
			return nil, err
		}

		return &WorkflowResult{
			RunId:  entry.WorkflowRunID,
			result: payload,
		}, nil
	}

	return &WorkflowRunRef{
		RunId:    entry.WorkflowRunID,
		resultFn: resultFn,
	}, true, nil
}

func runManyDurableChildWorkflows(
	ctx context.Context,
	otelCtx context.Context,
	workflowName string,
	inputs []RunManyOpt,
) ([]WorkflowRunRef, bool, error) {
	hCtx, ok := ctx.(Context)
	if !ok {
		return nil, false, nil
	}

	runtime, ok := ctx.(durableTaskRuntime)
	if !ok || runtime.DurableTaskListener() == nil || !runtime.DurableEvictionSupported() {
		return nil, false, nil
	}

	invocationCount := hCtx.DurableTaskInvocationCount()
	if invocationCount == 0 {
		invocationCount = 1
	}

	triggers := make([]*v1.TriggerWorkflowRequest, len(inputs))
	for i, input := range inputs {
		runOpts := &runOpts{}
		for _, opt := range input.Opts {
			opt(runOpts)
		}
		runOpts.AdditionalMetadata = injectTraceparentToMap(otelCtx, runOpts.AdditionalMetadata)

		trigger, err := buildDurableChildTrigger(hCtx, workflowName, input.Input, runOpts)
		if err != nil {
			return nil, true, err
		}
		triggers[i] = trigger
	}

	listener := runtime.DurableTaskListener()
	entries, err := listener.SendTriggerRunsRequest(
		hCtx.GetContext(),
		hCtx.StepRunId(),
		invocationCount,
		triggers,
	)
	if err != nil {
		return nil, true, fmt.Errorf("failed to spawn durable child workflows: %w", err)
	}
	if len(entries) != len(triggers) {
		return nil, true, fmt.Errorf("failed to spawn durable child workflows: expected %d run entries, got %d", len(triggers), len(entries))
	}

	refs := make([]WorkflowRunRef, len(entries))
	for i, entry := range entries {
		entry := entry
		refs[i] = WorkflowRunRef{
			RunId: entry.WorkflowRunID,
			resultFn: func() (*WorkflowResult, error) {
				payload, err := waitForDurableChildResult(ctx, listener, hCtx, invocationCount, workflowName, entry)
				if err != nil {
					return nil, err
				}

				return &WorkflowResult{
					RunId:  entry.WorkflowRunID,
					result: payload,
				}, nil
			},
		}
	}

	return refs, true, nil
}

func buildDurableChildTrigger(
	hCtx Context,
	workflowName string,
	input any,
	runOpts *runOpts,
) (*v1.TriggerWorkflowRequest, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("could not marshal input: %w", err)
	}

	childIndexValue := hCtx.CurChildIndex()
	if next, ok := hCtx.(childIndexProvider); ok {
		childIndexValue = next.NextChildIndex()
	} else {
		hCtx.IncChildIndex()
	}
	childIndex := int32(childIndexValue) // nolint:gosec

	var desiredWorkerID *string
	if runOpts.Sticky != nil && *runOpts.Sticky {
		if !hCtx.Worker().HasWorkflow(workflowName) {
			return nil, fmt.Errorf("cannot run with sticky: workflow %s is not registered on this worker", workflowName)
		}
		workerID := hCtx.Worker().ID()
		desiredWorkerID = &workerID
	}

	additionalMetadata, err := marshalAdditionalMetadata(runOpts.AdditionalMetadata)
	if err != nil {
		return nil, err
	}
	var priority *int32
	if runOpts.Priority != nil {
		priority = &[]int32{int32(*runOpts.Priority)}[0]
	}

	return &v1.TriggerWorkflowRequest{
		Name:                    workflowName,
		Input:                   string(inputBytes),
		ParentId:                strPtr(hCtx.WorkflowRunId()),
		ParentTaskRunExternalId: strPtr(hCtx.StepRunId()),
		ChildIndex:              &childIndex,
		ChildKey:                runOpts.Key,
		AdditionalMetadata:      additionalMetadata,
		DesiredWorkerId:         desiredWorkerID,
		Priority:                priority,
		DesiredWorkerLabels:     desiredWorkerLabelsToProto(runOpts.DesiredWorkerLabels),
	}, nil
}

func waitForDurableChildResult(
	ctx context.Context,
	listener *v0Client.DurableTaskListener,
	hCtx Context,
	invocationCount int32,
	workflowName string,
	entry v0Client.TriggerRunAckEntry,
) (any, error) {
	if hookProvider, ok := ctx.(durableEvictionHookProvider); ok {
		if hook := hookProvider.DurableEvictionHook(); hook != nil {
			hook.MarkWaiting(hCtx.StepRunId(), "spawn_child", workflowName)
			defer hook.MarkActive(hCtx.StepRunId())
		}
	}

	payloadBytes, err := listener.WaitForCallback(
		hCtx.GetContext(),
		hCtx.StepRunId(),
		invocationCount,
		entry.BranchID,
		entry.NodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for durable child workflow: %w", err)
	}

	if len(payloadBytes) == 0 {
		return map[string]any{}, nil
	}

	var payload any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode durable child workflow result: %w", err)
	}

	return payload, nil
}

func marshalAdditionalMetadata(metadata *map[string]string) (*string, error) {
	if metadata == nil {
		return nil, nil
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	metadataString := string(metadataBytes)
	return &metadataString, nil
}

func desiredWorkerLabelsToProto(labels map[string]*DesiredWorkerLabel) map[string]*v1.DesiredWorkerLabels {
	if labels == nil {
		return nil
	}

	result := make(map[string]*v1.DesiredWorkerLabels, len(labels))
	for key, label := range labels {
		protoLabel := &v1.DesiredWorkerLabels{
			Required: &label.Required,
			Weight:   &label.Weight,
		}

		if label.Comparator != nil {
			comparator := v1.WorkerLabelComparator(*label.Comparator)
			protoLabel.Comparator = &comparator
		}

		switch v := label.Value.(type) {
		case string:
			protoLabel.StrValue = &v
		case int:
			intVal := int32(v) // nolint:gosec
			protoLabel.IntValue = &intVal
		case int32:
			protoLabel.IntValue = &v
		}

		result[key] = protoLabel
	}

	return result
}

func strPtr(s string) *string {
	return &s
}
