import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'fmt\'\n\t\'log\'\n\n\t\'github.com/joho/godotenv\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:\'username\'`\n\tUserID   string            `json:\'user_id\'`\n\tData     map[string]string `json:\'data\'`\n}\n\ntype output struct {\n\tMessage string `json:\'message\'`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\'error creating client: %w\', err))\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\'error creating worker: %w\', err))\n\t}\n\n\tworkflow := \'webhook\'\n\tevent := \'user:create:webhook\'\n\twf := &worker.WorkflowJob{\n\t\tName:        workflow,\n\t\tDescription: workflow,\n\t\tSteps: []*worker.WorkflowStep{\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {\n\t\t\t\tlog.Printf(\'step name: %s\', ctx.StepName())\n\t\t\t\treturn &output{\n\t\t\t\t\tMessage: \'hi from \' + ctx.StepName(),\n\t\t\t\t}, nil\n\t\t\t}).SetName(\'webhook-step-one\').SetTimeout(\'10s\'),\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {\n\t\t\t\tlog.Printf(\'step name: %s\', ctx.StepName())\n\t\t\t\treturn &output{\n\t\t\t\t\tMessage: \'hi from \' + ctx.StepName(),\n\t\t\t\t}, nil\n\t\t\t}).SetName(\'webhook-step-one\').SetTimeout(\'10s\'),\n\t\t},\n\t}\n\n\thandler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{\n\t\tSecret: \'secret\',\n\t}, wf)\n\tport := \'8741\'\n\terr = run(\'webhook-demo\', w, port, handler, c, workflow, event)\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n',
  'source': 'out/go/z_v0/webhook/main.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
