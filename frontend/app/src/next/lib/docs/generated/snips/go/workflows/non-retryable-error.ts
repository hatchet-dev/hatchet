import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package v1_workflows\n\nimport (\n\t'errors'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client/create'\n\tv1 'github.com/hatchet-dev/hatchet/pkg/v1'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/factory'\n\t'github.com/hatchet-dev/hatchet/pkg/v1/workflow'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\ntype NonRetryableInput struct{}\ntype NonRetryableResult struct{}\n\n// NonRetryableError returns a workflow which throws a non-retryable error\nfunc NonRetryableError(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[NonRetryableInput, NonRetryableResult] {\n\t// > Non Retryable Error\n\tretries := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    'non-retryable-task',\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input NonRetryableInput) (*NonRetryableResult, error) {\n\t\t\treturn nil, worker.NewNonRetryableError(errors.New('intentional failure'))\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn retries\n}\n",
  source: 'out/go/workflows/non-retryable-error.go',
  blocks: {
    non_retryable_error: {
      start: 19,
      stop: 27,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
