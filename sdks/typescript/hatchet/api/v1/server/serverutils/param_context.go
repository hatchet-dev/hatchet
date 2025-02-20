package serverutils

import (
	"context"

	"github.com/labstack/echo/v4"
)

// ParamContext represents a subset of echo.Context to make testing easier
type ParamContext interface {
	// Get retrieves data from the context.
	Get(key string) interface{}

	// Set saves data in the context.
	Set(key string, val interface{})

	// Param returns path parameter by name.
	Param(name string) string

	// ParamNames returns path parameter names.
	ParamNames() []string

	// ParamValues returns path parameter values.
	ParamValues() []string
}

// RequestContext wraps echo.Context but Get and Set set values on the request context, rather than the
// echo context. This is necessary when we use the oapi-gen StrictServerInterface, since the implemented
// methods are passed a request context, not an echo context. Thus, our middleware needs to write to the
// request context as well.
type RequestContext struct {
	echo.Context
}

func NewRequestContext(ctx echo.Context) *RequestContext {
	return &RequestContext{ctx}
}

func (e *RequestContext) Get(key string) interface{} {
	return e.Request().Context().Value(key)
}

func (e *RequestContext) Set(key string, val interface{}) {
	r := e.Request()

	ctx := context.WithValue(e.Request().Context(), key, val)

	e.SetRequest(r.Clone(ctx))
}

type GoContext struct {
	context.Context

	echoContext echo.Context
}

func NewGoContext(ctx context.Context, echoContext echo.Context) *GoContext {
	return &GoContext{ctx, echoContext}
}

func (g *GoContext) Value(key any) interface{} {
	// first search echo context
	if keyStr, ok := key.(string); ok {
		val := g.echoContext.Get(keyStr)

		if val != nil {
			return val
		}
	}

	// then search go context
	return g.Context.Value(key)
}
