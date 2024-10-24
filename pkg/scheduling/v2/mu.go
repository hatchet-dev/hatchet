package v2

import (
	"fmt"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/sasha-s/go-deadlock"
)

type debugMu struct {
	l *zerolog.Logger
}

func (d debugMu) print(start time.Time) {
	if since := time.Since(start); since > 50*time.Millisecond {
		// get the line number of the caller
		_, file, line, ok := runtime.Caller(2)

		caller := "unknown"
		if ok {
			caller = fmt.Sprintf("%s:%d", file, line)
		}

		d.l.Warn().Dur("duration", since).Msgf("long lock %s", caller)
	}
}

type mutex struct {
	*deadlock.Mutex
	debugMu
}

func newMu(l *zerolog.Logger) mutex {
	return mutex{
		Mutex:   &deadlock.Mutex{},
		debugMu: debugMu{l: l},
	}
}

func (m mutex) Lock() {
	defer m.debugMu.print(time.Now())
	m.Mutex.Lock()
}

type rwMutex struct {
	*deadlock.RWMutex
	debugMu
}

func newRWMu(l *zerolog.Logger) rwMutex {
	return rwMutex{
		RWMutex: &deadlock.RWMutex{},
		debugMu: debugMu{l: l},
	}
}

func (m rwMutex) Lock() {
	defer m.debugMu.print(time.Now())
	m.RWMutex.Lock()
}

func (m rwMutex) RLock() {
	defer m.debugMu.print(time.Now())
	m.RWMutex.RLock()
}
