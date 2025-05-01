import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\n\t'github.com/joho/godotenv'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\t'github.com/hatchet-dev/hatchet/pkg/cmdutils'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\n// > Create\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println('called print:print')\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client, worker and workflow\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        'cron-workflow',\n\t\t\tDescription: 'Demonstrates a simple cron workflow',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\tgo func() {\n\t\t// 👀 define the cron expression to run every minute\n\t\tcron, err := c.Cron().Create(\n\t\t\tcontext.Background(),\n\t\t\t'cron-workflow',\n\t\t\t&client.CronOpts{\n\t\t\t\tName:       'every-minute',\n\t\t\t\tExpression: '* * * * *',\n\t\t\t\tInput: map[string]interface{}{\n\t\t\t\t\t'message': 'Hello, world!',\n\t\t\t\t},\n\t\t\t\tAdditionalMetadata: map[string]string{},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\n\t\tfmt.Println(*cron.Name, cron.Cron)\n\t}()\n\n\t// ... wait for interrupt signal\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf('error cleaning up: %w', err))\n\t}\n\n\t// ,\n}\n\n\nfunc ListCrons() {\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > List\n\tcrons, err := c.Cron().List(context.Background())\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor _, cron := range *crons.Rows {\n\t\tfmt.Println(cron.Cron, *cron.Name)\n\t}\n}\n\nfunc DeleteCron(id string) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > Delete\n\t// 👀 id is the cron's metadata id, can get it via cron.Metadata.Id\n\terr = c.Cron().Delete(context.Background(), id)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n}\n",
  source: 'out/go/z_v0/cron-programmatic/main.go',
  blocks: {
    create: {
      start: 15,
      stop: 106,
    },
    list: {
      start: 117,
      stop: 117,
    },
    delete: {
      start: 136,
      stop: 137,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
