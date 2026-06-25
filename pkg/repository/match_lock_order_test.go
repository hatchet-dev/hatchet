//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

func TestSortedUniqueLogFileRefs(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	ts1 := sqlchelpers.TimestamptzFromTime(t1)
	ts2 := sqlchelpers.TimestamptzFromTime(t2)

	taskIds := []int64{5, 3, 5, 3, 5}
	insertedAts := []pgtype.Timestamptz{ts1, ts2, ts1, ts2, ts2}

	gotIds, gotAts := sortedUniqueLogFileRefs(taskIds, insertedAts)

	// deduplicated and sorted by (task id, inserted at) so concurrent
	// transactions lock log files in a consistent order
	assert.Equal(t, []int64{3, 5, 5}, gotIds)
	assert.Equal(t, []time.Time{t2, t1, t2}, []time.Time{gotAts[0].Time, gotAts[1].Time, gotAts[2].Time})
}

func TestSortedUniqueLogFileRefs_Empty(t *testing.T) {
	gotIds, gotAts := sortedUniqueLogFileRefs(nil, nil)
	assert.Empty(t, gotIds)
	assert.Empty(t, gotAts)
}
