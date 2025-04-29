import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'fmt\'\n\t\'time\'\n\n\t\'github.com/joho/godotenv\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/client\'\n\t\'github.com/hatchet-dev/hatchet/pkg/cmdutils\'\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype Event struct {\n\tID        uint64    `json:\'id\'`\n\tCreatedAt time.Time `json:\'created_at\'`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:\'message\'`\n}\n\nfunc StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\tinput := &Event{}\n\n\terr = ctx.WorkflowInput(input)\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tfmt.Println(input.ID, \'delay\', time.Since(input.CreatedAt))\n\n\treturn &stepOneOutput{\n\t\tMessage: \'This ran at: \' + time.Now().Format(time.RubyDate),\n\t}, nil\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tclient, err := client.New()\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tw, err := worker.NewWorker(\n\t\tworker.WithClient(\n\t\t\tclient,\n\t\t),\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\terr = w.On(\n\t\tworker.Event(\'test:event\'),\n\t\t&worker.WorkflowJob{\n\t\t\tName:        \'scheduled-workflow\',\n\t\t\tDescription: \'This runs at a scheduled time.\',\n\t\t\tSteps: []*worker.WorkflowStep{\n\t\t\t\tworker.Fn(StepOne).SetName(\'step-one\'),\n\t\t\t},\n\t\t},\n\t)\n\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tch := cmdutils.InterruptChan()\n\n\tcleanup, err := w.Start()\n\tif err != nil {\n\t\tpanic(fmt.Errorf(\'error starting worker: %w\', err))\n\t}\n\n\t<-ch\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf(\'error cleaning up: %w\', err))\n\t}\n}\n',
  'source': 'out/go/z_v0/loadtest/worker/main.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
