package loader

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

type debugger struct {
	callerCounts map[string]int
	callerMu     sync.Mutex

	lastPrint *time.Time

	l *zerolog.Logger
}

func (d *debugger) beforeAcquire(ctx context.Context, conn *pgx.Conn) bool {
	_, file, line, ok := runtime.Caller(4)

	caller := "unknown"

	if ok {
		if strings.Contains(file, "tx.go") {
			_, file, line, _ = runtime.Caller(5)
		}

		caller = fmt.Sprintf("%s:%d", file, line)
	}

	d.callerMu.Lock()
	d.callerCounts[caller]++
	d.callerMu.Unlock()

	if d.lastPrint == nil || time.Since(*d.lastPrint) > 15*time.Second {
		d.printCallerCounts()
	}

	return true
}

type callerCount struct {
	caller string
	count  int
}

func (d *debugger) printCallerCounts() {
	d.callerMu.Lock()
	defer d.callerMu.Unlock()

	var counts []callerCount

	for caller, count := range d.callerCounts {
		counts = append(counts, callerCount{caller, count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	sl := d.l.Debug()

	for i, c := range counts {
		// print only the top 20 callers
		if i >= 20 {
			break
		}

		sl.Int(
			c.caller,
			c.count,
		)
	}

	sl.Msg("top 20 callers")

	d.callerCounts = make(map[string]int)
	now := time.Now()
	d.lastPrint = &now
}
