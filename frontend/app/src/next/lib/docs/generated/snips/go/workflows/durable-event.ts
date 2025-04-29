import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package v1_workflows\n\nimport (\n\t\'github.com/hatchet-dev/hatchet/pkg/client/create\'\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/factory\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype DurableEventInput struct {\n\tMessage string\n}\n\ntype EventData struct {\n\tMessage string\n}\n\ntype DurableEventOutput struct {\n\tData EventData\n}\n\nfunc DurableEvent(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableEventInput, DurableEventOutput] {\n\t// > Durable Event\n\tdurableEventTask := factory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'durable-event\',\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableEventInput) (*DurableEventOutput, error) {\n\t\t\teventData, err := ctx.WaitForEvent(\'user:update\', \'\')\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tv := EventData{}\n\t\t\terr = eventData.Unmarshal(&v)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableEventOutput{\n\t\t\t\tData: v,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\t\n\n\tfactory.NewDurableTask(\n\t\tcreate.StandaloneTask{\n\t\t\tName: \'durable-event\',\n\t\t},\n\t\tfunc(ctx worker.DurableHatchetContext, input DurableEventInput) (*DurableEventOutput, error) {\n\t\t\t// > Durable Event With Filter\n\t\t\teventData, err := ctx.WaitForEvent(\'user:update\', \'input.user_id == \'1234\'\')\n\t\t\t\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\tv := EventData{}\n\t\t\terr = eventData.Unmarshal(&v)\n\n\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n\n\t\t\treturn &DurableEventOutput{\n\t\t\t\tData: v,\n\t\t\t}, nil\n\t\t},\n\t\thatchet,\n\t)\n\n\treturn durableEventTask\n}\n',
  'source': 'out/go/workflows/durable-event.go',
  'blocks': {
    'durable_event': {
      'start': 25,
      'stop': 48
    },
    'durable_event_with_filter': {
      'start': 56,
      'stop': 56
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
