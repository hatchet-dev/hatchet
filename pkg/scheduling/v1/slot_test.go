//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var stableWorkerId1 = uuid.New()
var stableWorkerId2 = uuid.New()

func ptrUUID(s string) *uuid.UUID {
	u := uuid.MustParse(s)
	return &u
}

func TestGetRankedSlots(t *testing.T) {
	tests := []struct {
		name           string
		qi             *sqlcv1.V1QueueItem
		labels         []*sqlcv1.GetDesiredLabelsRow
		slots          []*slot
		expectedWorker []string
	}{
		{
			name: "HARD sticky strategy with desired worker available",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategyHARD,
				DesiredWorkerID: &stableWorkerId1,
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: stableWorkerId1}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: uuid.New()}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1.String()},
		},
		{
			name: "HARD sticky strategy without desired worker",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategyHARD,
				DesiredWorkerID: ptrUUID(uuid.New().String()),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: uuid.New()}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: uuid.New()}}, []string{}),
			},
			expectedWorker: []string{},
		},
		{
			name: "SOFT sticky strategy with desired worker available",
			qi: &sqlcv1.V1QueueItem{
				Sticky:          sqlcv1.V1StickyStrategySOFT,
				DesiredWorkerID: &stableWorkerId1,
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId2)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId1)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId1)}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1.String(), stableWorkerId1.String(), stableWorkerId2.String()},
		},
		{
			name: "Affinity labels with different worker weights",
			qi:   &sqlcv1.V1QueueItem{},
			labels: []*sqlcv1.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   false,
					Comparator: sqlcv1.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
				{
					Key:        "key2",
					Weight:     1,
					Required:   false,
					Comparator: sqlcv1.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId1), Labels: []*sqlcv1.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 2, Valid: true},
				}}}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId2), Labels: []*sqlcv1.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 4, Valid: true},
				}, {
					Key:      "key2",
					IntValue: pgtype.Int4{Int32: 4, Valid: true},
				}}}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId2.String(), stableWorkerId1.String()},
		},
		{
			name: "Affinity labels with strict requirements",
			qi:   &sqlcv1.V1QueueItem{},
			labels: []*sqlcv1.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   true,
					Comparator: sqlcv1.WorkerLabelComparatorEQUAL,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId1), Labels: []*sqlcv1.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 1, Valid: true},
				}}}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1.String()},
		},
		{
			name: "Affinity labels with strict requirements and unsatisfiable conditions",
			qi:   &sqlcv1.V1QueueItem{},
			labels: []*sqlcv1.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   true,
					Comparator: sqlcv1.WorkerLabelComparatorEQUAL,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v1.ListActiveWorkersResult{ID: (stableWorkerId2), Labels: []*sqlcv1.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 2, Valid: true},
				}}}}, []string{}),
			},
			expectedWorker: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSlots := getRankedSlots(tt.qi, tt.labels, tt.slots)
			actualWorkerIds := make([]string, len(actualSlots))
			for i, s := range actualSlots {
				actualWorkerIds[i] = s.getWorkerId().String()
			}

			assert.Equal(t, tt.expectedWorker, actualWorkerIds)
		})
	}
}
