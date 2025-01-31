package v2

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

var stableWorkerId1 = uuid.New().String()
var stableWorkerId2 = uuid.New().String()

func TestGetRankedSlots(t *testing.T) {
	tests := []struct {
		name           string
		qi             *dbsqlc.QueueItem
		labels         []*dbsqlc.GetDesiredLabelsRow
		slots          []*slot
		expectedWorker []string
	}{
		{
			name: "HARD sticky strategy with desired worker available",
			qi: &dbsqlc.QueueItem{
				Sticky:          dbsqlc.NullStickyStrategy{Valid: true, StickyStrategy: dbsqlc.StickyStrategyHARD},
				DesiredWorkerId: sqlchelpers.UUIDFromStr(stableWorkerId1),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId1)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (uuid.New().String())}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1},
		},
		{
			name: "HARD sticky strategy without desired worker",
			qi: &dbsqlc.QueueItem{
				Sticky:          dbsqlc.NullStickyStrategy{Valid: true, StickyStrategy: dbsqlc.StickyStrategyHARD},
				DesiredWorkerId: sqlchelpers.UUIDFromStr(uuid.New().String()),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (uuid.New().String())}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (uuid.New().String())}}, []string{}),
			},
			expectedWorker: []string{},
		},
		{
			name: "SOFT sticky strategy with desired worker available",
			qi: &dbsqlc.QueueItem{
				Sticky:          dbsqlc.NullStickyStrategy{Valid: true, StickyStrategy: dbsqlc.StickyStrategySOFT},
				DesiredWorkerId: sqlchelpers.UUIDFromStr(stableWorkerId1),
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId2)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId1)}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId1)}}, []string{}),
			},
			expectedWorker: []string{stableWorkerId1, stableWorkerId1, stableWorkerId2},
		},
		{
			name: "Affinity labels with different worker weights",
			qi:   &dbsqlc.QueueItem{},
			labels: []*dbsqlc.GetDesiredLabelsRow{
				{
					Key:        "key1",
					Weight:     1,
					Required:   false,
					Comparator: dbsqlc.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
				{
					Key:        "key2",
					Weight:     1,
					Required:   false,
					Comparator: dbsqlc.WorkerLabelComparatorGREATERTHAN,
					IntValue:   pgtype.Int4{Int32: 1, Valid: true},
				},
			},
			slots: []*slot{
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId1), Labels: []*dbsqlc.ListManyWorkerLabelsRow{{
					Key:      "key1",
					IntValue: pgtype.Int4{Int32: 2, Valid: true},
				}}}}, []string{}),
				newSlot(&worker{ListActiveWorkersResult: &repository.ListActiveWorkersResult{ID: (stableWorkerId2), Labels: []*dbsqlc.ListManyWorkerLabelsRow{{
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
