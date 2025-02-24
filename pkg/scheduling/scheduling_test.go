package scheduling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type args struct {
	Slots             []*Slot
	UniqueActionsArr  []string
	QueueItems        []*QueueItemWithOrder
	WorkerLabels      map[string][]*dbsqlc.GetWorkerLabelsRow
	StepDesiredLabels map[string][]*dbsqlc.GetDesiredLabelsRow
}

func loadFixture(filename string, noTimeout bool) (*args, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	args := &args{}

	// Unmarshal fixture into args
	err = json.Unmarshal(data, args)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal fixture: %v", err)
	}

	if noTimeout {

		for i := range args.QueueItems {
			var timestamp pgtype.Timestamp
			timestamp.Time = time.Now().Add(time.Hour * 24 * 7)
			args.QueueItems[i].ScheduleTimeoutAt = timestamp
		}
	}

	return args, nil
}

func assertResult(actual SchedulePlan, filename string) (bool, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}

	var expected SchedulePlan
	err = json.Unmarshal(data, &expected)
	if err != nil {
		return false, fmt.Errorf("Failed to unmarshal expected result: %v", err)
	}

	// Compare the results
	if actual.ShouldContinue != expected.ShouldContinue {
		return false, fmt.Errorf("ShouldContinue does not match")
	}

	if len(actual.StepRunIds) != len(expected.StepRunIds) {
		return false, fmt.Errorf("StepRunIds length does not match")
	}

	if len(actual.StepRunTimeouts) != len(expected.StepRunTimeouts) {
		return false, fmt.Errorf("StepRunTimeouts length does not match")
	}

	if len(actual.SlotIds) != len(expected.SlotIds) {
		return false, fmt.Errorf("SlotIds length does not match")
	}

	for i := range actual.QueuedStepRuns {
		if actual.QueuedStepRuns[i].WorkerId != expected.QueuedStepRuns[i].WorkerId {
			return false, fmt.Errorf("Expected worker mismatch")
		}
	}

	if len(actual.WorkerIds) != len(expected.WorkerIds) {
		return false, fmt.Errorf("WorkerIds length does not match")
	}

	if len(actual.UnassignedStepRunIds) != len(expected.UnassignedStepRunIds) {
		return false, fmt.Errorf("UnassignedStepRunIds length does not match")
	}

	if len(actual.QueuedStepRuns) != len(expected.QueuedStepRuns) {
		return false, fmt.Errorf("QueuedStepRuns length does not match")
	}

	if len(actual.TimedOutStepRuns) != len(expected.TimedOutStepRuns) {
		return false, fmt.Errorf("TimedOutStepRuns length does not match")
	}

	if len(actual.QueuedItems) != len(expected.QueuedItems) {
		return false, fmt.Errorf("QueuedItems length does not match")
	}

	return true, nil
}

func DumpResults(slots SchedulePlan, name string) error {
	data, err := json.MarshalIndent(slots, "", "  ")
	if err != nil {
		fmt.Println("Failed to marshal results")
		return err
	}

	err = os.WriteFile(name, data, 0600)
	if err != nil {
		fmt.Println("Failed to write results to file")
		return err
	}

	return nil
}

func TestGeneratePlan(t *testing.T) {
	type args struct {
		fixtureArgs   string
		fixtureResult string
		noTimeout     bool
	}
	tests := []struct {
		name    string
		args    args
		want    func(SchedulePlan, string) bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "GeneratePlan_Simple",
			args: args{
				fixtureArgs:   "./fixtures/simple_plan.json",
				fixtureResult: "./fixtures/simple_plan_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "./simple_plan_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_Affinity_Soft",
			args: args{
				fixtureArgs:   "./fixtures/affinity_soft.json",
				fixtureResult: "./fixtures/affinity_soft_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "affinity_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_Affinity_Hard",
			args: args{
				fixtureArgs:   "./fixtures/affinity_hard.json",
				fixtureResult: "./fixtures/affinity_hard_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "affinity_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_Sticky_Soft",
			args: args{
				fixtureArgs:   "./fixtures/sticky_soft.json",
				fixtureResult: "./fixtures/sticky_soft_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "sticky_soft_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_Sticky_Hard",
			args: args{
				fixtureArgs:   "./fixtures/sticky_hard.json",
				fixtureResult: "./fixtures/sticky_hard_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "sticky_hard_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_Sticky_Hard_No_Desired",
			args: args{
				fixtureArgs:   "./fixtures/sticky_hard_no_desired.json",
				fixtureResult: "./fixtures/sticky_hard_no_desired_output.json",
				noTimeout:     true,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "sticky_hard_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
		{
			name: "GeneratePlan_TimedOut",
			args: args{
				fixtureArgs:   "./fixtures/simple_plan.json",
				fixtureResult: "./fixtures/simple_plan_timeout_output.json",
				noTimeout:     false,
			},
			want: func(s SchedulePlan, fixtureResult string) bool {
				// DumpResults(s, "./simple_plan_timeout_output.json")

				assert, err := assertResult(s, fixtureResult)
				if err != nil {
					fmt.Println(err)
				}

				return assert
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load fixture
			fixtureData, err := loadFixture(tt.args.fixtureArgs, tt.args.noTimeout)

			if err != nil {
				t.Fatalf("Failed to load fixture: %v", err)
			}

			got, err := GeneratePlan(
				context.Background(),
				fixtureData.Slots,
				fixtureData.UniqueActionsArr,
				fixtureData.QueueItems,
				nil,
				nil,
				fixtureData.WorkerLabels,
				fixtureData.StepDesiredLabels,
			)

			if !tt.wantErr(t, err, "GeneratePlan_Simple") {
				return
			}
			assert.Equalf(t, true, tt.want(got, tt.args.fixtureResult), "GeneratePlan_Simple")
		})
	}
}

// Benchmark simple plan
func BenchmarkGeneratePlan(b *testing.B) {
	fixtureData, err := loadFixture("./fixtures/simple_plan.json", true)
	if err != nil {
		b.Fatalf("Failed to load fixture: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_, _ = GeneratePlan(
			context.Background(),
			fixtureData.Slots,
			fixtureData.UniqueActionsArr,
			fixtureData.QueueItems,
			nil,
			nil,
			fixtureData.WorkerLabels,
			fixtureData.StepDesiredLabels,
		)
	}
}
