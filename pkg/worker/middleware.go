package worker

import (
	"fmt"
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

func panicMiddleware(ctx HatchetContext, next func(HatchetContext) error) error {
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
