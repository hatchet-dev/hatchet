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
	logger    *zerolog.Logger
	Fns       []CleanupFn
	TimeLimit time.Duration
}

func New(logger *zerolog.Logger) Cleanup {
	return Cleanup{
		Fns:       []CleanupFn{},
		TimeLimit: time.Second * 9,
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
	// 1st and last line + 2 lines for each fn. Makes sure we don't block on the chan send
	lines := make(chan string)
	defer close(lines)
	ctx, cancel := context.WithTimeout(context.Background(), c.TimeLimit)
	defer cancel()
	go func() {
		// log at the debug level by default
		logger := c.logger.Debug
		<-ctx.Done()

		if ctx.Err() == context.DeadlineExceeded {
			// if the ctx is cancelled due to a timeout, then promote the logs
			// to an error
			logger = c.logger.Error
		}
		for line := range lines {
			logger().Msg(line)
		}
	}()
	lines <- "waiting for all other services to gracefully exit..."
	for i, fn := range c.Fns {
		lines <- fmt.Sprintf("shutting down %s (%d/%d)", fn.Name, i+1, len(c.Fns))
		before := time.Now()
		if err := fn.Fn(); err != nil {
			return fmt.Errorf("could not teardown %s: %w", fn.Name, err)
		}
		lines <- fmt.Sprintf("successfully shutdown %s in %s (%d/%d)\n", fn.Name, time.Since(before), i+1, len(c.Fns))
	}
	lines <- "all services have successfully gracefully exited"
	return nil
}
