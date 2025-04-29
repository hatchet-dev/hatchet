import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\thatchet_client \'hatchet-go-quickstart/hatchet_client\'\n\tworkflows \'hatchet-go-quickstart/workflows\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/cmdutils\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/worker\'\n\t\'github.com/hatchet-dev/hatchet/pkg/v1/workflow\'\n)\n\nfunc main() {\n\n\thatchet, err := hatchet_client.HatchetClient()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tworker, err := hatchet.Worker(\n\t\tworker.WorkerOpts{\n\t\t\tName: \'first-worker\',\n\t\t\tWorkflows: []workflow.WorkflowBase{\n\t\t\t\tworkflows.FirstTask(hatchet),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// we construct an interrupt context to handle Ctrl+C\n\t// you can pass in your own context.Context here to the worker\n\tinterruptCtx, cancel := cmdutils.NewInterruptContext()\n\n\tdefer cancel()\n\n\terr = worker.StartBlocking(interruptCtx)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
  'source': 'out/go/quickstart/cmd/worker/main.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
