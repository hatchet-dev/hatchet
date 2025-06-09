import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package v1_workflows\n\nimport (\n\t\"github.com/hatchet-dev/hatchet/pkg/client/create\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/factory\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype DagInput struct {\n\tMessage string\n}\n\ntype SimpleOutput struct {\n\tStep int\n}\n\ntype DagResult struct {\n\tStep1 SimpleOutput\n\tStep2 SimpleOutput\n}\n\nfunc DagWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagInput, DagResult] {\n\t// > Declaring a Workflow\n\tsimple := factory.NewWorkflow[DagInput, DagResult](\n\t\tcreate.WorkflowCreateOpts[DagInput]{\n\t\t\tName: \"simple-dag\",\n\n\t\t},\n\t\thatchet,\n\t)\n\n\t// > Defining a Task\n\tsimple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: \"step\",\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\t// > Adding a Task with a parent\n\tstep1 := simple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: \"step-1\",\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 1,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\tsimple.Task(\n\t\tcreate.WorkflowTask[DagInput, DagResult]{\n\t\t\tName: \"step-2\",\n\t\t\tParents: []create.NamedTask{\n\t\t\t\tstep1,\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {\n\t\t\t// Get the output of the parent task\n\t\t\tvar step1Output SimpleOutput\n\t\t\terr := ctx.ParentOutput(step1, &step1Output)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &SimpleOutput{\n\t\t\t\tStep: 2,\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n",
  "source": "out/go/workflows/dag.go",
  "blocks": {
    "declaring_a_workflow": {
      "start": 26,
      "stop": 32
    },
    "defining_a_task": {
      "start": 35,
      "stop": 43
    },
    "adding_a_task_with_a_parent": {
      "start": 46,
      "stop": 74
    }
  },
  "highlights": {}
};

export default snippet;
