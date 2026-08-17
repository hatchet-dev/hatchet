package repository

import (
	"testing"
)

func TestChecksumV1_BackwardsCompatibility(t *testing.T) {
	// Compute a baseline checksum with no IsDurable or SlotRequests fields set
	// (simulating a pre-feature workflow registration).
	baselineOpts := &CreateWorkflowVersionOpts{
		Name: "test-workflow",
		Tasks: []CreateStepOpts{
			{
				ReadableId: "step1",
				Action:     "default:step1",
			},
		},
	}

	baselineChecksum, _, err := checksumV1(baselineOpts)
	if err != nil {
		t.Fatalf("unexpected error computing baseline checksum: %v", err)
	}

	t.Run("IsDurable false does not change hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId: "step1",
					Action:     "default:step1",
					IsDurable:  false,
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs != baselineChecksum {
			t.Errorf("IsDurable=false changed the hash\n  baseline: %s\n  got:      %s", baselineChecksum, cs)
		}
	})

	t.Run("SlotRequests default:1 does not change hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId:   "step1",
					Action:       "default:step1",
					SlotRequests: map[string]int32{SlotTypeDefault: 1},
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs != baselineChecksum {
			t.Errorf("SlotRequests={default:1} changed the hash\n  baseline: %s\n  got:      %s", baselineChecksum, cs)
		}
	})

	t.Run("IsDurable false and SlotRequests default:1 together do not change hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId:   "step1",
					Action:       "default:step1",
					IsDurable:    false,
					SlotRequests: map[string]int32{SlotTypeDefault: 1},
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs != baselineChecksum {
			t.Errorf("IsDurable=false + SlotRequests={default:1} changed the hash\n  baseline: %s\n  got:      %s", baselineChecksum, cs)
		}
	})

	t.Run("IsDurable true changes hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId: "step1",
					Action:     "default:step1",
					IsDurable:  true,
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs == baselineChecksum {
			t.Error("IsDurable=true should change the hash, but it did not")
		}
	})

	t.Run("custom SlotRequests changes hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId:   "step1",
					Action:       "default:step1",
					SlotRequests: map[string]int32{"gpu": 2},
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs == baselineChecksum {
			t.Error("SlotRequests={gpu:2} should change the hash, but it did not")
		}
	})

	t.Run("SlotRequests default:2 changes hash", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name: "test-workflow",
			Tasks: []CreateStepOpts{
				{
					ReadableId:   "step1",
					Action:       "default:step1",
					SlotRequests: map[string]int32{SlotTypeDefault: 2},
				},
			},
		}

		cs, _, err := checksumV1(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cs == baselineChecksum {
			t.Error("SlotRequests={default:2} should change the hash, but it did not")
		}
	})
}

func TestMergeWorkflowConcurrencyOntoSingleTask(t *testing.T) {
	maxRuns := int32(3)
	strategy := "GROUP_ROUND_ROBIN"
	wfConc := CreateConcurrencyOpts{
		MaxRuns:       &maxRuns,
		LimitStrategy: &strategy,
		Expression:    "input.user_id",
	}
	taskMaxRuns := int32(1)
	taskStrategy := "CANCEL_IN_PROGRESS"
	taskConc := CreateConcurrencyOpts{
		MaxRuns:       &taskMaxRuns,
		LimitStrategy: &taskStrategy,
		Expression:    "input.task_id",
	}

	t.Run("single task prepends workflow concurrency and clears workflow-level", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name:        "single-task",
			Concurrency: []CreateConcurrencyOpts{wfConc},
			Tasks: []CreateStepOpts{
				{
					ReadableId:  "step1",
					Action:      "default:step1",
					Concurrency: []CreateConcurrencyOpts{taskConc},
				},
			},
		}

		mergeWorkflowConcurrencyOntoSingleTask(opts)

		if opts.Concurrency != nil {
			t.Fatalf("expected workflow concurrency to be cleared, got %d entries", len(opts.Concurrency))
		}
		if len(opts.Tasks[0].Concurrency) != 2 {
			t.Fatalf("expected 2 task concurrency entries, got %d", len(opts.Tasks[0].Concurrency))
		}
		if opts.Tasks[0].Concurrency[0].Expression != wfConc.Expression {
			t.Errorf("workflow concurrency should be first, got %s", opts.Tasks[0].Concurrency[0].Expression)
		}
		if opts.Tasks[0].Concurrency[1].Expression != taskConc.Expression {
			t.Errorf("existing task concurrency should follow, got %s", opts.Tasks[0].Concurrency[1].Expression)
		}
	})

	t.Run("multi-task DAG keeps workflow concurrency", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name:        "multi-task",
			Concurrency: []CreateConcurrencyOpts{wfConc},
			Tasks: []CreateStepOpts{
				{ReadableId: "step1", Action: "default:step1"},
				{ReadableId: "step2", Action: "default:step2", Parents: []string{"step1"}},
			},
		}

		mergeWorkflowConcurrencyOntoSingleTask(opts)

		if len(opts.Concurrency) != 1 {
			t.Fatalf("expected workflow concurrency to remain, got %d entries", len(opts.Concurrency))
		}
		if len(opts.Tasks[0].Concurrency) != 0 {
			t.Errorf("did not expect task concurrency to be mutated, got %d entries", len(opts.Tasks[0].Concurrency))
		}
	})

	t.Run("on-failure keeps workflow concurrency", func(t *testing.T) {
		opts := &CreateWorkflowVersionOpts{
			Name:        "with-on-failure",
			Concurrency: []CreateConcurrencyOpts{wfConc},
			Tasks: []CreateStepOpts{
				{ReadableId: "step1", Action: "default:step1"},
			},
			OnFailure: &CreateStepOpts{ReadableId: "on-failure", Action: "default:on-failure"},
		}

		mergeWorkflowConcurrencyOntoSingleTask(opts)

		if len(opts.Concurrency) != 1 {
			t.Fatalf("expected workflow concurrency to remain when on-failure is set, got %d entries", len(opts.Concurrency))
		}
	})

	t.Run("checksum matches putting the same concurrency on the task", func(t *testing.T) {
		fromWorkflow := &CreateWorkflowVersionOpts{
			Name:        "checksum-single-task",
			Concurrency: []CreateConcurrencyOpts{wfConc},
			Tasks: []CreateStepOpts{
				{ReadableId: "step1", Action: "default:step1"},
			},
		}
		fromTask := &CreateWorkflowVersionOpts{
			Name: "checksum-single-task",
			Tasks: []CreateStepOpts{
				{
					ReadableId:  "step1",
					Action:      "default:step1",
					Concurrency: []CreateConcurrencyOpts{wfConc},
				},
			},
		}

		mergeWorkflowConcurrencyOntoSingleTask(fromWorkflow)

		csWorkflow, _, err := checksumV1(fromWorkflow)
		if err != nil {
			t.Fatalf("unexpected error hashing workflow-level opts: %v", err)
		}
		csTask, _, err := checksumV1(fromTask)
		if err != nil {
			t.Fatalf("unexpected error hashing task-level opts: %v", err)
		}

		if csWorkflow != csTask {
			t.Errorf("merged workflow concurrency should checksum the same as task concurrency\n  workflow: %s\n  task:     %s", csWorkflow, csTask)
		}
	})
}
