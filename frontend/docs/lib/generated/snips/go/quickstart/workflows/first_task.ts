import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package workflows\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client/create\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/factory\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype SimpleInput struct {\n\tMessage string `json:\"message\"`\n}\n\ntype LowerOutput struct {\n\tTransformedMessage string `json:\"transformed_message\"`\n}\n\ntype SimpleResult struct {\n\tToLower LowerOutput\n}\n\nfunc FirstTask(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[SimpleInput, SimpleResult] {\n\tsimple := factory.NewWorkflow[SimpleInput, SimpleResult](\n\t\tcreate.WorkflowCreateOpts[SimpleInput]{\n\t\t\tName: \"first-task\",\n\t\t},\n\t\thatchet,\n\t)\n\n\tsimple.Task(\n\t\tcreate.WorkflowTask[SimpleInput, SimpleResult]{\n\t\t\tName: \"first-task\",\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input SimpleInput) (any, error) {\n\t\t\tfmt.Println(\"first-task task called\")\n\t\t\treturn &LowerOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t)\n\n\treturn simple\n}\n",
  "source": "out/go/quickstart/workflows/first_task.go",
  "blocks": {},
  "highlights": {}
};

export default snippet;
