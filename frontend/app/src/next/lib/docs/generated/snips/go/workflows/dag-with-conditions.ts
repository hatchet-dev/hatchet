import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package v1_workflows\n\nimport (\n\t'fmt'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client/create'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/factory'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/workflow'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\ntype DagWithConditionsInput struct {\n\tMessage string\n}\n\ntype DagWithConditionsResult struct {\n\tStep1 SimpleOutput\n\tStep2 SimpleOutput\n}\n\ntype conditionOpts = create.WorkflowTask[DagWithConditionsInput, DagWithConditionsResult]\n\nfunc DagWithConditionsWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagWithConditionsInput, DagWithConditionsResult] {\n\n\tsimple := factory.NewWorkflow[DagWithConditionsInput, DagWithConditionsResult](\n\t\tcreate.WorkflowCreateOpts[DagWithConditionsInput]{\n\t\t\tName: 'simple-dag',\n\t\t},\n\t\thatchet,\n\t)\n\n\tstep1 := simple.Task(\n\t\tconditionOpts{\n\t\t\tName: 'Step1',\n\t\t}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\tsimple.Task(\n\t\tconditionOpts{\n\t\t\tName: 'Step2',\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstep1,\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input DagWithConditionsInput) (interface{}, error) {\n\n\t\t\tvar step1Output SimpleOutput\n\t\t\terr := ctx.ParentOutput(step1, &step1Output)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tfmt.Println(step1Output.Step)\n\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 2,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n",
  source: 'out/go/workflows/dag-with-conditions.go',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
