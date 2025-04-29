import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'strings\'\n\t\'time\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype DurableSleepInput struct {\n\tMessage string\n}\n\ntype DurableSleepOutput struct {\n\tTransformedMessage string\n}\n\nfunc DurableSleep(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableSleepInput, DurableSleepOutput] {\n\t// > Durable Sleep\n\tsimple := factory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'durable-sleep\',\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableSleepInput) (*DurableSleepOutput, error) {\n\t\t\t_, err := ctx.SleepFor(10 * time.Second)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableSleepOutput{\n\t\t\t\tTransformedMessage: strings.ToLower(input.Message),\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn simple\n}\n',
  'source': 'out/go/workflows/durable-sleep.go',
  'blocks': {
    'durable_sleep': {
      'start': 24,
      'stop': 40
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
