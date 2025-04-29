import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'math/rand\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\t\'github.com/hatchet-dev/hatchet/pkg/client/types\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype ConcurrencyInput struct {\n\tMessage string\n\tTier    string\n\tAccount string\n}\n\ntype TransformedOutput struct {\n\tTransformedMessage string\n}\n\nfunc ConcurrencyRoundRobin(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {\n\t// > Concurrency Strategy With Key\n\tvar maxRuns int32 = 1\n\tstrategy := types.GroupRoundRobin\n\n\tconcurrency := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'simple-concurrency\',\n\t\t\tConcurrency: []*types.Concurrency{\n\t\t\t\t{\n\t\t\t\t\tExpression:    \'input.GroupKey\',\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {\n\t\t\t// Random sleep between 200ms and 1000ms\n\t\t\ttime.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)\n\n\t\t\treturn &TransformedOutput{\n\t\t\t\tTransformedMessage: input.Message,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t\n\n\treturn concurrency\n}\n\nfunc MultipleConcurrencyKeys(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {\n\t// > Multiple Concurrency Keys\n\tstrategy := types.GroupRoundRobin\n\tvar maxRuns int32 = 20\n\n\tconcurrency := factory.NewTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'simple-concurrency\',\n\t\t\tConcurrency: []*types.Concurrency{\n\t\t\t\t{\n\t\t\t\t\tExpression:    \'input.Tier\',\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t\t{\n\t\t\t\t\tExpression:    \'input.Account\',\n\t\t\t\t\tMaxRuns:       &maxRuns,\n\t\t\t\t\tLimitStrategy: &strategy,\n\t\t\t\t},\n\t\t\t},\n\t\t}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {\n\t\t\t// Random sleep between 200ms and 1000ms\n\t\t\ttime.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)\n\n\t\t\treturn &TransformedOutput{\n\t\t\t\tTransformedMessage: input.Message,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t\n\n\treturn concurrency\n}\n',
  'source': 'out/go/workflows/concurrency-rr.go',
  'blocks': {
    'concurrency_strategy_with_key': {
      'start': 27,
      'stop': 49
    },
    'multiple_concurrency_keys': {
      'start': 56,
      'stop': 83
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
