package scheduling

import (
	"sort"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

// WorkerWithWeight represents a worker with an associated weight
type WorkerWithWeight struct {
	WorkerId string
	Weight   int
}

// SortWorkerWeights sorts a slice of WorkerWithWeight in descending order of Weight
func SortWorkerWeights(weights []WorkerWithWeight) {
	sort.SliceStable(weights, func(i, j int) bool {
		// Sort by weight in descending order
		return weights[i].Weight > weights[j].Weight
	})
}

func ComputeWeight(s []*dbsqlc.GetDesiredLabelsRow, l []*dbsqlc.GetWorkerLabelsRow) int {
	totalWeight := 0

	for _, desiredLabel := range s {

		labelFound := false
		for _, workerLabel := range l {
			if desiredLabel.Key == workerLabel.Key {
				labelFound = true
				conditionMet := false
				switch desiredLabel.Comparator {
				case dbsqlc.WorkerLabelComparatorEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String == workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 == workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case dbsqlc.WorkerLabelComparatorNOTEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String != workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 != workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case dbsqlc.WorkerLabelComparatorGREATERTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 > desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case dbsqlc.WorkerLabelComparatorLESSTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 < desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case dbsqlc.WorkerLabelComparatorGREATERTHANOREQUAL:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 >= desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case dbsqlc.WorkerLabelComparatorLESSTHANOREQUAL:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 <= desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				}

				if !conditionMet && desiredLabel.Required {
					return -1
				}
				break // Move to the next desired label
			}
		}

		// If the label is required but not found, return -1 to indicate an invalid match
		if desiredLabel.Required && !labelFound {
			return -1
		}
	}

	return totalWeight
}
