import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package v1_workflows\n\nimport (\n\t'errors'\n\t'time'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client/create'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/factory'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/workflow'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\ntype CancellationInput struct{}\ntype CancellationResult struct {\n\tCompleted bool\n}\n\nfunc Cancellation(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[CancellationInput, CancellationResult] {\n\n\t// > Cancelled task\n\t// Create a task that sleeps for 10 seconds and checks if it was cancelled\n\tcancellation := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: 'cancellation-task',\n\t\t}, func(ctx worker.HatchetContext, input CancellationInput) (*CancellationResult, error) {\n\t\t\t// Sleep for 10 seconds\n\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t// Check if the context was cancelled\n\t\t\tselect {\n\t\t\tcase <-ctx.Done():\n\t\t\t\treturn nil, errors.New('Task was cancelled')\n\t\t\tdefault:\n\t\t\t\t// Continue execution\n\t\t\t}\n\n\t\t\treturn &CancellationResult{\n\t\t\t\tCompleted: true,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn cancellation\n}\n",
  source: 'out/go/workflows/cancellations.go',
  blocks: {
    cancelled_task: {
      start: 22,
      stop: 43,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
