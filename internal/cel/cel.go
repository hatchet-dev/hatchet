package cel

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"

	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type CELParser struct {
	workflowStrEnv *cel.Env
	stepRunEnv     *cel.Env
	eventEnv       *cel.Env
}

var checksumDecl = decls.NewFunction("checksum",
	decls.NewOverload("checksum_string",
		[]*expr.Type{decls.String},
		decls.String,
	),
)

var checksum = cel.Function("checksum",
	cel.MemberOverload(
		"checksum_string_impl",
		[]*cel.Type{cel.StringType},
		cel.StringType,
		cel.FunctionBinding(func(args ...ref.Val) ref.Val {
			if len(args) != 1 {
				return types.NewErr("checksum requires 1 argument")
			}
			str, ok := args[0].(types.String)
			if !ok {
				return types.NewErr("argument must be a string")
			}
			hash := sha256.Sum256([]byte(str))
			return types.String(fmt.Sprintf("%x", hash))
		})),
)

func NewCELParser() *CELParser {
	workflowStrEnv, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("additional_metadata", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("workflow_run_id", decls.String),
			checksumDecl,
		),
		checksum,
	)

	stepRunEnv, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("additional_metadata", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("parents", decls.NewMapType(decls.String, decls.NewMapType(decls.String, decls.Dyn))),
			decls.NewVar("workflow_run_id", decls.String),
			checksumDecl,
		),
		checksum,
	)

	eventEnv, _ := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("input", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("additional_metadata", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("event_id", decls.String),
			checksumDecl,
		),
	)

	return &CELParser{
		workflowStrEnv: workflowStrEnv,
		stepRunEnv:     stepRunEnv,
		eventEnv:       eventEnv,
	}
}

type Input map[string]interface{}

type InputOpts func(Input)

func WithInput(input map[string]interface{}) InputOpts {
	return func(w Input) {
		w["input"] = input
	}
}

func WithParents(parents any) InputOpts {
	return func(w Input) {
		w["parents"] = parents
	}
}

func WithAdditionalMetadata(metadata map[string]interface{}) InputOpts {
	return func(w Input) {
		w["additional_metadata"] = metadata
	}
}

func WithWorkflowRunID(workflowRunID string) InputOpts {
	return func(w Input) {
		w["workflow_run_id"] = workflowRunID
	}
}

func NewInput(opts ...InputOpts) Input {
	res := make(map[string]interface{})

	for _, opt := range opts {
		opt(res)
	}

	return res
}

func (p *CELParser) ParseWorkflowString(workflowExp string) (cel.Program, error) {
	ast, issues := p.workflowStrEnv.Compile(workflowExp)

	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	return p.workflowStrEnv.Program(ast)
}

func (p *CELParser) ParseAndEvalWorkflowString(workflowExp string, in Input) (string, error) {
	prg, err := p.ParseWorkflowString(workflowExp)
	if err != nil {
		return "", err
	}

	var inMap map[string]interface{} = in

	out, _, err := prg.Eval(inMap)
	if err != nil {
		return "", err
	}

	// Switch on the type of the output.
	switch out.Type() {
	case types.StringType:
		return out.Value().(string), nil
	default:
		return "", fmt.Errorf("output must evaluate to a string: got %s", out.Type().TypeName())
	}
}

type StepRunOutType string

const (
	StepRunOutTypeString StepRunOutType = "string"
	StepRunOutTypeInt    StepRunOutType = "int"
)

type StepRunOut struct {
	String *string
	Int    *int
	Type   StepRunOutType
}

func (p *CELParser) ParseStepRun(stepRunExpr string) (cel.Program, error) {
	ast, issues := p.stepRunEnv.Compile(stepRunExpr)

	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}

	return p.stepRunEnv.Program(ast)
}

