package worker

import (
	"context"
	"fmt"
	"sync"
)

type MiddlewareFunc func(ctx context.Context, next func(context.Context) error) error

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

func (m *middlewares) runAll(ctx context.Context, next func(context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return run(ctx, m.middlewares, next)
}

func run(ctx context.Context, fs []MiddlewareFunc, next func(context.Context) error) error {
	// base case: no more middleware to run
	if len(fs) == 0 {
		return next(ctx)
	}

	return fs[0](ctx, func(ctx context.Context) error {
		return run(ctx, fs[1:], next)
	})
}

func panicMiddleware(ctx context.Context, next func(context.Context) error) error {
	var err error
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer func() {
			defer wg.Done()
			if r := recover(); r != nil {
				var ok bool
				err, ok = r.(error)

				if !ok {
					err = fmt.Errorf("%v", r)
				}

				return
			}
		}()

		err = next(ctx)
	}()

	wg.Wait()

	return err
}
