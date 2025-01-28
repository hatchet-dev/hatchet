package v2

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"

	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

var stableWorkerId1 = uuid.New().String()
var stableWorkerId2 = uuid.New().String()

func TestGetRankedSlots(t *testing.T) {
	tests := []struct {
		name           string
		qi             *sqlcv2.V2QueueItem
		labels         []*sqlcv2.GetDesiredLabelsRow
		slots          []*slot
		expectedWorker []string
	}{
		{
			name: "HARD sticky strategy with desired worker available",
			qi: &sqlcv2.V2QueueItem{
				Sticky:          sqlcv2.V2StickyStrategyHARD,
				DesiredWorkerID: sqlchelpers.UUIDFromStr(stableWorkerId1),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId1)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(uuid.New().String())}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1},
		},
		{
			name: "HARD sticky strategy without desired worker",
			qi: &sqlcv2.V2QueueItem{
				Sticky:          sqlcv2.V2StickyStrategyHARD,
				DesiredWorkerID: sqlchelpers.UUIDFromStr(uuid.New().String()),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(uuid.New().String())}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(uuid.New().String())}}, []string{}),
			},
			expectedWorker: []string{},
		},
		{
			name: "SOFT sticky strategy with desired worker available",
			qi: &sqlcv2.V2QueueItem{
				Sticky:          sqlcv2.V2StickyStrategySOFT,
				DesiredWorkerID: sqlchelpers.UUIDFromStr(stableWorkerId1),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId2)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId1)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId1)}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1, stableWorkerId1, stableWorkerId2},
		},
		{
			name: "Affinity labels with different worker weights",
			qi:   &sqlcv2.V2QueueItem{},
			labels: []*sqlcv2.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   false,
					Comparator: sqlcv2.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
				{
					Key:        "key2",
					Weight:     1,
					Required:   false,
					Comparator: sqlcv2.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId1), Labels: []*sqlcv2.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 2, Valid: true},
				}}}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &v2.ListActiveWorkersResult{ID: sqlchelpers.UUIDFromStr(stableWorkerId2), Labels: []*sqlcv2.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 4, Valid: true},
				}, {
					Key:      "key2",
					IntValue: pgtype.Int4{Int32: 4, Valid: true},
				}}}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId2, stableWorkerId1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualSlots := getRankedSlots(tt.qi, tt.labels, tt.slots)
			actualWorkerIds := make([]string, len(actualSlots))
			for i, s := range actualSlots {
				actualWorkerIds[i] = s.getWorkerId()
			}

			assert.Equal(t, tt.expectedWorker, actualWorkerIds)
		})
	}
}
