package v2

import (
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type worker struct {
	*v1.ListActiveWorkersResult
}

// computeWeight computes the weight of a worker based on the desired labels. If the worker does not
// meet the required labels, the weight is -1.
func (w *worker) computeWeight(s []*sqlcv1.GetDesiredLabelsRow) int {
	totalWeight := 0

	for _, desiredLabel := range s {
		labelFound := false

		for _, workerLabel := range w.Labels {
			if desiredLabel.Key == workerLabel.Key {
				labelFound = true
				conditionMet := false
				switch desiredLabel.Comparator {
				case sqlcv1.WorkerLabelComparatorEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String == workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 == workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case sqlcv1.WorkerLabelComparatorNOTEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String != workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 != workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case sqlcv1.WorkerLabelComparatorGREATERTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 > desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case sqlcv1.WorkerLabelComparatorLESSTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 < desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case sqlcv1.WorkerLabelComparatorGREATERTHANOREQUAL:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 >= desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
						conditionMet = true
					}
				case sqlcv1.WorkerLabelComparatorLESSTHANOREQUAL:
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
