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
	Fns []CleanupFn
}

func New() Cleanup {
	return Cleanup{
		Fns: []CleanupFn{},
	}
}

func (c *Cleanup) Add(fn func() error, name string) {
	c.Fns = append(c.Fns, CleanupFn{
		Name: name,
		Fn:   fn,
	})
}

func (c *Cleanup) Run(l *zerolog.Logger) error {
	l.Warn().Msgf("waiting for all other services to gracefully exit...")
	for i, fn := range c.Fns {
		l.Warn().Msgf("shutting down %s (%d/%d)\n", fn.Name, i+1, len(c.Fns))
		before := time.Now()
		if err := fn.Fn(); err != nil {
			return fmt.Errorf("could not teardown %s: %w", fn.Name, err)
		}
		l.Warn().Msgf("successfully shutdown %s in %s (%d/%d)\n", fn.Name, time.Since(before), i+1, len(c.Fns))
	}
	l.Warn().Msgf("all services have successfully gracefully exited\n")
	return nil
}
