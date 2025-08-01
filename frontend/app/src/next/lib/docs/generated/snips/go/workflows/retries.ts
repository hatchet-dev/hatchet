import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    'package v1_workflows\n\nimport (\n\t"errors"\n\t"fmt"\n\n\t"github.com/hatchet-dev/hatchet/pkg/client/create"\n\tv1 "github.com/hatchet-dev/hatchet/pkg/v1"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/factory"\n\t"github.com/hatchet-dev/hatchet/pkg/v1/workflow"\n\t"github.com/hatchet-dev/hatchet/pkg/worker"\n)\n\ntype (\n\tRetriesInput  struct{}\n\tRetriesResult struct{}\n)\n\n// Simple retries example that always fails\nfunc Retries(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesInput, RetriesResult] {\n\t// > Simple Step Retries\n\tretries := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "retries-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input RetriesInput) (*RetriesResult, error) {\n\t\t\treturn nil, errors.New("intentional failure")\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn retries\n}\n\ntype (\n\tRetriesWithCountInput  struct{}\n\tRetriesWithCountResult struct {\n\t\tMessage string `json:"message"`\n\t}\n)\n\n// Retries example that succeeds after a certain number of retries\nfunc RetriesWithCount(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesWithCountInput, RetriesWithCountResult] {\n\t// > Retries with Count\n\tretriesWithCount := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName:    "fail-twice-task",\n\t\t\tRetries: 3,\n\t\t}, func(ctx worker.HatchetContext, input RetriesWithCountInput) (*RetriesWithCountResult, error) {\n\t\t\t// Get the current retry count\n\t\t\tretryCount := ctx.RetryCount()\n\n\t\t\tfmt.Printf("Retry count: %d\\n", retryCount)\n\n\t\t\tif retryCount < 2 {\n\t\t\t\treturn nil, errors.New("intentional failure")\n\t\t\t}\n\n\t\t\treturn &RetriesWithCountResult{\n\t\t\t\tMessage: "success",\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn retriesWithCount\n}\n\ntype (\n\tBackoffInput  struct{}\n\tBackoffResult struct{}\n)\n\n// Retries example with simple backoff (no configuration in this API version)\nfunc WithBackoff(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[BackoffInput, BackoffResult] {\n\t// > Retries with Backoff\n\twithBackoff := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: "with-backoff-task",\n\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\tRetries: 3,\n\t\t\t// 👀 Factor to increase the wait time between retries.\n\t\t\tRetryBackoffFactor: 2,\n\t\t\t// 👀 Maximum number of seconds to wait between retries\n\t\t\t// This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n\t\t\tRetryMaxBackoffSeconds: 10,\n\t\t}, func(ctx worker.HatchetContext, input BackoffInput) (*BackoffResult, error) {\n\t\t\treturn nil, errors.New("intentional failure")\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn withBackoff\n}\n',
  source: 'out/go/workflows/retries.go',
  blocks: {
    simple_step_retries: {
      start: 22,
      stop: 30,
    },
    retries_with_count: {
      start: 45,
      stop: 64,
    },
    retries_with_backoff: {
      start: 77,
      stop: 91,
    },
  },
  highlights: {},
};

export default snippet;
