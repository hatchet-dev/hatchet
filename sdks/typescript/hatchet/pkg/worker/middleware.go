package worker

import (
	"fmt"
	"runtime/debug"
	"sync"
)

type MiddlewareFunc func(ctx HatchetContext, next func(HatchetContext) error) error

type middlewares struct {
	mu          sync.Mutex
	middlewares []MiddlewareFunc
}

func newMiddlewares() *middlewares {
	return &middlewares{
		middlewares: []MiddlewareFunc{},
	}
}

func (m *middlewares) add(mws ...MiddlewareFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.middlewares = append(m.middlewares, mws...)
}

func (m *middlewares) runAll(ctx HatchetContext, next func(HatchetContext) error) error {
	return run(ctx, m.middlewares, next)
}

func run(ctx HatchetContext, fs []MiddlewareFunc, next func(HatchetContext) error) error {
	// base case: no more middleware to run
	if len(fs) == 0 {
		return next(ctx)
	}

	return fs[0](ctx, func(ctx HatchetContext) error {
		return run(ctx, fs[1:], next)
	})
}

func (w *Worker) panicMiddleware(ctx HatchetContext, next func(HatchetContext) error) error {
	var err error

	func() {
		defer func() {
			if r := recover(); r != nil {
				var ok bool
				err, ok = r.(error)

				if !ok {
					err = fmt.Errorf("%v", r)
				}

				innerErr := w.sendFailureEvent(ctx, fmt.Errorf("recovered from panic: %w. Stack trace:\n%s", err, string(debug.Stack())))

				if innerErr != nil {
					w.l.Error().Err(innerErr).Msg("could not send failure event")
				}

				return
			}
		}()

		err = next(ctx)
	}()

	return err
}
