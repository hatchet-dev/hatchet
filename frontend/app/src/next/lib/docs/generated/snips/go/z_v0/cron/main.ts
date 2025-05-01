import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\n\t'github.com/joho/godotenv'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\t'github.com/hatchet-dev/hatchet/pkg/cmdutils'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\n// > Workflow Definition Cron Trigger\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println('called print:print')\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client and worker\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\t// 👀 define the cron expression to run every minute\n\t\t\tOn:          worker.Cron('* * * * *'),\n\t\t\tName:        'cron-workflow',\n\t\t\tDescription: 'Demonstrates a simple cron workflow',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ... start worker\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf('error cleaning up: %w', err))\n\t}\n\n\t// ,\n}\n\n",
  source: 'out/go/z_v0/cron/main.go',
  blocks: {
    workflow_definition_cron_trigger: {
      start: 15,
      stop: 84,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
