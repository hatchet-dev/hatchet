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

// return func(c echo.Context) (returnErr error) {
// 	if config.Skipper(c) {
// 		return next(c)
// 	}

// 	defer func() {
// 		if r := recover(); r != nil {
// 			if r == http.ErrAbortHandler {
// 				panic(r)
// 			}
// 			err, ok := r.(error)
// 			if !ok {
// 				err = fmt.Errorf("%v", r)
// 			}
// 			var stack []byte
// 			var length int

// 			if !config.DisablePrintStack {
// 				stack = make([]byte, config.StackSize)
// 				length = runtime.Stack(stack, !config.DisableStackAll)
// 				stack = stack[:length]
// 			}

// 			if config.LogErrorFunc != nil {
// 				err = config.LogErrorFunc(c, err, stack)
// 			} else if !config.DisablePrintStack {
// 				msg := fmt.Sprintf("[PANIC RECOVER] %v %s\n", err, stack[:length])
// 				switch config.LogLevel {
// 				case log.DEBUG:
// 					c.Logger().Debug(msg)
// 				case log.INFO:
// 					c.Logger().Info(msg)
// 				case log.WARN:
// 					c.Logger().Warn(msg)
// 				case log.ERROR:
// 					c.Logger().Error(msg)
// 				case log.OFF:
// 					// None.
// 				default:
// 					c.Logger().Print(msg)
// 				}
// 			}

// 			if err != nil && !config.DisableErrorHandler {
// 				c.Error(err)
// 			} else {
// 				returnErr = err
// 			}
// 		}
// 	}()
// 	return next(c)
// }
