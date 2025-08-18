package scheduling

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRateLimit_AddStepRunId(t *testing.T) {
	testCases := []struct {
		name          string
		initialUnits  int32
		maxUnits      int32
		addUnits      int32
		expectSuccess bool
	}{
		{
			name:          "Add within limits",
			initialUnits:  0,
			maxUnits:      100,
			addUnits:      30,
			expectSuccess: true,
		},
		{
			name:          "Add exceeding limits",
			initialUnits:  0,
			maxUnits:      50,
			addUnits:      80,
			expectSuccess: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := "test_key"
			rlRow := &dbsqlc.ListRateLimitsForTenantWithMutateRow{
				Value:        tc.maxUnits,
				NextRefillAt: pgtype.Timestamp{Time: time.Now(), Valid: true}, // Not used in this test
			}
			rl := NewRateLimit(key, rlRow)
			rl.currUnitsConsumed = tc.initialUnits

			stepRunId := "step1"
			success := rl.AddStepRunId(stepRunId, tc.addUnits)
			if success != tc.expectSuccess {
				t.Errorf("(*RateLimit).AddStepRunId() = %v, want %v, tc: %+v", success, tc.expectSuccess, tc)
			}
		})
	}
}
func TestRateLimit_Rollback(t *testing.T) {
	type step struct {
		id    string
		units int32
	}

	testCases := []struct {
		name           string
		initialUnits   int32
		steps          []step
		rollbackStepID string
		expectedUnits  int32
	}{
		{
			name:         "Rollback single step",
			initialUnits: 0,
			steps: []step{
				{"step1", 30},
			},
			rollbackStepID: "step1",
			expectedUnits:  0,
		},
		{
			name:         "Rollback non-existing step",
			initialUnits: 0,
			steps: []step{
				{"step1", 30},
			},
			rollbackStepID: "step2",
			expectedUnits:  30,
		},
		{
			name:         "Rollback one step among multiple",
			initialUnits: 0,
			steps: []step{
				{"step1", 30},
				{"step2", 40},
			},
			rollbackStepID: "step1",
			expectedUnits:  40,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := "test_key"
			rlRow := &dbsqlc.ListRateLimitsForTenantWithMutateRow{
				Value:        100,
				NextRefillAt: pgtype.Timestamp{Time: time.Now(), Valid: true},
			}
			rl := NewRateLimit(key, rlRow)
			rl.currUnitsConsumed = tc.initialUnits

			// Add steps
			for _, step := range tc.steps {
				_ = rl.AddStepRunId(step.id, step.units)
			}

			// Execute rollback for 지정된 step
			rl.Rollback(tc.rollbackStepID)

			if rl.currUnitsConsumed != tc.expectedUnits {
				t.Errorf("After Rollback(%s), currUnitsConsumed = %d, want %d", tc.rollbackStepID, rl.currUnitsConsumed, tc.expectedUnits)
			}
		})
	}
}
