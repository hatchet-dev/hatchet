import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'fmt\'\n\t\'time\'\n\n\t\'github.com/joho/godotenv\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/cmdutils\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype stepOneOutput struct {\n\tMessage string `json:\'message\'`\n}\n\n// > OnFailure Step\n// This workflow will fail because the step will throw an error\n// we define an onFailure step to handle this case\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t// 👀 this step will always raise an exception\n\treturn nil, fmt.Errorf(\'test on failure\')\n}\n\nfunc OnFailure(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t// run cleanup code or notifications here\n\n\t// 👀 you can access the error from the failed step(s) like this\n\tfmt.Println(ctx.StepRunErrors())\n\n\treturn &stepOneOutput{\n\t\tMessage: \'Failure!\',\n\t}, nil\n}\n\nfunc main() {\n\t// ...\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tc, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tc,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t// 👀 we define an onFailure step to handle this case\n\terr = w.On(\n\t\tworker.NoTrigger(),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        \'on-failure-workflow\',\n\t\t\tDescription: \'This runs at a scheduled time.\',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName(\'step-one\'),\n\t\t\t},\n\t\t\tOnFailure: &worker.WorkflowJob{\n\t\t\t\tName:        \'scheduled-workflow-failure\',\n\t\t\t\tDescription: \'This runs when the scheduled workflow fails.\',\n\t\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\t\tworker.Fn(OnFailure).SetName(\'on-failure\'),\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t)\n\n\t// ...\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tinterruptCtx, cancel := cmdutils.InterruptContextFromChan(cmdutils.InterruptChan())\n\tdefer cancel()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\'error cleaning up: %w\', err))\n\t}\n\n\tfor {\n\t\tselect {\n\t\tcase <-interruptCtx.Done():\n\t\t\tif err := cleanup(); err != nil {\n\t\t\t\tpanic(fmt.Errorf(\'error cleaning up: %w\', err))\n\t\t\t}\n\t\t\treturn\n\t\tdefault:\n\t\t\ttime.Sleep(time.Second)\n\t\t}\n\t}\n\t// ,\n}\n\n\n',
  'source': 'out/go/z_v0/on-failure/main.go',
  'blocks': {
    'onfailure_step': {
      'start': 19,
      'stop': 108
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
