package debugger

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Debugger struct {
	callerCounts map[string]int
	activeConns  map[*pgx.Conn]string
	lastPrint    *time.Time
	l            *zerolog.Logger
	pool         *pgxpool.Pool
	callerMu     sync.Mutex
	poolMu       sync.Mutex
}

func NewDebugger(l *zerolog.Logger) *Debugger {
	return &Debugger{
		callerCounts: make(map[string]int),
		activeConns:  make(map[*pgx.Conn]string),
		l:            l,
	}
}

func (d *Debugger) Setup(pool *pgxpool.Pool) {
	d.poolMu.Lock()
	defer d.poolMu.Unlock()

	d.pool = pool
	d.activeConns = make(map[*pgx.Conn]string)
}

func (d *Debugger) getPool() *pgxpool.Pool {
	d.poolMu.Lock()
	defer d.poolMu.Unlock()

	return d.pool
}

func (d *Debugger) BeforeAcquire(ctx context.Context, conn *pgx.Conn) bool {
	// if we don't have a pool set yet, skip
	if d.getPool() == nil {
		return true
	}

	_, file, line, ok := runtime.Caller(5)

	caller := "unknown"

	if ok {
		if strings.Contains(file, "tx.go") {
			_, file, line, _ = runtime.Caller(6)
		}

		caller = fmt.Sprintf("%s:%d", file, line)
	}

	d.callerMu.Lock()
	d.callerCounts[caller]++
	d.activeConns[conn] = caller
	d.callerMu.Unlock()

	if d.lastPrint == nil || time.Since(*d.lastPrint) > 120*time.Second {
		d.printCallerCounts()
	}

	if d.pool.Stat().AcquiredConns() == d.pool.Config().MaxConns {
		d.printActiveCallers()
	}

	return true
}

func (d *Debugger) AfterRelease(conn *pgx.Conn) bool {
	if d.getPool() == nil {
		return true
	}

	d.callerMu.Lock()
	delete(d.activeConns, conn)
	d.callerMu.Unlock()

	return true
}

type callerCount struct {
	caller string
	count  int
}

func (d *Debugger) printCallerCounts() {
	d.callerMu.Lock()
	defer d.callerMu.Unlock()

	var counts []callerCount

	for caller, count := range d.callerCounts {
		counts = append(counts, callerCount{caller, count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	sl := d.l.Warn()

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

func (d *Debugger) printActiveCallers() {
	// print the active callers, grouped by caller
	d.callerMu.Lock()
	defer d.callerMu.Unlock()

	callerMap := make(map[string]int)

	for _, caller := range d.activeConns {
		callerMap[caller]++
	}

	sl := d.l.Warn()

	for caller, count := range callerMap {
		sl.Int(
			caller,
			count,
		)
	}

	sl.Msg("hit max database connections, showing active callers")
}
