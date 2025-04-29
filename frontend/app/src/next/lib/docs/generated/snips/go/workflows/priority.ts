import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package v1_workflows\n\nimport (\n\t'time'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client/create'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/factory'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/workflow'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\ntype PriorityInput struct {\n\tUserId string `json:'userId'`\n}\n\ntype PriorityOutput struct {\n\tTransformedMessage string `json:'TransformedMessage'`\n}\n\ntype Result struct {\n\tStep PriorityOutput\n}\n\nfunc Priority(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[PriorityInput, Result] {\n\t// Create a standalone task that transforms a message\n\n\t// > Default priority\n\tdefaultPriority := int32(1)\n\n\tworkflow := factory.NewWorkflow[PriorityInput, Result](\n\t\tcreate.WorkflowCreateOpts[PriorityInput]{\n\t\t\tName:            'priority',\n\t\t\tDefaultPriority: &defaultPriority,\n\t\t},\n\t\thatchet,\n\t)\n\t\n\n\t// > Defining a Task\n\tworkflow.Task(\n\t\tcreate.WorkflowTask[PriorityInput, Result]{\n\t\t\tName: 'step',\n\t\t}, func(ctx worker.HatchetContext, input PriorityInput) (interface{}, error) {\n\t\t\ttime.Sleep(time.Second * 5)\n\t\t\treturn &PriorityOutput{\n\t\t\t\tTransformedMessage: input.UserId,\n\t\t\t}, nil\n\t\t},\n\t)\n\t\n\treturn workflow\n}\n\n\n",
  source: 'out/go/workflows/priority.go',
  blocks: {
    default_priority: {
      start: 29,
      stop: 37,
    },
    defining_a_task: {
      start: 40,
      stop: 49,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
