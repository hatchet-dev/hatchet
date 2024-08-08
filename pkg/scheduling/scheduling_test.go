package scheduling

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type args struct {
	Slots            []*dbsqlc.ListSemaphoreSlotsToAssignRow
	UniqueActionsArr []string
	QueueItems       []QueueItemWithOrder
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

func assertResult(res SchedulePlan, filename string) (bool, error) {
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
	if res.ShouldContinue != expected.ShouldContinue {
		return false, fmt.Errorf("ShouldContinue does not match")
	}

	if len(res.StepRunIds) != len(expected.StepRunIds) {
		return false, fmt.Errorf("StepRunIds length does not match")
	}

	if len(res.StepRunTimeouts) != len(expected.StepRunTimeouts) {
		return false, fmt.Errorf("StepRunTimeouts length does not match")
	}

	if len(res.SlotIds) != len(expected.SlotIds) {
		return false, fmt.Errorf("SlotIds length does not match")
	}

	if len(res.WorkerIds) != len(expected.WorkerIds) {
		return false, fmt.Errorf("WorkerIds length does not match")
	}

	if len(res.UnassignedStepRunIds) != len(expected.UnassignedStepRunIds) {
		return false, fmt.Errorf("UnassignedStepRunIds length does not match")
	}

	if len(res.QueuedStepRuns) != len(expected.QueuedStepRuns) {
		return false, fmt.Errorf("QueuedStepRuns length does not match")
	}

	if len(res.TimedOutStepRuns) != len(expected.TimedOutStepRuns) {
		return false, fmt.Errorf("TimedOutStepRuns length does not match")
	}

	if len(res.QueuedItems) != len(expected.QueuedItems) {
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

			got, err := GeneratePlan(fixtureData.Slots, fixtureData.UniqueActionsArr, fixtureData.QueueItems)

			if !tt.wantErr(t, err, "GeneratePlan_Simple") {
				return
			}
			assert.Equalf(t, true, tt.want(got, tt.args.fixtureResult), "GeneratePlan_Simple")
		})
	}
}
