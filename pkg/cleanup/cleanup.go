package cleanup

import (
	"context"
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
	logger    *zerolog.Logger
}

func New(logger *zerolog.Logger) Cleanup {
	return Cleanup{
		Fns:       []CleanupFn{},
		TimeLimit: time.Second * 10,
		logger:    logger,
	}
}

func (c *Cleanup) Add(fn func() error, name string) {
	c.Fns = append(c.Fns, CleanupFn{
		Name: name,
		Fn:   fn,
	})
}

func (c *Cleanup) Run() error {
	lines := []string{}
	lines = append(lines, "waiting for all other services to gracefully exit...")
	go func() {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), c.TimeLimit)
		<-timeoutCtx.Done()
		for _, line := range lines {
			c.logger.Error().Msg(line)
		}
		cancel()
	}()
	for i, fn := range c.Fns {
		lines = append(lines, fmt.Sprintf("shutting down %s (%d/%d)", fn.Name, i+1, len(c.Fns)))
		before := time.Now()
		if err := fn.Fn(); err != nil {
			return fmt.Errorf("could not teardown %s: %w", fn.Name, err)
		}
		lines = append(lines, fmt.Sprintf("successfully shutdown %s in %s (%d/%d)\n", fn.Name, time.Since(before), i+1, len(c.Fns)))
	}
	lines = append(lines, "all services have successfully gracefully exited")
	return nil
}
