import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'errors\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype TimeoutInput struct{}\ntype TimeoutResult struct {\n\tCompleted bool\n}\n\nfunc Timeout(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[TimeoutInput, TimeoutResult] {\n\n\t// Create a task with a timeout of 3 seconds that tries to sleep for 10 seconds\n\ttimeout := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:             \'timeout-task\',\n\t\t\tExecutionTimeout: 3 * time.Second, // Task will timeout after 3 seconds\n\t\t}, func(ctx worker.HatchetContext, input TimeoutInput) (*TimeoutResult, error) {\n\t\t\t// Sleep for 10 seconds\n\t\t\ttime.Sleep(10 * time.Second)\n\n\t\t\t// Check if the context was cancelled due to timeout\n\t\t\tselect {\n\t\t\tcase <-ctx.Done():\n\t\t\t\treturn nil, errors.New(\'Task timed out\')\n\t\t\tdefault:\n\t\t\t\t// Continue execution\n\t\t\t}\n\n\t\t\treturn &TimeoutResult{\n\t\t\t\tCompleted: true,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn timeout\n}\n',
  'source': 'out/go/workflows/timeouts.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
