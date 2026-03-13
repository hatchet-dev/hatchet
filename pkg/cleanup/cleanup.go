package cleanup

import (
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type CleanupFn struct {
	Fn   func() error
	Name string
}

type Cleanup struct {
	Fns       []CleanupFn
	TimeLimit time.Duration
}

func New() Cleanup {
	return Cleanup{
		Fns:       []CleanupFn{},
		TimeLimit: time.Second * 30,
	}
}

func (c *Cleanup) Add(fn func() error, name string) {
	c.Fns = append(c.Fns, CleanupFn{
		Name: name,
		Fn:   fn,
	})
}

func (c *Cleanup) Run(l *zerolog.Logger) error {
	lines := []string{}
	start := time.Now()
	lines = append(lines, "waiting for all other services to gracefully exit...")
	for i, fn := range c.Fns {
		lines = append(lines, fmt.Sprintf("shutting down %s (%d/%d)", fn.Name, i+1, len(c.Fns)))
		before := time.Now()
		if err := fn.Fn(); err != nil {
			return fmt.Errorf("could not teardown %s: %w", fn.Name, err)
		}
		lines = append(lines, fmt.Sprintf("successfully shutdown %s in %s (%d/%d)\n", fn.Name, time.Since(before), i+1, len(c.Fns)))
	}
	lines = append(lines, "all services have successfully gracefully exited")
	if time.Since(start) > c.TimeLimit {
		for _, line := range lines {
			l.Warn().Msg(line)
		}
		return fmt.Errorf("cleanup exceeded time limit of %d seconds", c.TimeLimit)
	}
	return nil
}
