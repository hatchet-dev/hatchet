import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package main\n\nimport (\n\t\'fmt\'\n\t\'time\'\n\n\t\'github.com/joho/godotenv\'\n\n\t\'github.com/hatchet-dev/hatchet/pkg/worker\'\n)\n\ntype userCreateEvent struct {\n\tUsername string            `json:\'username\'`\n\tUserID   string            `json:\'user_id\'`\n\tData     map[string]string `json:\'data\'`\n}\n\ntype stepOneOutput struct {\n\tMessage string `json:\'message\'`\n}\n\nfunc main() {\n\terr := godotenv.Load()\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\tevents := make(chan string, 50)\n\n\t// > TimeoutStep\n\tcleanup, err := run(events, worker.WorkflowJob{\n\t\tName:        \'timeout\',\n\t\tDescription: \'timeout\',\n\t\tSteps: []*worker.WorkflowStep{\n\t\t\tworker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {\n\t\t\t\ttime.Sleep(time.Second * 60)\n\t\t\t\treturn nil, nil\n\t\t\t}).SetName(\'step-one\').SetTimeout(\'10s\'),\n\t\t},\n\t})\n\t\n\tif err != nil {\n\t\tpanic(err)\n\t}\n\n\t<-events\n\n\tif err := cleanup(); err != nil {\n\t\tpanic(fmt.Errorf(\'cleanup() error = %v\', err))\n\t}\n}\n',
  'source': 'out/go/z_v0/timeout/main.go',
  'blocks': {
    'timeoutstep': {
      'start': 31,
      'stop': 40
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
