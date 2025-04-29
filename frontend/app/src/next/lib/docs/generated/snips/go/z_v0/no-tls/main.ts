import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'fmt'\n\n\t'github.com/joho/godotenv'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\t'github.com/hatchet-dev/hatchet/pkg/cmdutils'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\ntype stepOutput struct{}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(fmt.Sprintf('error creating client: %v', err))\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t\tworker.WithMaxRuns(1),\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Sprintf('error creating worker: %v', err))\n\t}\n\n\ttestSvc := w.NewService('test')\n\n\terr = testSvc.On(\n\t\tworker.Events('simple'),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        'simple-workflow',\n\t\t\tDescription: 'Simple one-step workflow.',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOutput, err error) {\n\t\t\t\t\tfmt.Println('executed step 1')\n\n\t\t\t\t\treturn &stepOutput{}, nil\n\t\t\t\t},\n\t\t\t\t).SetName('step-one'),\n\t\t\t},\n\t\t},\n\t)\n\tif err != nil {\n\t\tpanic(fmt.Sprintf('error registering workflow: %v', err))\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Sprintf('error starting worker: %v', err))\n\t}\n\n\t<-interruptCtx.Done()\n\tif err := cleanup(); err != nil {\n\t\tpanic(err)\n\t}\n}\n",
  source: 'out/go/z_v0/no-tls/main.go',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
