import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\t'time'\n\n\t'github.com/joho/godotenv'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\t'github.com/hatchet-dev/hatchet/pkg/cmdutils'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\n// > Create\n// ... normal workflow definition\ntype printOutput struct{}\n\nfunc print(ctx context.Context) (result *printOutput, err error) {\n\tfmt.Println('called print:print')\n\n\treturn &printOutput{}, nil\n}\n\n// ,\nfunc main() {\n\t// ... initialize client, worker and workflow\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        'schedule-workflow',\n\t\t\tDescription: 'Demonstrates a simple scheduled workflow',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(print),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterrupt := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// ,\n\n\tgo func() {\n\t\t// 👀 define the scheduled workflow to run in a minute\n\t\tschedule, err := c.Schedule().Create(\n\t\t\tcontext.Background(),\n\t\t\t'schedule-workflow',\n\t\t\t&client.ScheduleOpts{\n\t\t\t\t// 👀 define the time to run the scheduled workflow, in UTC\n\t\t\t\tTriggerAt: time.Now().UTC().Add(time.Minute),\n\t\t\t\tInput: map[string]interface{}{\n\t\t\t\t\t'message': 'Hello, world!',\n\t\t\t\t},\n\t\t\t\tAdditionalMetadata: map[string]string{},\n\t\t\t},\n\t\t)\n\n\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n\n\t\tfmt.Println(schedule.TriggerAt, schedule.WorkflowName)\n\t}()\n\n\t// ... wait for interrupt signal\n\n\t<-interrupt\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf('error cleaning up: %w', err))\n\t}\n\n\t// ,\n}\n\n\n\nfunc ListScheduledWorkflows() {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > List\n\tschedules, err := c.Schedule().List(context.Background())\n\t\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tfor _, schedule := range *schedules.Rows {\n\t\tfmt.Println(schedule.TriggerAt, schedule.WorkflowName)\n\t}\n}\n\nfunc DeleteScheduledWorkflow(id string) {\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// > Delete\n\t// 👀 id is the schedule's metadata id, can get it via schedule.Metadata.Id\n\terr = c.Schedule().Delete(context.Background(), id)\n\t\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n}\n",
  source: 'out/go/z_v0/scheduled/main.go',
  blocks: {
    create: {
      start: 16,
      stop: 107,
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
