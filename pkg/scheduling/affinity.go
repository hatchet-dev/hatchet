package scheduling

import "github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"

func ComputeWeight(s []*dbsqlc.GetDesiredLabelsRow, l []*dbsqlc.GetWorkerLabelsRow) int {
	totalWeight := 0

	for _, desiredLabel := range s {
		labelFound := false
		for _, workerLabel := range l {
			if desiredLabel.Key == workerLabel.Key {
				labelFound = true
				switch desiredLabel.Comparator {
				case dbsqlc.WorkerLabelComparatorEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String == workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 == workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
					}
				case dbsqlc.WorkerLabelComparatorNOTEQUAL:
					if (desiredLabel.StrValue.Valid && workerLabel.StrValue.Valid && desiredLabel.StrValue.String != workerLabel.StrValue.String) ||
						(desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && desiredLabel.IntValue.Int32 != workerLabel.IntValue.Int32) {
						totalWeight += int(desiredLabel.Weight)
					}
				case dbsqlc.WorkerLabelComparatorGREATERTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 > desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
					}
				case dbsqlc.WorkerLabelComparatorLESSTHAN:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 < desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
					}
				case dbsqlc.WorkerLabelComparatorGREATERTHANOREQUAL:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 >= desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
					}
				case dbsqlc.WorkerLabelComparatorLESSTHANOREQUAL:
					if desiredLabel.IntValue.Valid && workerLabel.IntValue.Valid && workerLabel.IntValue.Int32 <= desiredLabel.IntValue.Int32 {
						totalWeight += int(desiredLabel.Weight)
					}
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
