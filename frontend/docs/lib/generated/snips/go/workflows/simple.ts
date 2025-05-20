import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package v1_workflows\n\nimport (\n\t\"context\"\n\t\"fmt\"\n\t\"strings\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client/create\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/factory\"\n\tv1worker \"github.com/hatchet-dev/hatchet/pkg/v1/worker\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype SimpleInput struct {\n\tMessage string\n}\ntype SimpleResult struct {\n\tTransformedMessage string\n}\n\nfunc Simple(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {\n\n\t// Create a simple standalone task using the task factory\n\t// Note the use of typed generics for both input and output\n\n\t// > Declaring a Task\n\tsimple := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \"simple-task\",\n\t\t}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &SimpleResult{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\t// Example of running a task\n\t_ = func() error {\n\t\t// > Running a Task\n\t\tresult, err := simple.Run(context.Background(), SimpleInput{Message: \"Hello, World!\"})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tfmt.Println(result.TransformedMessage)\n\t\treturn nil\n\t}\n\n\t// Example of registering a task on a worker\n\t_ = func() error {\n\t\t// > Declaring a Worker\n\t\tw, err := hatchet.Worker(v1worker.WorkerOpts{\n\t\t\tName: \"simple-worker\",\n\t\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\t\tsimple,\n\t\t\t},\n\t\t})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\terr = w.StartBlocking(context.Background())\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\treturn nil\n\t}\n\n\treturn simple\n}\n\nfunc ParentTask(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {\n\n\t// > Spawning Tasks from within a Task\n\tsimple := Simple(hatchet)\n\n\tparent := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \"parent-task\",\n\t\t}, func(ctx worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {\n\n\t\t\t// Run the child task\n\t\t\tchild, err := workflow.RunChildWorkflow(ctx, simple, SimpleInput{Message: input.Message})\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &SimpleResult{\n\t\t\t\tTransformedMessage: child.TransformedMessage,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn parent\n}\n",
  "source": "out/go/workflows/simple.go",
  "blocks": {
    "declaring_a_task": {
      "start": 29,
      "stop": 39
    },
    "running_a_task": {
      "start": 44,
      "stop": 48
    },
    "declaring_a_worker": {
      "start": 55,
      "stop": 67
    },
    "spawning_tasks_from_within_a_task": {
      "start": 77,
      "stop": 96
    }
  },
  "highlights": {}
};

export default snippet;
