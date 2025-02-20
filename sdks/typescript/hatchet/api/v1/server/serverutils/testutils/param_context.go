package testutils

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/serverutils"
)

type TestContext struct {
	ctx    context.Context
	params map[string]string
}

func GetTestContext(params map[string]string) serverutils.ParamContext {
	return &TestContext{context.Background(), params}
}

// Get retrieves data from the context.
func (t *TestContext) Get(key string) any {
	return t.ctx.Value(key)
}

// Set saves data in the context.
func (t *TestContext) Set(key string, val any) {
	t.ctx = context.WithValue(t.ctx, key, val)
}

// Param returns path parameter by name.
func (t *TestContext) Param(name string) string {
	return t.params[name]
}

// ParamNames returns path parameter names.
func (t *TestContext) ParamNames() []string {
	names := []string{}

	for name := range t.params {
		names = append(names, name)
	}

	return names
}

// ParamValues returns path parameter values.
func (t *TestContext) ParamValues() []string {
	values := []string{}

	for _, val := range t.params {
		values = append(values, val)
	}

	return values
}
