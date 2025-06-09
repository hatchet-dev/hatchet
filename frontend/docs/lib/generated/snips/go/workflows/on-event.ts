import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "go",
  "content": "package v1_workflows\n\nimport (\n\t\"strings\"\n\n\t\"github.com/hatchet-dev/hatchet/pkg/client/create\"\n\t\"github.com/hatchet-dev/hatchet/pkg/client/types\"\n\tv1 \"github.com/hatchet-dev/hatchet/pkg/v1\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/factory\"\n\t\"github.com/hatchet-dev/hatchet/pkg/v1/workflow\"\n\t\"github.com/hatchet-dev/hatchet/pkg/worker\"\n)\n\ntype EventInput struct {\n\tMessage string\n}\n\ntype LowerTaskOutput struct {\n\tTransformedMessage string\n}\n\ntype UpperTaskOutput struct {\n\tTransformedMessage string\n}\n\n// > Run workflow on event\nconst SimpleEvent = \"simple-event:create\"\n\nfunc Lower(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \"lower\",\n\t\t\t// 👀 Declare the event that will trigger the workflow\n\t\t\tOnEvents: []string{SimpleEvent},\n\t\t}, func(ctx worker.HatchetContext, input EventInput) (*LowerTaskOutput, error) {\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &LowerTaskOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n}\n\n\n// > Declare with filter\nfunc LowerWithFilter(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, LowerTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \"lower\",\n\t\t\t// 👀 Declare the event that will trigger the workflow\n\t\t\tOnEvents: []string{SimpleEvent},\n\t\t\tDefaultFilters: []types.DefaultFilter{{\n\t\t\t\tExpression: \"true\",\n\t\t\t\tScope:      \"example-scope\",\n\t\t\t\tPayload: map[string]interface{}{\n\t\t\t\t\t\"example_key\": \"example_value\",\n\t\t\t\t},\n\t\t\t}},\n\t\t}, func(ctx worker.HatchetContext, input EventInput) (*LowerTaskOutput, error) {\n\t\t\t// Transform the input message to lowercase\n\t\t\treturn &LowerTaskOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n}\n\n\nfunc Upper(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[EventInput, UpperTaskOutput] {\n\treturn factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:     \"upper\",\n\t\t\tOnEvents: []string{SimpleEvent},\n\t\t},\n\t\tfunc(ctx worker.HatchetContext, input EventInput) (*UpperTaskOutput, error) {\n\t\t\treturn &UpperTaskOutput{\n\t\t\t\tTransformedMessage: strings.ToUpper(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n}\n",
  "source": "out/go/workflows/on-event.go",
  "blocks": {
    "run_workflow_on_event": {
      "start": 27,
      "stop": 44
    },
    "declare_with_filter": {
      "start": 47,
      "stop": 69
    }
  },
  "highlights": {}
};

export default snippet;
