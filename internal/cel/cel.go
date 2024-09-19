package cel

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

type CELParser struct {
	workflowStrEnv *cel.Env
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

	return &CELParser{
		workflowStrEnv: workflowStrEnv,
	}
}

type WorkflowStringInput map[string]interface{}

type WorkflowStringInputOpts func(WorkflowStringInput)

func WithInput(input map[string]interface{}) WorkflowStringInputOpts {
	return func(w WorkflowStringInput) {
		w["input"] = input
	}
}

func WithAdditionalMetadata(metadata map[string]interface{}) WorkflowStringInputOpts {
	return func(w WorkflowStringInput) {
		w["additional_metadata"] = metadata
	}
}

func WithWorkflowRunID(workflowRunID string) WorkflowStringInputOpts {
	return func(w WorkflowStringInput) {
		w["workflow_run_id"] = workflowRunID
	}
}

func NewWorkflowStringInput(opts ...WorkflowStringInputOpts) WorkflowStringInput {
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

func (p *CELParser) ParseAndEvalWorkflowString(workflowExp string, in WorkflowStringInput) (string, error) {
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
