package scheduling

import (
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func TestComputeWeight(t *testing.T) {
	tests := []struct {
		name          string
		desiredLabels []*dbsqlc.GetDesiredLabelsRow
		workerLabels  []*dbsqlc.GetWorkerLabelsRow
		expected      int
	}{

		{
			name: "Simple equal match with valid string",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "environment",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					StrValue:   sqlchelpers.TextFromStr("production"),
					Weight:     10,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "environment",
					StrValue: sqlchelpers.TextFromStr("production"),
				},
			},
			expected: 10,
		},
		{
			name: "Simple equal match with valid int",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "cpu",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(4),
					Weight:     20,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "cpu",
					IntValue: sqlchelpers.ToInt(4),
				},
			},
			expected: 20,
		},
		{
			name: "No match returns zero weight",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "region",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					StrValue:   sqlchelpers.TextFromStr("us-west"),
					Weight:     15,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "region",
					StrValue: sqlchelpers.TextFromStr("us-east"),
				},
			},
			expected: 0,
		},
		{
			name: "No match for a required returns -1",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "memory",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					StrValue:   sqlchelpers.TextFromStr("100"),
					Weight:     15,
					Required:   false,
				},
				{
					Key:        "region",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					StrValue:   sqlchelpers.TextFromStr("us-west"),
					Weight:     15,
					Required:   true,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "region",
					StrValue: sqlchelpers.TextFromStr("us-east"),
				},
			},
			expected: -1,
		},
		{
			name: "Required label not found",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "gpu",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(1),
					Weight:     30,
					Required:   true,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "memory",
					IntValue: sqlchelpers.ToInt(16),
				},
			},
			expected: -1,
		},
		{
			name: "Greater than comparator match",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "cpu",
					Comparator: dbsqlc.WorkerLabelComparatorGREATERTHAN,
					IntValue:   sqlchelpers.ToInt(2),
					Weight:     25,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "cpu",
					IntValue: sqlchelpers.ToInt(4),
				},
			},
			expected: 25,
		},
		{
			name: "Less than comparator match",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "latency",
					Comparator: dbsqlc.WorkerLabelComparatorLESSTHAN,
					IntValue:   sqlchelpers.ToInt(100),
					Weight:     50,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "latency",
					IntValue: sqlchelpers.ToInt(80),
				},
			},
			expected: 50,
		},
		{
			name: "Greater than or equal comparator match",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "memory",
					Comparator: dbsqlc.WorkerLabelComparatorGREATERTHANOREQUAL,
					IntValue:   sqlchelpers.ToInt(16),
					Weight:     40,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "memory",
					IntValue: sqlchelpers.ToInt(16),
				},
			},
			expected: 40,
		},
		{
			name: "Less than or equal comparator match",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "latency",
					Comparator: dbsqlc.WorkerLabelComparatorLESSTHANOREQUAL,
					IntValue:   sqlchelpers.ToInt(50),
					Weight:     60,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "latency",
					IntValue: sqlchelpers.ToInt(30),
				},
			},
			expected: 60,
		},
		{
			name: "Label not found and not required",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "storage",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(500),
					Weight:     10,
					Required:   false,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "network",
					IntValue: sqlchelpers.ToInt(1000),
				},
			},
			expected: 0,
		},
		{
			name: "Multiple labels with mixed results",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "cpu",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(4),
					Weight:     20,
					Required:   false,
				},
				{
					Key:        "memory",
					Comparator: dbsqlc.WorkerLabelComparatorGREATERTHANOREQUAL,
					IntValue:   sqlchelpers.ToInt(8),
					Weight:     15,
					Required:   false,
				},
				{
					Key:        "gpu",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(1),
					Weight:     50,
					Required:   true,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "cpu",
					IntValue: sqlchelpers.ToInt(4),
				},
				{
					Key:      "memory",
					IntValue: sqlchelpers.ToInt(7),
				},
				{
					Key:      "gpu",
					IntValue: sqlchelpers.ToInt(1),
				},
			},
			expected: 70,
		},
		{
			name: "Required label missing and invalid match",
			desiredLabels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "region",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					StrValue:   sqlchelpers.TextFromStr("us-west"),
					Weight:     15,
					Required:   true,
				},
				{
					Key:        "cpu",
					Comparator: dbsqlc.WorkerLabelComparatorEQUAL,
					IntValue:   sqlchelpers.ToInt(4),
					Weight:     20,
					Required:   true,
				},
			},
			workerLabels: []*dbsqlc.GetWorkerLabelsRow{
				{
					Key:      "cpu",
					IntValue: sqlchelpers.ToInt(4),
				},
			},
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeWeight(tt.desiredLabels, tt.workerLabels)
			if result != tt.expected {
				t.Errorf("ComputeWeight() = %v, want %v", result, tt.expected)
			}
		})
	}
}
