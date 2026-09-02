package repository

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// evalMaxRunsExpression shares the key-expression failure contract: any error fails the
// task at insert time, so the error cases below are user-visible task failures.
func TestEvalMaxRunsExpression(t *testing.T) {
	r := &sharedRepository{celParser: cel.NewCELParser()}

	strat := func(expr string) *sqlcv1.V1StepConcurrency {
		return &sqlcv1.V1StepConcurrency{MaxRunsExpression: pgtype.Text{String: expr, Valid: true}}
	}
	input := func(m map[string]interface{}) cel.Input {
		return cel.NewInput(cel.WithInput(m))
	}

	got, err := r.evalMaxRunsExpression(strat("input.tier == 'premium' ? 10 : 1"), input(map[string]interface{}{"tier": "premium"}))
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if !got.Valid || got.Int32 != 10 {
		t.Fatalf("got %+v, want valid 10", got)
	}

	got, err = r.evalMaxRunsExpression(strat("input.limit"), input(map[string]interface{}{"limit": 3}))
	if err != nil || got.Int32 != 3 {
		t.Fatalf("got (%+v, %v), want valid 3", got, err)
	}

	// a string result must fail the task, not silently coerce
	if _, err = r.evalMaxRunsExpression(strat("input.name"), input(map[string]interface{}{"name": "abc"})); err == nil {
		t.Fatalf("expected error for string result")
	}

	// zero and negative results must fail the task, matching the static min=1 validation
	if _, err = r.evalMaxRunsExpression(strat("input.limit"), input(map[string]interface{}{"limit": 0})); err == nil {
		t.Fatalf("expected error for zero result")
	}
	if _, err = r.evalMaxRunsExpression(strat("input.limit"), input(map[string]interface{}{"limit": -2})); err == nil {
		t.Fatalf("expected error for negative result")
	}

	// evaluation errors (missing key) propagate as task failures
	if _, err = r.evalMaxRunsExpression(strat("input.missing"), input(map[string]interface{}{})); err == nil {
		t.Fatalf("expected error for missing input key")
	}
}
