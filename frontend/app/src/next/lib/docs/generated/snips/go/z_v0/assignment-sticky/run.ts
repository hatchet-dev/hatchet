import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'go',
  content:
    "package main\n\nimport (\n\t'context'\n\t'fmt'\n\t'log'\n\t'time'\n\n\t'github.com/hatchet-dev/hatchet/pkg/client'\n\t'github.com/hatchet-dev/hatchet/pkg/client/types'\n\t'github.com/hatchet-dev/hatchet/pkg/worker'\n)\n\nfunc run() (func() error, error) {\n\tc, err := client.New()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf('error creating client: %w', err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf('error creating worker: %w', err)\n\t}\n\n\t// > StickyWorker\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.Events('user:create:sticky'),\n\t\t\tName:        'sticky',\n\t\t\tDescription: 'sticky',\n\t\t\t// 👀 Specify a sticky strategy when declaring the workflow\n\t\t\tStickyStrategy: types.StickyStrategyPtr(types.StickyStrategy_HARD),\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\n\t\t\t\t\tsticky := true\n\n\t\t\t\t\t_, err = ctx.SpawnWorkflow('sticky-child', nil, &worker.SpawnWorkflowOpts{\n\t\t\t\t\t\tSticky: &sticky,\n\t\t\t\t\t})\n\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn nil, fmt.Errorf('error spawning workflow: %w', err)\n\t\t\t\t\t}\n\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName('step-one'),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName('step-two').AddParents('step-one'),\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName('step-three').AddParents('step-two'),\n\t\t\t},\n\t\t},\n\t)\n\n\t\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf('error registering workflow: %w', err)\n\t}\n\n\t// > StickyChild\n\n\terr = w.RegisterWorkflow(\n\t\t&worker.WorkflowJob{\n\t\t\tOn:          worker.NoTrigger(),\n\t\t\tName:        'sticky-child',\n\t\t\tDescription: 'sticky',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\t\treturn &stepOneOutput{\n\t\t\t\t\t\tMessage: ctx.Worker().ID(),\n\t\t\t\t\t}, nil\n\t\t\t\t}).SetName('step-one'),\n\t\t\t},\n\t\t},\n\t)\n\n\t\n\n\tif err != nil {\n\t\treturn nil, fmt.Errorf('error registering workflow: %w', err)\n\t}\n\n\tgo func() {\n\t\tlog.Printf('pushing event')\n\n\t\ttestEvent := userCreateEvent{\n\t\t\tUsername: 'echo-test',\n\t\t\tUserID:   '1234',\n\t\t\tData: map[string]string{\n\t\t\t\t'test': 'test',\n\t\t\t},\n\t\t}\n\n\t\t// push an event\n\t\terr := c.Event().Push(\n\t\t\tcontext.Background(),\n\t\t\t'user:create:sticky',\n\t\t\ttestEvent,\n\t\t)\n\t\tif err != nil {\n\t\t\tpanic(fmt.Errorf('error pushing event: %w', err))\n\t\t}\n\n\t\ttime.Sleep(10 * time.Second)\n\t}()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\treturn nil, fmt.Errorf('error starting worker: %w', err)\n\t}\n\n\treturn cleanup, nil\n}\n",
  source: 'out/go/z_v0/assignment-sticky/run.go',
  blocks: {
    stickyworker: {
      start: 30,
      stop: 68,
    },
    stickychild: {
      start: 75,
      stop: 90,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
