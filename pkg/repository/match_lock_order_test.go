//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

func TestSortedUniqueLogFileRefs(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	ts1 := sqlchelpers.TimestamptzFromTime(t1)
	ts2 := sqlchelpers.TimestamptzFromTime(t2)

	refs := []IdInsertedAt{
		{ID: 5, InsertedAt: ts1},
		{ID: 3, InsertedAt: ts2},
		{ID: 5, InsertedAt: ts1},
		{ID: 3, InsertedAt: ts2},
		{ID: 5, InsertedAt: ts2},
	}

	got := sortedUniqueLogFileRefs(refs)

	// deduplicated and sorted by (task id, inserted at) so concurrent
	// transactions lock log files in a consistent order
	gotIds := []int64{got[0].ID, got[1].ID, got[2].ID}
	gotAts := []time.Time{got[0].InsertedAt.Time, got[1].InsertedAt.Time, got[2].InsertedAt.Time}
	assert.Equal(t, []int64{3, 5, 5}, gotIds)
	assert.Equal(t, []time.Time{t2, t1, t2}, gotAts)
}

func TestSortedUniqueLogFileRefs_Empty(t *testing.T) {
	assert.Empty(t, sortedUniqueLogFileRefs(nil))
}
