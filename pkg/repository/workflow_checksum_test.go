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