func (p *CELParser) ParseAndEvalStepRun(stepRunExpr string, in Input) (*StepRunOut, error) {
	prg, err := p.ParseWorkflowString(stepRunExpr)
	if err != nil {
		return nil, err
	}

	var inMap map[string]interface{} = in

	out, _, err := prg.Eval(inMap)
	if err != nil {
		return nil, err
	}

	res := &StepRunOut{}

	switch out.Type() {
	case cel.StringType:
		str := out.Value().(string)
		res.String = &str
		res.Type = StepRunOutTypeString
	case cel.IntType:
		i := int(out.Value().(int64))
		res.Int = &i
		res.Type = StepRunOutTypeInt
	case cel.DoubleType:
		i := int(out.Value().(float64))
		res.Int = &i
		res.Type = StepRunOutTypeInt
	default:
		return nil, fmt.Errorf("output must evaluate to a string or integer: got %s", out.Type().TypeName())
	}

	return res, nil
}

func (p *CELParser) CheckStepRunOutAgainstKnown(out *StepRunOut, knownType dbsqlc.StepExpressionKind) error {
	switch knownType {
	case dbsqlc.StepExpressionKindDYNAMICRATELIMITKEY:
		if out.String == nil {
			prefix := "expected string output for dynamic rate limit key"

			if out.Int != nil {
				return fmt.Errorf("%s, got int", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case dbsqlc.StepExpressionKindDYNAMICRATELIMITVALUE:
		if out.Int == nil {
			prefix := "expected int output for dynamic rate limit value"

			if out.String != nil {
				return fmt.Errorf("%s, got string", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case dbsqlc.StepExpressionKindDYNAMICRATELIMITWINDOW:
		if out.String == nil {
			prefix := "expected string output for dynamic rate limit window"

			if out.Int != nil {
				return fmt.Errorf("%s, got int", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case dbsqlc.StepExpressionKindDYNAMICRATELIMITUNITS:
		if out.Int == nil {
			prefix := "expected int output for dynamic rate limit units"

			if out.String != nil {
				return fmt.Errorf("%s, got string", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	}

	return nil
}

func (p *CELParser) CheckStepRunOutAgainstKnownV1(out *StepRunOut, knownType sqlcv1.StepExpressionKind) error {
	switch knownType {
	case sqlcv1.StepExpressionKindDYNAMICRATELIMITKEY:
		if out.String == nil {
			prefix := "expected string output for dynamic rate limit key"

			if out.Int != nil {
				return fmt.Errorf("%s, got int", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case sqlcv1.StepExpressionKindDYNAMICRATELIMITVALUE:
		if out.Int == nil {
			prefix := "expected int output for dynamic rate limit value"

			if out.String != nil {
				return fmt.Errorf("%s, got string", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case sqlcv1.StepExpressionKindDYNAMICRATELIMITWINDOW:
		if out.String == nil {
			prefix := "expected string output for dynamic rate limit window"

			if out.Int != nil {
				return fmt.Errorf("%s, got int", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	case sqlcv1.StepExpressionKindDYNAMICRATELIMITUNITS:
		if out.Int == nil {
			prefix := "expected int output for dynamic rate limit units"

			if out.String != nil {
				return fmt.Errorf("%s, got string", prefix)
			}

			return fmt.Errorf("%s, got unknown type", prefix)
		}
	}

	return nil
}

func (p *CELParser) EvaluateEventExpression(expr, eventId string, input, additionalMetadata map[string]interface{}) (bool, error) {
	ast, issues := p.eventEnv.Compile(expr)

	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile expression: %w", issues.Err())
	}

	program, err := p.eventEnv.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create program: %w", err)
	}

	evalContext := map[string]interface{}{
		"input":               input,
		"additional_metadata": additionalMetadata,
		"event_id":            eventId,
	}

	out, _, err := program.Eval(evalContext)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate expression: %w", err)
	}

	if out.Type() != types.BoolType {
		return false, fmt.Errorf("expression did not evaluate to a boolean: got %s", out.Type().TypeName())
	}

	return out.Value().(bool), nil
}
