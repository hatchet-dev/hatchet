package cel

import (
	"context"
	"fmt"
	"hash/fnv"

	celgo "github.com/google/cel-go/cel"
	lru "github.com/hashicorp/golang-lru/v2"
)

type BoolExprEvaluator struct {
	env   *celgo.Env
	cache *lru.Cache[uint64, celgo.Program]
}

func NewBoolExprEvaluator() (*BoolExprEvaluator, error) {
	env, err := celgo.NewEnv(
		celgo.Variable("input", celgo.MapType(celgo.StringType, celgo.DynType)),
		celgo.Variable("output", celgo.MapType(celgo.StringType, celgo.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	cache, err := lru.New[uint64, celgo.Program](50000)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program cache: %w", err)
	}

	return &BoolExprEvaluator{env: env, cache: cache}, nil
}

func (e *BoolExprEvaluator) Compile(expr string) (celgo.Program, error) {
	if expr == "" {
		expr = "true"
	}

	hasher := fnv.New64a()
	hasher.Write([]byte(expr))
	exprHash := hasher.Sum64()

	if program, ok := e.cache.Get(exprHash); ok {
		return program, nil
	}

	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression %q: %w", expr, issues.Err())
	}

	program, err := e.env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program for %q: %w", expr, err)
	}

	e.cache.Add(exprHash, program)
	return program, nil
}

func (e *BoolExprEvaluator) EvalBoolExpr(ctx context.Context, expr string, vars map[string]interface{}) (bool, error) {
	program, err := e.Compile(expr)
	if err != nil {
		return false, err
	}

	out, _, err := program.ContextEval(ctx, vars)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression %q: %w", expr, err)
	}

	b, ok := out.Value().(bool)
	return ok && b, nil
}
